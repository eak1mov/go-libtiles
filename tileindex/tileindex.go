package tileindex

import (
	"bytes"
	"encoding/binary"
	"io"
)

type IndexItem struct {
	X      uint32
	Y      uint32
	Z      uint32
	Length uint32
	Offset uint64
}

func WriteIndex(indexItems []IndexItem, writer io.Writer) error {
	return binary.Write(writer, binary.LittleEndian, indexItems)
}

func ReadIndex(indexData []byte) ([]IndexItem, error) {
	count := len(indexData) / binary.Size(IndexItem{})
	indexItems := make([]IndexItem, count)

	err := binary.Read(bytes.NewReader(indexData), binary.LittleEndian, indexItems)
	if err != nil {
		return nil, err
	}

	return indexItems, nil
}
