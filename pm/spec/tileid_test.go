package spec_test

import (
	"testing"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeTileId(t *testing.T) {
	for z := range 10 {
		for x := range 1 << z {
			for y := range 1 << z {
				tileId := spec.TileId{X: uint32(x), Y: uint32(y), Z: uint32(z)}
				require.Equal(t, tileId, spec.DecodeTileId(spec.EncodeTileId(tileId)))
			}
		}
	}
	for z := range 31 {
		tileId := spec.TileId{X: uint32(1<<z) - 1, Y: uint32(1<<z) - 1, Z: uint32(z)}
		require.Equal(t, tileId, spec.DecodeTileId(spec.EncodeTileId(tileId)))
	}
}
