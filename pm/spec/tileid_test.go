package spec_test

import (
	"testing"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/google/go-cmp/cmp"
)

func TestEncodeDecodeTileID(t *testing.T) {
	for z := range 10 {
		for x := range 1 << z {
			for y := range 1 << z {
				tileID := tile.ID{X: uint32(x), Y: uint32(y), Z: uint32(z)}
				if diff := cmp.Diff(tileID, spec.DecodeTileID(spec.EncodeTileID(tileID))); diff != "" {
					t.Errorf("DecodeTileID(EncodeTileID(%v)) mismatch (-want+got):\n%v", tileID, diff)
				}
			}
		}
	}
	for z := range 31 {
		tileID := tile.ID{X: uint32(1<<z) - 1, Y: uint32(1<<z) - 1, Z: uint32(z)}
		if diff := cmp.Diff(tileID, spec.DecodeTileID(spec.EncodeTileID(tileID))); diff != "" {
			t.Errorf("DecodeTileID(EncodeTileID(%v)) mismatch (-want+got):\n%v", tileID, diff)
		}
	}
}
