package spec_test

import (
	"cmp"
	"slices"
	"testing"

	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/internal"
	"github.com/eak1mov/go-libtiles/pm/spec"
	gcmp "github.com/google/go-cmp/cmp"
)

func TestDirectorySerializer(t *testing.T) {
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

			testData, err := internal.ReadTestdata("../../testdata/input.zip", tc)
			if err != nil {
				t.Fatalf("failed to read test data: %v", err)
			}

			indexItems, err := index.ReadAll(testData)
			if err != nil {
				t.Fatalf("index.ReadAll failed: %v", err)
			}

			entries := make([]spec.Entry, 0)
			for _, item := range indexItems {
				entries = append(entries, spec.Entry{
					TileCode:  spec.EncodeTileID(item.TileID()),
					Offset:    item.Offset,
					Length:    item.Length,
					RunLength: 1,
				})
			}

			slices.SortFunc(entries, func(a, b spec.Entry) int {
				return cmp.Compare(a.TileCode, b.TileCode)
			})

			deserialized, err := spec.DeserializeDirectory(spec.SerializeDirectory(entries))
			if err != nil {
				t.Errorf("DeserializeDirectory failed: %v", err)
			}
			if !gcmp.Equal(entries, deserialized) {
				t.Error("DeserializeDirectory(SerializeDirectory(input)) != input")
			}
		})
	}
}
