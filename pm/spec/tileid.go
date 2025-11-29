package spec

import (
	"math/bits"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/google/hilbert"
)

func EncodeTileID(tileID tile.ID) uint64 {
	h, _ := hilbert.NewHilbert(1 << tileID.Z)
	tileCode, _ := h.MapInverse(int(tileID.X), int(tileID.Y))

	tilesCount := (1<<(tileID.Z*2) - 1) / 3
	return uint64(tileCode + tilesCount)
}

func DecodeTileID(tileCode uint64) tile.ID {
	z := (bits.Len64(3*tileCode+1) - 1) / 2
	tilesCount := (1<<(z*2) - 1) / 3

	h, _ := hilbert.NewHilbert(1 << z)
	x, y, _ := h.Map(int(tileCode) - tilesCount)

	return tile.ID{X: uint32(x), Y: uint32(y), Z: uint32(z)}
}
