package spec

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type Compression uint8

const (
	CompressionUnknown Compression = iota
	CompressionNone
	CompressionGzip
	CompressionBrotli
	CompressionZstd
)

type TileType uint8

const (
	TileTypeUnknown TileType = iota
	TileTypeMvt
	TileTypePng
	TileTypeJpeg
	TileTypeWebp
	TileTypeAvif
)

type Header struct {
	HeaderMagic         uint64
	RootOffset          uint64
	RootLength          uint64
	MetadataOffset      uint64
	MetadataLength      uint64
	LeafDirectoryOffset uint64
	LeafDirectoryLength uint64
	TileDataOffset      uint64
	TileDataLength      uint64
	AddressedTilesCount uint64
	TileEntriesCount    uint64
	TileContentsCount   uint64
	Clustered           bool
	InternalCompression Compression
	TileCompression     Compression
	TileType            TileType
	MinZoom             uint8
	MaxZoom             uint8
	MinLonE7            int32
	MinLatE7            int32
	MaxLonE7            int32
	MaxLatE7            int32
	CenterZoom          uint8
	CenterLonE7         int32
	CenterLatE7         int32
}

const (
	headerMagic     uint64 = 0x73656C69544D50 // "PMTiles"
	headerMagicMask uint64 = 1<<56 - 1
	HeaderMagicV3   uint64 = headerMagic | (0x03 << 56)

	HeaderLength = 127

	// spec v3: root directory MUST be contained in the first 16,384 bytes (16 KiB)
	HeaderRootDirMaxLength = 16 << 10
	RootDirOffset          = HeaderLength
	RootDirMaxLength       = HeaderRootDirMaxLength - HeaderLength
)

var ErrInvalidHeader = errors.New("invalid file header")
var ErrInvalidVersion = errors.New("invalid version")

func SerializeHeader(header *Header) []byte {
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	binary.Write(writer, binary.LittleEndian, header)
	writer.Flush()
	return buffer.Bytes()
}

func DeserializeHeader(buffer []byte) (*Header, error) {
	header := Header{}
	reader := bytes.NewReader(buffer)
	err := binary.Read(reader, binary.LittleEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidHeader, err)
	}
	if header.HeaderMagic&headerMagicMask != headerMagic {
		return nil, ErrInvalidHeader
	}
	if header.HeaderMagic != HeaderMagicV3 {
		return nil, ErrInvalidVersion
	}
	return &header, nil
}
