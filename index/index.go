// Package index provides utilities for custom index formats.
package index

import (
	"bytes"
	"encoding/binary"
	"errors"
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

func NewItem(tileID tile.ID, tileLocation tile.Location) Item {
	return Item{
		X:      tileID.X,
		Y:      tileID.Y,
		Z:      tileID.Z,
		Length: uint32(tileLocation.Length),
		Offset: tileLocation.Offset,
	}
}

func (i Item) TileID() tile.ID {
	return tile.ID{X: i.X, Y: i.Y, Z: i.Z}
}

func (i Item) TileLocation() tile.Location {
	return tile.Location{Offset: i.Offset, Length: uint64(i.Length)}
}

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(item Item) error {
	return binary.Write(e.w, binary.LittleEndian, item)
}

func (e *Encoder) EncodeAll(items []Item) error {
	return binary.Write(e.w, binary.LittleEndian, items)
}

func (e *Encoder) EncodeFrom(src tile.LocationVisitor) error {
	return src.VisitLocations(func(tileID tile.ID, location tile.Location) error {
		return e.Encode(NewItem(tileID, location))
	})
}

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (d *Decoder) Decode() (item Item, err error) {
	err = binary.Read(d.r, binary.LittleEndian, &item)
	return
}

func (d *Decoder) VisitLocations(fn tile.LocationVisitFunc) error {
	for {
		item, err := d.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := fn(item.TileID(), item.TileLocation()); err != nil {
			return err
		}
	}
}

func DecodeAll(indexData []byte) ([]Item, error) {
	count := len(indexData) / binary.Size(Item{})
	items := make([]Item, count)

	err := binary.Read(bytes.NewReader(indexData), binary.LittleEndian, items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func Collect(v tile.LocationVisitor) (items []Item, err error) {
	err = v.VisitLocations(func(tileID tile.ID, location tile.Location) error {
		items = append(items, NewItem(tileID, location))
		return nil
	})
	return
}

func ItemsVisitor(items []Item) tile.LocationVisitor {
	return itemsVisitor(items)
}

type itemsVisitor []Item

func (v itemsVisitor) VisitLocations(fn tile.LocationVisitFunc) error {
	for _, item := range v {
		if err := fn(item.TileID(), item.TileLocation()); err != nil {
			return err
		}
	}
	return nil
}
