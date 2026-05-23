package morton_test

import (
	"testing"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/index/internal/morton"
	"github.com/google/go-cmp/cmp"
)

func TestEncodeDecode(t *testing.T) {
	for z := range 10 {
		for x := range 1 << z {
			for y := range 1 << z {
				tileID := tile.ID{X: uint32(x), Y: uint32(y), Z: uint32(z)}
				if diff := cmp.Diff(tileID, morton.Decode(morton.Encode(tileID), tileID.Z)); diff != "" {
					t.Fatalf("Decode(Encode(%v)) mismatch (-want+got):\n%v", tileID, diff)
				}
			}
		}
	}
	for z := range 16 {
		tileID := tile.ID{X: uint32(1<<z - 1), Y: uint32(1<<z - 1), Z: uint32(z)}
		if diff := cmp.Diff(tileID, morton.Decode(morton.Encode(tileID), tileID.Z)); diff != "" {
			t.Errorf("Decode(Encode(%v)) mismatch (-want+got):\n%v", tileID, diff)
		}
	}
}
