package xyz

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/eak1mov/go-libtiles/tile"
)

// Reader implements tile.Reader interface for tiles in XYZ format.
type Reader struct {
	filePattern string
	rootDir     string
	pathRegexp  *regexp.Regexp
}

// NewReader creates a new Reader for the given file pattern (e.g. "/home/user/tiles/{z}/{x}/{y}.png").
func NewReader(filePattern string) (*Reader, error) {
	if err := validatePattern(filePattern); err != nil {
		return nil, err
	}

	// TODO(eak1mov): make filePattern regexp-safe?
	// regexPattern := regexp.QuoteMeta(filePattern)
	regexPattern := filePattern
	regexPattern = strings.ReplaceAll(regexPattern, "{x}", "(?P<x>\\d+)")
	regexPattern = strings.ReplaceAll(regexPattern, "{y}", "(?P<y>\\d+)")
	regexPattern = strings.ReplaceAll(regexPattern, "{z}", "(?P<z>\\d+)")
	pathRegex, err := regexp.Compile("^" + regexPattern + "$")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPattern, err)
	}

	path0 := formatPattern(filePattern, tile.ID{X: 0, Y: 0, Z: 0})
	path1 := formatPattern(filePattern, tile.ID{X: 1, Y: 1, Z: 1})
	for path0 != path1 {
		path0 = filepath.Dir(path0)
		path1 = filepath.Dir(path1)
	}
	rootDir := path0

	return &Reader{filePattern, rootDir, pathRegex}, nil
}

func (r *Reader) ReadTile(tileID tile.ID) ([]byte, error) {
	filePath := formatPattern(r.filePattern, tileID)
	tileData, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return make([]byte, 0), nil
	}
	if err != nil {
		return nil, err
	}
	return tileData, nil
}

func (r *Reader) VisitTiles(visitor func(tile.ID, []byte) error) error {
	return filepath.WalkDir(r.rootDir, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		matches := r.pathRegexp.FindStringSubmatch(filePath)
		if matches == nil {
			return nil // TODO(eak1mov): should we return error?
		}

		x, _ := strconv.Atoi(matches[r.pathRegexp.SubexpIndex("x")])
		y, _ := strconv.Atoi(matches[r.pathRegexp.SubexpIndex("y")])
		z, _ := strconv.Atoi(matches[r.pathRegexp.SubexpIndex("z")])

		tileData, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		return visitor(tile.ID{X: uint32(x), Y: uint32(y), Z: uint32(z)}, tileData)
	})
}
