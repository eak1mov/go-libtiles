package index

import (
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
)

const ErrInvalidIndex tile.Error = "libtiles: invalid tile index"

type FileAccessFunc func(offset, length uint64) ([]byte, error)

// Map is in-memory storage for relative locations of tile data inside data section
// of tileset (all offsets are relative to FileHeader.DataOffset).
type Map map[tile.ID]packed.Location
