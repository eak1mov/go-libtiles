package xyz_test

import (
	"maps"
	"path/filepath"
	"testing"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/xyz"
	"github.com/google/go-cmp/cmp"
)

func TestWriterReader(t *testing.T) {
	rootDir := t.TempDir()
	pattern := filepath.Join(rootDir, "{z}", "{x}", "{y}.png")

	tiles := map[tile.ID][]byte{
		{X: 0, Y: 0, Z: 0}: []byte("tile000"),
		{X: 1, Y: 1, Z: 1}: []byte("tile111"),
		{X: 0, Y: 0, Z: 6}: []byte("tile006"),
		{X: 6, Y: 6, Z: 6}: []byte("tile666"),
	}

	writer, err := xyz.NewWriter(pattern)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	for tileID, tileData := range tiles {
		if err := writer.WriteTile(tileID, tileData); err != nil {
			t.Errorf("WriteTile(%v) failed: %v", tileID, err)
		}
	}

	if err := writer.Finalize(); err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	reader, err := xyz.NewReader(pattern)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	if got, want := maps.Collect(tile.IterTiles(reader)), tiles; !cmp.Equal(got, want) {
		t.Errorf("VisitTiles data mismatch")
	}

	for tileID, tileData := range tiles {
		data, err := reader.ReadTile(tileID)
		if err != nil {
			t.Errorf("ReadTile(%v) failed: %v", tileID, err)
			continue
		}
		if !cmp.Equal(data, tileData) {
			t.Errorf("ReadTile data mismatch for %v", tileID)
		}
	}

	tileData, err := reader.ReadTile(tile.ID{X: 9, Y: 9, Z: 9})
	if err != nil {
		t.Errorf("ReadTile(missing tile) failed: %v", err)
	}
	if len(tileData) != 0 {
		t.Errorf("ReadTile(missing tile) expected empty tile, got: %v bytes", len(tileData))
	}
}
