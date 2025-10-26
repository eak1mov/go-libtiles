package spec_test

import (
	"cmp"
	"slices"
	"testing"

	tu "github.com/eak1mov/go-libtiles/pm/internal"
	"github.com/eak1mov/go-libtiles/pm/spec"
	ti "github.com/eak1mov/go-libtiles/tileindex"
	"github.com/stretchr/testify/require"
)

func TestDirectorySerializer(t *testing.T) {
	for testName, fileData := range tu.TestdataCases(t, "../../testdata/input.tar.gz") {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			indexItems, err := ti.ReadIndex(fileData)
			require.NoError(t, err)

			entries := make([]spec.Entry, 0)
			for _, item := range indexItems {
				tileId := spec.TileId{item.X, item.Y, item.Z}
				entries = append(entries, spec.Entry{
					TileCode:  spec.EncodeTileId(tileId),
					Offset:    item.Offset,
					Length:    item.Length,
					RunLength: 1,
				})
			}

			slices.SortFunc(entries, func(a, b spec.Entry) int {
				return cmp.Compare(a.TileCode, b.TileCode)
			})

			actualEntries, err := spec.DeserializeDirectory(spec.SerializeDirectory(entries))
			require.NoError(t, err)
			require.Equal(t, entries, actualEntries)
		})
	}
}
