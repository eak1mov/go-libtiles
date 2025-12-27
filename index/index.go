// Package index provides utilities for custom index formats.
package index

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/eak1mov/go-libtiles/tile"
)

// Item represents a single record in the index, mapping tile coordinates (X, Y, Z)
// to its location (Offset, Length) in the tile storage file.
// It is designed to be easily portable to other languages and utilities.
type Item struct {
	X      uint32
	Y      uint32
	Z      uint32
	Length uint32
	Offset uint64
}

func (i Item) TileID() tile.ID {
	return tile.ID{X: i.X, Y: i.Y, Z: i.Z}
}

func (i Item) TileLocation() tile.Location {
	return tile.Location{Offset: i.Offset, Length: uint64(i.Length)}
}

func WriteAll(items []Item, writer io.Writer) error {
	return binary.Write(writer, binary.LittleEndian, items)
}

func ReadAll(indexData []byte) ([]Item, error) {
	count := len(indexData) / binary.Size(Item{})
	items := make([]Item, count)

	err := binary.Read(bytes.NewReader(indexData), binary.LittleEndian, items)
	if err != nil {
		return nil, err
	}

	return items, nil
}
