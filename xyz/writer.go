package xyz

import (
	"os"
	"path/filepath"

	"github.com/eak1mov/go-libtiles/tile"
)

// Writer implements tile.Writer interface for tiles in XYZ format.
type Writer struct {
	filePattern string
}

// NewWriter creates a new Writer for the given file pattern (e.g. "/home/user/tiles/{z}/{x}/{y}.png").
func NewWriter(filePattern string) (*Writer, error) {
	if err := validatePattern(filePattern); err != nil {
		return nil, err
	}
	return &Writer{filePattern}, nil
}

func (w *Writer) WriteTile(tileID tile.ID, tileData []byte) error {
	filePath := formatPattern(w.filePattern, tileID)

	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, tileData, 0644)
}

func (w *Writer) Finalize() error {
	return nil
}
