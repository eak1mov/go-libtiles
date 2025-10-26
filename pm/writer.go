package pm

import (
	"bufio"
	"cmp"
	"crypto/md5"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/eak1mov/go-libtiles/pm/spec"
)

type Writer interface {
	io.Closer

	WriteTile(tileId TileId, tileData []byte) error
	Finalize() error
}

type WriterParams struct {
	Metadata       []byte
	HeaderMetadata HeaderMetadata
	Logger         *slog.Logger
}

type writer struct {
	logger *slog.Logger
	file   *os.File
	header spec.Header

	tileWriter *bufio.Writer
	tileOffset uint64

	entries   []spec.Entry
	locations map[[16]byte]uint32 // hash -> entry index
}

func NewWriter(filePath string) (Writer, error) {
	return NewWriterParams(filePath, WriterParams{})
}

func NewWriterParams(filePath string, params WriterParams) (w Writer, err error) {
	logger := params.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			file.Close()
		}
	}()

	header := spec.Header{}
	offset := uint64(spec.HeaderRootDirMaxLength)

	_, err = file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	if params.Metadata != nil {
		_, err := file.Write(params.Metadata)
		if err != nil {
			return nil, err
		}
		header.MetadataOffset = offset
		header.MetadataLength = uint64(len(params.Metadata))
		offset += header.MetadataLength
	}

	header.HeaderMagic = spec.HeaderMagicV3
	header.Clustered = true
	header.InternalCompression = spec.CompressionGzip
	header.TileDataOffset = offset
	params.HeaderMetadata.CopyToHeader(&header)

	return &writer{
		logger:     logger,
		file:       file,
		header:     header,
		tileWriter: bufio.NewWriter(file),
		tileOffset: 0,
		locations:  make(map[[16]byte]uint32),
	}, nil
}

func (w *writer) WriteTile(tileId TileId, tileData []byte) error {
	if len(tileData) == 0 {
		return nil
	}

	digest := md5.Sum(tileData)
	entryIdx, exists := w.locations[digest]

	if exists {
		entry := spec.Entry{
			TileCode:  spec.EncodeTileId(tileId),
			Offset:    w.entries[entryIdx].Offset,
			Length:    w.entries[entryIdx].Length,
			RunLength: 1,
		}
		w.entries = append(w.entries, entry)
		return nil
	}

	entry := spec.Entry{
		TileCode:  spec.EncodeTileId(tileId),
		Offset:    w.tileOffset,
		Length:    uint32(len(tileData)),
		RunLength: 1,
	}

	_, err := w.tileWriter.Write(tileData)
	if err != nil {
		return err
	}

	w.tileOffset += uint64(len(tileData))

	w.locations[digest] = uint32(len(w.entries))
	w.entries = append(w.entries, entry)

	return nil
}

func (w *writer) Finalize() error {
	if w.tileWriter == nil {
		panic("libtiles: finalize called twice")
	}

	w.logger.Debug("libtiles: flush")
	err := w.tileWriter.Flush()
	if err != nil {
		return err
	}
	w.header.TileDataLength = w.tileOffset
	w.tileWriter = nil

	w.logger.Debug("libtiles: sort")
	slices.SortFunc(w.entries, func(a, b spec.Entry) int {
		return cmp.Compare(a.TileCode, b.TileCode)
	})

	w.logger.Debug("libtiles: compact")
	w.entries = spec.CompactEntries(w.entries)

	w.logger.Debug("libtiles: serialize")
	rootBytes, leavesBytes := spec.SerializeAll(w.entries, w.header.InternalCompression)

	w.logger.Debug("libtiles: write leaves")
	leavesOffset, err := w.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	_, err = w.file.Write(leavesBytes)
	if err != nil {
		return err
	}
	w.header.LeafDirectoryOffset = uint64(leavesOffset)
	w.header.LeafDirectoryLength = uint64(len(leavesBytes))

	w.logger.Debug("libtiles: write root")
	_, err = w.file.Seek(spec.RootDirOffset, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = w.file.Write(rootBytes)
	if err != nil {
		return err
	}
	w.header.RootOffset = spec.RootDirOffset
	w.header.RootLength = uint64(len(rootBytes))

	w.logger.Debug("libtiles: write header")
	_, err = w.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	headerData := spec.SerializeHeader(&w.header)
	_, err = w.file.Write(headerData)
	if err != nil {
		return err
	}

	w.logger.Debug("libtiles: flush")
	err = w.file.Close()
	if err != nil {
		return err
	}
	w.file = nil

	w.logger.Debug("libtiles: done!")
	return nil
}

func (w *writer) Close() error {
	if w.file == nil {
		return nil
	}
	return w.file.Close()
}
