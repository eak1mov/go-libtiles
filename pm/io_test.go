package pm_test

import (
	"fmt"
	"maps"
	"path/filepath"
	"testing"

	"github.com/eak1mov/go-libtiles/pm"
	tu "github.com/eak1mov/go-libtiles/pm/internal"
	ti "github.com/eak1mov/go-libtiles/tileindex"
	"github.com/stretchr/testify/require"
)

func tileId(item ti.IndexItem) pm.TileId {
	return pm.TileId{X: item.X, Y: item.Y, Z: item.Z}
}

func TestWriterReader(t *testing.T) {
	for testName, fileData := range tu.TestdataCases(t, "../testdata/input.tar.gz") {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			indexItems, err := ti.ReadIndex(fileData)
			require.NoError(t, err)

			tiles := make(map[pm.TileId][]byte)
			for _, item := range indexItems {
				tiles[tileId(item)] = fmt.Appendf(nil, "%v-%v", item.Offset, item.Length)
			}

			filePath := filepath.Join(t.TempDir(), "tiles.pmtiles")

			writer, err := pm.NewWriter(filePath)
			require.NoError(t, err)
			defer writer.Close()

			for _, item := range indexItems {
				tileData := tiles[tileId(item)]
				err = writer.WriteTile(tileId(item), tileData)
				require.NoError(t, err)
			}

			err = writer.Finalize()
			require.NoError(t, err)

			reader, err := pm.NewFileReader(filePath)
			require.NoError(t, err)
			defer reader.Close()

			require.Equal(t, tiles, maps.Collect(reader.Tiles()))

			for _, item := range indexItems[:min(10_000, len(indexItems))] {
				expectedData := tiles[tileId(item)]
				tileData, err := reader.ReadTile(tileId(item))
				require.NoError(t, err)
				require.Equalf(t, expectedData, tileData, "%v", item)
			}
		})
	}
}
