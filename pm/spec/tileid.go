package spec

import (
	"math/bits"

	"github.com/google/hilbert"
)

type TileId struct {
	X uint32
	Y uint32
	Z uint32
}

func EncodeTileId(tileId TileId) uint64 {
	h, _ := hilbert.NewHilbert(1 << tileId.Z)
	tileCode, _ := h.MapInverse(int(tileId.X), int(tileId.Y))

	tilesCount := (1<<(tileId.Z*2) - 1) / 3
	return uint64(tileCode + tilesCount)
}

func DecodeTileId(tileCode uint64) TileId {
	z := (bits.Len64(3*tileCode+1) - 1) / 2
	tilesCount := (1<<(z*2) - 1) / 3

	h, _ := hilbert.NewHilbert(1 << z)
	x, y, _ := h.Map(int(tileCode) - tilesCount)

	return TileId{X: uint32(x), Y: uint32(y), Z: uint32(z)}
}
