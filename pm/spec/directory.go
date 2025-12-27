package spec

import (
	"bytes"
	"encoding/binary"
	"math"
	"slices"
	"sort"
)

type Entry struct {
	TileCode  uint64 // spec v3: TileID
	Offset    uint64
	Length    uint32
	RunLength uint32
}

func SerializeDirectory(entries []Entry) []byte {
	buffer := make([]byte, 0)

	buffer = binary.AppendUvarint(buffer, uint64(len(entries)))

	lastCode := uint64(0)
	for _, entry := range entries {
		buffer = binary.AppendUvarint(buffer, uint64(entry.TileCode)-lastCode)
		lastCode = uint64(entry.TileCode)
	}

	for _, entry := range entries {
		buffer = binary.AppendUvarint(buffer, uint64(entry.RunLength))
	}

	for _, entry := range entries {
		buffer = binary.AppendUvarint(buffer, uint64(entry.Length))
	}

	nextOffset := uint64(0)
	for i, entry := range entries {
		if i > 0 && entry.Offset == nextOffset {
			buffer = binary.AppendUvarint(buffer, 0)
		} else {
			buffer = binary.AppendUvarint(buffer, uint64(entry.Offset)+1)
		}
		nextOffset = entry.Offset + uint64(entry.Length)
	}

	return buffer
}

func DeserializeDirectory(data []byte) ([]Entry, error) {
	byteReader := bytes.NewReader(data)

	var err error
	readUvarint := func() uint64 {
		if err != nil {
			return 0
		}
		var value uint64
		value, err = binary.ReadUvarint(byteReader)
		return value
	}

	numEntries := readUvarint()
	entries := make([]Entry, numEntries)

	lastCode := uint64(0)
	for i := range numEntries {
		value := readUvarint()
		entries[i].TileCode = lastCode + value
		lastCode += value
	}

	for i := range numEntries {
		entries[i].RunLength = uint32(readUvarint())
	}

	for i := range numEntries {
		entries[i].Length = uint32(readUvarint())
	}

	for i := range numEntries {
		value := readUvarint()
		if value == 0 && i > 0 {
			entries[i].Offset = entries[i-1].Offset + uint64(entries[i-1].Length)
		} else {
			entries[i].Offset = value - 1
		}
	}

	return entries, err
}

func CompactEntries(entries []Entry) []Entry {
	if len(entries) == 0 {
		return entries
	}
	wi := 0
	for ri := 1; ri < len(entries); ri++ {
		if entries[ri].Offset == entries[wi].Offset &&
			entries[ri].TileCode == entries[wi].TileCode+uint64(entries[wi].RunLength) {
			entries[wi].RunLength++
		} else {
			wi++
			entries[wi] = entries[ri]
		}
	}
	return entries[:wi+1]
}

func FindEntry(entries []Entry, tileCode uint64) (Entry, bool) {
	idx := sort.Search(len(entries), func(i int) bool {
		return entries[i].TileCode > tileCode
	})

	if idx == 0 {
		return Entry{}, false
	}

	entry := &entries[idx-1]
	if entry.RunLength == 0 {
		// should continue search in leaf directory
		return *entry, true
	}
	if tileCode < entry.TileCode+uint64(entry.RunLength) {
		// found in root directory
		return *entry, true
	}

	return Entry{}, false
}

func SerializeAll(entries []Entry, compression Compression) ([]byte, []byte) {
	rootEntries := entries
	rootData := SerializeDirectory(rootEntries)
	rootCompressed, _ := Compress(rootData, compression)
	leavesCompressed := make([]byte, 0)

	if len(entries) == 0 {
		return rootCompressed, leavesCompressed
	}

	entriesCount := float64(len(entries))
	entriesSize := float64(len(rootCompressed))
	entrySize := entriesSize / entriesCount
	targetRootSize := float64(RootDirMaxLength) * 0.9

	maxRootEntries := targetRootSize / entrySize
	minLeafEntries := max(entriesCount/maxRootEntries, 4096)
	leafNumEntries := max(minLeafEntries, math.Sqrt(entriesCount))

	for len(rootCompressed) > RootDirMaxLength {
		rootEntries = rootEntries[:0]
		leavesCompressed = leavesCompressed[:0]

		for leafEntries := range slices.Chunk(entries, int(leafNumEntries)) {
			leafData := SerializeDirectory(leafEntries)
			leafCompressed, _ := Compress(leafData, compression)

			rootEntries = append(rootEntries, Entry{
				TileCode:  leafEntries[0].TileCode,
				Offset:    uint64(len(leavesCompressed)),
				Length:    uint32(len(leafCompressed)),
				RunLength: 0,
			})

			leavesCompressed = append(leavesCompressed, leafCompressed...)
		}

		rootData = SerializeDirectory(rootEntries)
		rootCompressed, _ = Compress(rootData, compression)

		leafNumEntries *= 1.1
	}

	return rootCompressed, leavesCompressed
}
