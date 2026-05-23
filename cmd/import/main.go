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
)

func run() error {
	indexData, err := os.ReadFile(*inputIndexPath)
	if err != nil {
		return err
	}

	indexItems, err := index.ReadAll(indexData)
	if err != nil {
		return err
	}

	tilesFile, err := os.Open(*inputTilesPath)
	if err != nil {
		return err
	}
	defer tilesFile.Close()

	var writer tile.Writer
	switch internal.DeduceFormat(*outputFormat, *outputPath) {
	case "mbtiles":
		writer, err = mb.NewWriter(*outputPath)
	case "pmtiles":
		writer, err = pm.NewWriter(*outputPath, pm.WithLogger(log.Default()))
	case "wtiles":
		writer, err = wt.NewWriter(*outputPath, wt.WithLogger(log.Default()))
	default:
		return fmt.Errorf("invalid output format: %q", *outputFormat)
	}
	if err != nil {
		return err
	}
	if closer, ok := writer.(io.Closer); ok {
		defer closer.Close()
	}

	maxLength := slices.MaxFunc(indexItems, func(a, b index.Item) int {
		return cmp.Compare(a.Length, b.Length)
	}).Length
	buffer := make([]byte, maxLength)

	slices.SortFunc(indexItems, func(a, b index.Item) int {
		return cmp.Compare(a.Offset, b.Offset)
	})

	bar := progressbar.New(len(indexItems))

	for _, item := range indexItems {
		tileData := buffer[:item.Length]
		if _, err := tilesFile.ReadAt(tileData, int64(item.Offset)); err != nil {
			return err
		}
		tileID := tile.ID{X: item.X, Y: item.Y, Z: item.Z}
		if err := writer.WriteTile(tileID, tileData); err != nil {
			return err
		}
		bar.Add(1)
	}

	bar.Finish()
	fmt.Println()

	return writer.Finalize()
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s -i <path> -t <path> -o <path> [-of <format>]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
