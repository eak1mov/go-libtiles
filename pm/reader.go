package pm

import (
	"os"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/eak1mov/go-libtiles/tile"
)

// FileAccessFunc is a function to access file data (local or remote).
// It must ensure that there are no partial reads.
// TODO(eak1mov): specify zero-length reads
type FileAccessFunc = func(offset, length uint64) ([]byte, error)

// Reader implements tile.Reader and tile.LocationReader interfaces for PMTiles format.
type Reader struct {
	fileAccess FileAccessFunc
	fileCloser func() error
	header     *spec.Header
}

// NewFileReader opens a local PMTiles file and returns a Reader for it.
func NewFileReader(filePath string) (*Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	fileAccess := func(offset uint64, length uint64) ([]byte, error) {
		buffer := make([]byte, length)
		if _, err := file.ReadAt(buffer, int64(offset)); err != nil {
			return nil, err
		}
		return buffer, nil
	}
	headerData, err := fileAccess(0, spec.HeaderLength)
	if err != nil {
		return nil, err
	}
	header, err := spec.DeserializeHeader(headerData)
	if err != nil {
		return nil, err
	}
	return &Reader{
		fileAccess: fileAccess,
		fileCloser: func() error { return file.Close() },
		header:     header,
	}, nil
}

// NewReader creates a Reader using a custom file access function.
// This is useful for remote or in-memory access.
func NewReader(fileAccess FileAccessFunc) (*Reader, error) {
	headerData, err := fileAccess(0, spec.HeaderLength)
	if err != nil {
		return nil, err
	}
	header, err := spec.DeserializeHeader(headerData)
	if err != nil {
		return nil, err
	}
	return &Reader{
		fileAccess: fileAccess,
		fileCloser: func() error { return nil },
		header:     header,
	}, nil
}

// TODO(eak1mov): add directory cache (offset -> []Entry) and reader with cache
// func NewCachingFileReader(filePath string) (Reader, error)
// func NewCachingReader(fileAccess FileAccessFunc) (Reader, error)

func (r *Reader) Close() error {
	return r.fileCloser()
}

// HeaderMetadata returns the metadata from the PMTiles header.
func (r *Reader) HeaderMetadata() HeaderMetadata {
	result := HeaderMetadata{}
	result.CopyFromHeader(r.header)
	return result
}

// ReadMetadata reads and returns the raw metadata from the PMTiles file.
func (r *Reader) ReadMetadata() ([]byte, error) {
	metadata, err := r.fileAccess(r.header.MetadataOffset, r.header.MetadataLength)
	if err != nil {
		return nil, err
	}
	return spec.Decompress(metadata, r.header.InternalCompression)
}

func (r *Reader) readDirectory(dirOffset, dirLength uint64) ([]spec.Entry, error) {
	dirCompressed, err := r.fileAccess(dirOffset, dirLength)
	if err != nil {
		return nil, err
	}
	dirData, err := spec.Decompress(dirCompressed, r.header.InternalCompression)
	if err != nil {
		return nil, err
	}
	dirEntries, err := spec.DeserializeDirectory(dirData)
	if err != nil {
		return nil, err
	}
	return dirEntries, nil
}

func (r *Reader) ReadLocation(tileID tile.ID) (tile.Location, error) {
	dirOffset := r.header.RootOffset
	dirLength := r.header.RootLength
	for {
		dirEntries, err := r.readDirectory(dirOffset, dirLength)
		if err != nil {
			return tile.Location{}, err
		}
		entry, found := spec.FindEntry(dirEntries, spec.EncodeTileID(tileID))
		if !found {
			return tile.Location{}, nil
		}
		if entry.RunLength > 0 {
			return tile.Location{
				Offset: r.header.TileDataOffset + entry.Offset,
				Length: uint64(entry.Length),
			}, nil
		}
		dirOffset = r.header.LeafDirectoryOffset + entry.Offset
		dirLength = uint64(entry.Length)
	}
}

func (r *Reader) ReadTile(tileID tile.ID) ([]byte, error) {
	location, err := r.ReadLocation(tileID)
	if err != nil {
		return nil, err
	}
	return r.fileAccess(location.Offset, location.Length)
}

func (r *Reader) VisitLocations(visitor func(tile.ID, tile.Location) error) error {
	var traverse func(uint64, uint64) error
	traverse = func(dirOffset, dirLength uint64) error {
		dirEntries, err := r.readDirectory(dirOffset, dirLength)
		if err != nil {
			return err
		}
		for _, entry := range dirEntries {
			if entry.RunLength > 0 {
				for i := range entry.RunLength {
					tileID := spec.DecodeTileID(entry.TileCode + uint64(i))
					location := tile.Location{
						Offset: r.header.TileDataOffset + entry.Offset,
						Length: uint64(entry.Length),
					}
					if err := visitor(tileID, location); err != nil {
						return err
					}
				}
			} else {
				if err := traverse(r.header.LeafDirectoryOffset+entry.Offset, uint64(entry.Length)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return traverse(r.header.RootOffset, r.header.RootLength)
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
