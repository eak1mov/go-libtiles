package main

import (
	"cmp"
	"flag"
	"log"
	"log/slog"
	"os"
	"slices"

	"github.com/eak1mov/go-libtiles/pm"
	ti "github.com/eak1mov/go-libtiles/tileindex"
	"github.com/schollz/progressbar/v3"
)

func importTiles(inputIndexPath string, inputTilesPath string, outputPath string) error {
	indexData, err := os.ReadFile(inputIndexPath)
	if err != nil {
		return err
	}

	indexItems, err := ti.ReadIndex(indexData)
	if err != nil {
		return err
	}

	tilesFile, err := os.Open(inputTilesPath)
	if err != nil {
		return err
	}
	defer tilesFile.Close()

	params := pm.WriterParams{Logger: slog.Default()}
	writer, err := pm.NewWriterParams(outputPath, params)
	if err != nil {
		return err
	}
	defer writer.Close()

	maxLength := slices.MaxFunc(indexItems, func(a, b ti.IndexItem) int {
		return cmp.Compare(a.Length, b.Length)
	}).Length
	buffer := make([]byte, maxLength)

	slices.SortFunc(indexItems, func(a, b ti.IndexItem) int {
		return cmp.Compare(a.Offset, b.Offset)
	})

	bar := progressbar.New(len(indexItems))

	for _, item := range indexItems {
		tileData := buffer[:item.Length]
		_, err := tilesFile.ReadAt(tileData, int64(item.Offset))
		if err != nil {
			return err
		}
		tileId := pm.TileId{X: item.X, Y: item.Y, Z: item.Z}
		writer.WriteTile(tileId, tileData)
		bar.Add(1)
	}

	bar.Finish()

	err = writer.Finalize()
	if err != nil {
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

	err := importTiles(*inputIndexPath, *inputTilesPath, *outputPath)
	if err != nil {
		log.Fatal(err)
	}
}
