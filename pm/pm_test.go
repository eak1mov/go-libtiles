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
	for _, tc := range []string{
		"empty.index",
		"full5.index",
		"full08.index",
		"small.index",
		"medium.index",
		"large.index",
	} {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()

			testData, err := internal.ReadTestdata("../testdata/input.zip", tc)
			if err != nil {
				t.Fatalf("failed to read test data: %v", err)
			}

			indexItems, err := index.ReadAll(testData)
			if err != nil {
				t.Fatalf("index.ReadAll failed: %v", err)
			}

			tiles := make(map[tile.ID][]byte)
			for _, item := range indexItems {
				tiles[item.TileID()] = fmt.Appendf(nil, "%v-%v", item.Offset, item.Length)
			}

			filePath := filepath.Join(t.TempDir(), "tiles.pmtiles")
			writerMetadata := []byte(`{"foo":"bar"}`)

			writer, err := pm.NewWriter(filePath, pm.WithMetadata(writerMetadata))
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

			readerMetadata, err := reader.ReadMetadata()
			if err != nil {
				t.Fatalf("ReadMetadata failed: %v", err)
			}
			if got, want := readerMetadata, writerMetadata; !cmp.Equal(got, want) {
				t.Errorf("ReadMetadata data mismatch")
			}

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
