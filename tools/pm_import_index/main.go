package main

import (
	"cmp"
	"flag"
	"log"
	"log/slog"
	"os"
	"slices"

	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/schollz/progressbar/v3"
)

func importTiles(inputIndexPath string, inputTilesPath string, outputPath string) error {
	indexData, err := os.ReadFile(inputIndexPath)
	if err != nil {
		return err
	}

	indexItems, err := index.ReadAll(indexData)
	if err != nil {
		return err
	}

	tilesFile, err := os.Open(inputTilesPath)
	if err != nil {
		return err
	}
	defer tilesFile.Close()

	writer, err := pm.NewWriter(outputPath, pm.WithLogger(slog.Default()))
	if err != nil {
		return err
	}
	defer writer.Close()

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
		writer.WriteTile(tileID, tileData)
		bar.Add(1)
	}

	bar.Finish()

	if err := writer.Finalize(); err != nil {
		return err
	}

	return nil
}

func main() {
	inputIndexPath := flag.String("i", "", "input index file path")
	inputTilesPath := flag.String("t", "", "input tiles file path")
	outputPath := flag.String("o", "", "output pmtiles file path")
	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	if err := importTiles(*inputIndexPath, *inputTilesPath, *outputPath); err != nil {
		log.Fatal(err)
	}
}
