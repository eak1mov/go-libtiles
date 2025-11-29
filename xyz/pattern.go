// Package xyz provides API for reading and writing tiles in XYZ directory format,
// where tiles are stored as individual files with paths like "/z/x/y.ext".
package xyz

import (
	"errors"
	"fmt"
	"strings"

	"github.com/eak1mov/go-libtiles/tile"
)

var ErrInvalidPattern = errors.New("libtiles: invalid file pattern")

func validatePattern(pattern string) error {
	for _, p := range []string{"{x}", "{y}", "{z}"} {
		if !strings.Contains(pattern, p) {
			return fmt.Errorf("%w: placeholder %v not found", ErrInvalidPattern, p)
		}
	}
	return nil
}

func formatPattern(pattern string, tileID tile.ID) string {
	result := pattern
	result = strings.ReplaceAll(result, "{x}", fmt.Sprintf("%d", tileID.X))
	result = strings.ReplaceAll(result, "{y}", fmt.Sprintf("%d", tileID.Y))
	result = strings.ReplaceAll(result, "{z}", fmt.Sprintf("%d", tileID.Z))
	return result
}
