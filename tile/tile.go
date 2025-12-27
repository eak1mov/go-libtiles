// Package tile provides common tile interfaces and types.
package tile

// ID represents tile coordinates in the XYZ scheme (Tiled web map).
type ID struct {
	X uint32
	Y uint32
	Z uint32
}

func (t ID) Valid() bool {
	return t.Z < 32 && t.X < (1<<t.Z) && t.Y < (1<<t.Z)
}

// Writer defines an interface for writing tiles to a tileset.
type Writer interface {
	// WriteTile writes a single tile to the tileset.
	WriteTile(tileID ID, tileData []byte) error

	// Finalize completes the writing process: flushes buffers, writes header and indices.
	// It must be called before closing the Writer.
	Finalize() error
}

type Reader interface {
	// ReadTile reads a single tile from the tileset.
	// It returns the tile data or an error if the tile cannot be read.
	// If the tile does not exist, it returns an empty slice with no error.
	ReadTile(tileID ID) ([]byte, error)
}

type Visitor interface {
	// VisitTiles visits all tiles in the tileset, calling the visitor for each.
	// It returns an error if visiting fails.
	// Order of tiles, upfront cpu and memory consumption are implementation-defined.
	VisitTiles(visitor func(ID, []byte) error) error
}

// Location represents the absolute location of tile data inside a tileset file.
type Location struct {
	Offset uint64
	Length uint64
}

type LocationReader interface {
	ReadLocation(tileID ID) (Location, error)
}

type LocationVisitor interface {
	VisitLocations(visitor func(ID, Location) error) error
}
