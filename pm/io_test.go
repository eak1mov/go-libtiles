package pm_test

import (
	"fmt"
	"maps"
	"path/filepath"
	"testing"

	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/internal"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/google/go-cmp/cmp"
)

func TestWriterReader(t *testing.T) {
	for testName, fileData := range internal.TestdataCases(t, "../testdata/input.tar.gz") {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			indexItems, err := index.ReadAll(fileData)
			if err != nil {
				t.Fatalf("index.ReadAll failed: %v", err)
			}

			tiles := make(map[tile.ID][]byte)
			for _, item := range indexItems {
				tiles[item.TileID()] = fmt.Appendf(nil, "%v-%v", item.Offset, item.Length)
			}

			filePath := filepath.Join(t.TempDir(), "tiles.pmtiles")

			writer, err := pm.NewWriter(filePath)
			if err != nil {
				t.Fatalf("NewWriter failed: %v", err)
			}
			defer writer.Close()

			for _, item := range indexItems {
				tileData := tiles[item.TileID()]
				if err := writer.WriteTile(item.TileID(), tileData); err != nil {
					t.Fatalf("WriteTile failed: %v", err)
				}
			}

			if err := writer.Finalize(); err != nil {
				t.Fatalf("Finalize failed: %v", err)
			}

			reader, err := pm.NewFileReader(filePath)
			if err != nil {
				t.Fatalf("NewFileReader failed: %v", err)
			}
			defer reader.Close()

			if got, want := maps.Collect(tile.IterTiles(reader)), tiles; !cmp.Equal(got, want) {
				t.Errorf("VisitTiles data mismatch")
			}

			for _, item := range indexItems[:min(10_000, len(indexItems))] {
				tileData, err := reader.ReadTile(item.TileID())
				if err != nil {
					t.Fatalf("ReadTile(%v) failed: %v", item.TileID(), err)
				}
				if got, want := tileData, tiles[item.TileID()]; !cmp.Equal(got, want) {
					t.Fatalf("ReadTile(%v) = %v, want = %v", item.TileID(), got, want)
				}
			}
		})
	}
}
