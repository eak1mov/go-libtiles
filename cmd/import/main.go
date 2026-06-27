package main

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"slices"

	"github.com/eak1mov/go-libtiles/cmd/internal"
	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/mb"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
)

var (
	inputIndexPath = flag.String("i", "", "Input index file path")
	inputTilesPath = flag.String("t", "", "Input tiles file path")
	outputPath     = flag.String("o", "", "Output file path")
	outputFormat   = flag.String("of", "", "Output file format (mbtiles, pmtiles, wtiles)")
	sortOffsets    = flag.Bool("sort", false, "Sort by offset before writing")
	bulkMode       = flag.Bool("bulk", true, "Use bulk import")
	disableLogs    = flag.Bool("q", false, "Disable debug logs")
)

var logger = log.Default()

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <path> -t <path> -o <path> [-of <format>]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *disableLogs {
		logger = log.New(io.Discard, "", log.LstdFlags)
	}

	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	indexData, err := os.ReadFile(*inputIndexPath)
	if err != nil {
		return err
	}
	indexItems, err := index.DecodeAll(indexData)
	if err != nil {
		return err
	}
	indexData = nil

	tilesFile, err := os.Open(*inputTilesPath)
	if err != nil {
		return err
	}
	defer tilesFile.Close()

	if *sortOffsets {
		slices.SortFunc(indexItems, func(a, b index.Item) int {
			return cmp.Compare(a.Offset, b.Offset)
		})
	}

	if *bulkMode {
		return importBulk(indexItems, tilesFile)
	} else {
		return importIterative(indexItems, tilesFile)
	}
}

func importBulk(indexItems []index.Item, tilesFile *os.File) error {
	switch internal.DeduceFormat(*outputFormat, *outputPath) {
	case "pmtiles":
		return pm.Import(
			*outputPath,
			index.ItemsVisitor(indexItems),
			tilesFile,
			pm.WithLogger(logger),
		)
	case "wtiles":
		return wt.Import(
			*outputPath,
			index.ItemsVisitor(indexItems),
			tilesFile,
			wt.WithLogger(logger),
		)
	default:
		return fmt.Errorf("invalid output format: %q", *outputFormat)
	}
}

func importIterative(indexItems []index.Item, tilesFile *os.File) (err error) {
	var writer tile.Writer
	switch internal.DeduceFormat(*outputFormat, *outputPath) {
	case "mbtiles":
		writer, err = mb.NewWriter(*outputPath)
	case "pmtiles":
		writer, err = pm.NewWriter(*outputPath, pm.WithLogger(logger))
	case "wtiles":
		writer, err = wt.NewWriter(*outputPath, wt.WithLogger(logger))
	default:
		return fmt.Errorf("invalid output format: %q", *outputFormat)
	}
	if err != nil {
		return err
	}
	if closer, ok := writer.(io.Closer); ok {
		defer closer.Close()
	}

	sumLength := int64(0)
	maxLength := uint32(0)
	for _, item := range indexItems {
		sumLength += int64(item.Length)
		maxLength = max(maxLength, item.Length)
	}

	bar := progressbar.DefaultBytes(sumLength)
	defer bar.Close()

	buffer := make([]byte, maxLength)

	for _, item := range indexItems {
		tileData := buffer[:item.Length]
		if _, err := tilesFile.ReadAt(tileData, int64(item.Offset)); err != nil {
			return err
		}
		tileID := tile.ID{X: item.X, Y: item.Y, Z: item.Z}
		if err := writer.WriteTile(tileID, tileData); err != nil {
			return err
		}
		bar.Add(len(tileData))
	}

	return writer.Finalize()
}
