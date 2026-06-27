package wt_test

import (
	"fmt"
	"maps"
	"path/filepath"
	"testing"

	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/internal/testdata"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt"
	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/google/go-cmp/cmp"
)

func TestWriterReader(t *testing.T) {
	for _, format := range []fbs.IndexFormat{
		fbs.IndexFormatBasicPlain,
		fbs.IndexFormatPlain,
		fbs.IndexFormatSparse,
	} {
		for _, tc := range []string{
			"empty.index",
			"full5.index",
			"full08.index",
			"small.index",
			"medium.index",
			"large.index",
		} {
			if tc == "large.index" && format != fbs.IndexFormatSparse {
				continue
			}
			t.Run(fbs.EnumNamesIndexFormat[format]+"."+tc, func(t *testing.T) {
				// t.Parallel()

				testData, err := testdata.Read("../testdata/input.zip", tc)
				if err != nil {
					t.Fatalf("failed to read test data: %v", err)
				}

				indexItems, err := index.DecodeAll(testData)
				if err != nil {
					t.Fatalf("index.ReadAll failed: %v", err)
				}

				testTiles := make(map[tile.ID][]byte)
				for _, item := range indexItems {
					testTiles[item.TileID()] = fmt.Appendf(nil, "%v-%v", item.Offset, item.Length)
				}

				filePath := filepath.Join(t.TempDir(), "tiles.wtiles")
				writerMetadata := []byte(`{"foo":"bar"}`)
				writerHeaderMetadata := []byte(`{"foo2":"bar2"}`)

				writer, err := wt.NewWriter(
					filePath,
					wt.WithMetadata(writerMetadata),
					wt.WithHeaderMetadata(writerHeaderMetadata),
					wt.WithIndexFormat(format),
				)
				if err != nil {
					t.Fatalf("NewWriter failed: %v", err)
				}
				defer writer.Close()

				for _, item := range indexItems {
					tileData := testTiles[item.TileID()]
					if err := writer.WriteTile(item.TileID(), tileData); err != nil {
						t.Fatalf("WriteTile failed: %v", err)
					}
				}

				if err := writer.Finalize(); err != nil {
					t.Fatalf("Finalize failed: %v", err)
				}

				reader, err := wt.NewFileReader(filePath)
				if err != nil {
					t.Fatalf("NewFileReader failed: %v", err)
				}
				defer reader.Close()

				readerHeaderMetadata := reader.HeaderMetadata()
				if got, want := readerHeaderMetadata, writerHeaderMetadata; !cmp.Equal(got, want) {
					t.Errorf("HeaderMetadata data mismatch")
				}

				readerMetadata, err := reader.ReadMetadata()
				if err != nil {
					t.Fatalf("ReadMetadata failed: %v", err)
				}
				if got, want := readerMetadata, writerMetadata; !cmp.Equal(got, want) {
					t.Errorf("ReadMetadata data mismatch")
				}

				if got, want := maps.Collect(tile.IterTiles(reader)), testTiles; !cmp.Equal(got, want) {
					t.Errorf("VisitTiles data mismatch")
				}

				for _, item := range indexItems[:min(10_000, len(indexItems))] {
					tileData, err := reader.ReadTile(item.TileID())
					if err != nil {
						t.Fatalf("ReadTile(%v) failed: %v", item.TileID(), err)
					}
					if got, want := tileData, testTiles[item.TileID()]; !cmp.Equal(got, want) {
						t.Fatalf("ReadTile(%v) = %v, want = %v", item.TileID(), string(got), string(want))
					}
				}
			})
		}
	}
}
