package wt

import (
	"os"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/eak1mov/go-libtiles/wt/index"
	"github.com/eak1mov/go-libtiles/wt/index/formats/basic"
	"github.com/eak1mov/go-libtiles/wt/index/formats/plain"
	"github.com/eak1mov/go-libtiles/wt/index/formats/sparse"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
)

// FileAccessFunc is a function to access file data (local or remote).
// It must ensure that there are no partial reads, and handle zero-length requests correctly.
type FileAccessFunc func(offset, length uint64) ([]byte, error)

const (
	ErrInvalidHeader  tile.Error = "libtiles: invalid file header"
	ErrInvalidVersion tile.Error = "libtiles: invalid version"
	ErrInvalidRequest tile.Error = "libtiles: invalid request"
	ErrInvalidDataset tile.Error = "libtiles: invalid dataset"
)

// Reader implements tile.Reader and tile.LocationReader interfaces for WebTiles format.
type Reader struct {
	fileAccess     FileAccessFunc
	fileHeader     *fbs.FileHeader
	indexHeader    *fbs.IndexHeader
	headerMetadata []byte
}

type FileReader struct {
	Reader
	file *os.File
}

// NewFileReader opens a local WebTiles file and returns a Reader for it.
//
// The returned Reader must be closed after use to release file resources.
func NewFileReader(filePath string) (*FileReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	fileAccess := func(offset, length uint64) ([]byte, error) {
		buffer := make([]byte, length)
		if _, err := file.ReadAt(buffer, int64(offset)); err != nil {
			return nil, err
		}
		return buffer, nil
	}
	reader, err := NewReader(fileAccess)
	if err != nil {
		file.Close()
		return nil, err
	}
	return &FileReader{*reader, file}, nil
}

func (r *FileReader) Close() error {
	return r.file.Close()
}

// NewReader creates a Reader using a custom file access function.
// This is useful for remote or in-memory access.
func NewReader(fileAccess FileAccessFunc) (*Reader, error) {
	headerData, err := fileAccess(0, uint64(fbs.HeaderSizeExtended))
	if err != nil {
		return nil, err
	}

	header := fbs.Header{}
	header.Init(headerData, 0)

	fileHeader := header.FileHeader(nil)
	indexHeader := header.IndexHeader(nil)

	if fileHeader.Signature() != fbs.HeaderSignatureValue {
		return nil, ErrInvalidHeader
	}
	if fileHeader.Version() != fbs.HeaderVersionV02 {
		return nil, ErrInvalidVersion
	}

	headerMetadata := headerData[fileHeader.ExtendedOffset():][:fileHeader.ExtendedSize()]

	return &Reader{
		fileAccess:     fileAccess,
		fileHeader:     fileHeader,
		indexHeader:    indexHeader,
		headerMetadata: headerMetadata,
	}, nil
}

// HeaderMetadata returns the metadata from the WebTiles header.
func (r *Reader) HeaderMetadata() []byte {
	return r.headerMetadata
}

// ReadMetadata reads and returns the metadata section from the WebTiles file.
func (r *Reader) ReadMetadata() ([]byte, error) {
	return r.fileAccess(r.fileHeader.MetadataOffset(), r.fileHeader.MetadataSize())
}

func queryIndex(header *fbs.IndexHeader, tileID tile.ID, indexAccess index.FileAccessFunc) (tile.Location, error) {
	switch header.Format() {
	case fbs.IndexFormatBasicPlain:
		return basic.Query(header, tileID, indexAccess)
	case fbs.IndexFormatPlain:
		return plain.Query(header, tileID, indexAccess)
	case fbs.IndexFormatSparse:
		return sparse.Query(header, tileID, indexAccess)
	default:
		return tile.Location{}, ErrInvalidDataset
	}
}

func (r *Reader) ReadLocation(tileID tile.ID) (tile.Location, error) {
	if !tileID.Valid() || tileID.Z > MaxZoom {
		return tile.Location{}, ErrInvalidRequest
	}

	indexAccess := func(offset, length uint64) ([]byte, error) {
		return r.fileAccess(r.fileHeader.IndexOffset()+offset, length)
	}

	tileLocation, err := queryIndex(r.indexHeader, tileID, indexAccess)
	if err != nil {
		return tile.Location{}, err
	}

	tileLocation.Offset += r.fileHeader.DataOffset()

	return tileLocation, nil
}

// ReadTile reads a single tile from the WebTiles file.
//
// It returns the tile data or an error if the tile cannot be read.
// If the tile does not exist, it returns an empty slice with no error.
func (r *Reader) ReadTile(tileID tile.ID) ([]byte, error) {
	tileLocation, err := r.ReadLocation(tileID)
	if err != nil {
		return nil, err
	}
	return r.fileAccess(tileLocation.Offset, tileLocation.Length)
}

func readIndex(header *fbs.IndexHeader, indexData []byte) (index.Map, error) {
	switch header.Format() {
	case fbs.IndexFormatBasicPlain:
		return basic.Read(header, indexData)
	case fbs.IndexFormatPlain:
		return plain.Read(header, indexData)
	case fbs.IndexFormatSparse:
		return sparse.Read(header, indexData)
	default:
		return nil, ErrInvalidDataset
	}
}

func (r *Reader) VisitLocations(visitor func(tile.ID, tile.Location) error) error {
	indexData, err := r.fileAccess(r.fileHeader.IndexOffset(), r.fileHeader.IndexSize())
	if err != nil {
		return err
	}

	indexMap, err := readIndex(r.indexHeader, indexData)
	if err != nil {
		return err
	}

	for tileID, location := range indexMap {
		tileLocation := packed.Unpack(location)
		tileLocation.Offset += r.fileHeader.DataOffset()
		if err := visitor(tileID, tileLocation); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reader) VisitTiles(visitor func(tile.ID, []byte) error) error {
	return r.VisitLocations(func(tileID tile.ID, location tile.Location) error {
		tileData, err := r.fileAccess(location.Offset, location.Length)
		if err != nil {
			return err
		}
		return visitor(tileID, tileData)
	})
}
