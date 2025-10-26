package pm

import (
	"errors"
	"io"
	"iter"
	"os"

	"github.com/eak1mov/go-libtiles/pm/spec"
)

type Reader interface {
	io.Closer

	HeaderMetadata() HeaderMetadata
	ReadMetadata() ([]byte, error)

	ReadTile(tileId TileId) ([]byte, error)
	ReadLocation(tileId TileId) (Location, error)

	Tiles() iter.Seq2[TileId, []byte]
	VisitTiles(visitor func(TileId, []byte) error) error

	TileLocations() iter.Seq2[TileId, Location]
	VisitTileLocations(visitor func(TileId, Location) error) error
}

type Location struct {
	Offset uint64
	Length uint64
}

type FileAccessFunc = func(offset, length uint64) ([]byte, error)

type reader struct {
	fileAccess FileAccessFunc
	fileCloser func() error
	header     *spec.Header
}

func NewFileReader(filePath string) (Reader, error) {
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
	return &reader{
		fileAccess: fileAccess,
		fileCloser: func() error { return file.Close() },
		header:     header,
	}, nil
}

func NewReader(fileAccess FileAccessFunc) (Reader, error) {
	headerData, err := fileAccess(0, spec.HeaderLength)
	if err != nil {
		return nil, err
	}
	header, err := spec.DeserializeHeader(headerData)
	if err != nil {
		return nil, err
	}
	return &reader{
		fileAccess: fileAccess,
		fileCloser: func() error { return nil },
		header:     header,
	}, nil
}

// TODO: add directory cache (offset -> []Entry) and reader with cache
// func NewCachingFileReader(filePath string) (Reader, error)
// func NewCachingReader(fileAccess FileAccessFunc) (Reader, error)

func (r *reader) Close() error {
	return r.fileCloser()
}

func (r *reader) HeaderMetadata() HeaderMetadata {
	result := HeaderMetadata{}
	result.CopyFromHeader(r.header)
	return result
}

func (r *reader) ReadMetadata() ([]byte, error) {
	return r.fileAccess(r.header.MetadataOffset, r.header.MetadataLength)
}

func (r *reader) readDirectory(dirOffset, dirLength uint64) ([]spec.Entry, error) {
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

func (r *reader) ReadLocation(tileId TileId) (Location, error) {
	dirOffset := r.header.RootOffset
	dirLength := r.header.RootLength
	for {
		dirEntries, err := r.readDirectory(dirOffset, dirLength)
		if err != nil {
			return Location{}, err
		}
		entry, found := spec.FindEntry(dirEntries, spec.EncodeTileId(tileId))
		if !found {
			return Location{}, nil
		}
		if entry.RunLength > 0 {
			return Location{
				Offset: r.header.TileDataOffset + entry.Offset,
				Length: uint64(entry.Length),
			}, nil
		}
		dirOffset = r.header.LeafDirectoryOffset + entry.Offset
		dirLength = uint64(entry.Length)
	}
}

func (r *reader) ReadTile(tileId TileId) ([]byte, error) {
	location, err := r.ReadLocation(tileId)
	if err != nil {
		return nil, err
	}
	tileData, err := r.fileAccess(location.Offset, location.Length)
	return tileData, err
}

func (r *reader) VisitTileLocations(visitor func(TileId, Location) error) error {
	var traverse func(uint64, uint64) error
	traverse = func(dirOffset, dirLength uint64) error {
		dirEntries, err := r.readDirectory(dirOffset, dirLength)
		if err != nil {
			return err
		}
		for _, entry := range dirEntries {
			if entry.RunLength > 0 {
				for i := range entry.RunLength {
					tileId := spec.DecodeTileId(entry.TileCode + uint64(i))
					location := Location{
						Offset: r.header.TileDataOffset + entry.Offset,
						Length: uint64(entry.Length),
					}

					err := visitor(tileId, location)
					if err != nil {
						return err
					}
				}
			} else {
				err := traverse(r.header.LeafDirectoryOffset+entry.Offset, uint64(entry.Length))
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	return traverse(r.header.RootOffset, r.header.RootLength)
}

var errVisitCancelled = errors.New("cancelled")

// panics on any error from fileAccess
func (r *reader) TileLocations() iter.Seq2[TileId, Location] {
	return func(yield func(TileId, Location) bool) {
		err := r.VisitTileLocations(func(tileId TileId, location Location) error {
			if !yield(tileId, location) {
				return errVisitCancelled
			}
			return nil
		})
		if err != nil && err != errVisitCancelled {
			panic(err)
		}
	}
}

func (r *reader) VisitTiles(visitor func(TileId, []byte) error) error {
	return r.VisitTileLocations(func(tileId TileId, location Location) error {
		tileData, err := r.fileAccess(location.Offset, location.Length)
		if err != nil {
			return err
		}
		return visitor(tileId, tileData)
	})
}

func (r *reader) Tiles() iter.Seq2[TileId, []byte] {
	return func(yield func(TileId, []byte) bool) {
		err := r.VisitTiles(func(tileId TileId, tileData []byte) error {
			if !yield(tileId, tileData) {
				return errVisitCancelled
			}
			return nil
		})
		if err != nil && err != errVisitCancelled {
			panic(err)
		}
	}
}
