package main

import (
	"bufio"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/eak1mov/go-libtiles/pm"
	ti "github.com/eak1mov/go-libtiles/tileindex"
)

func exportIndex(inputPath string, outputPath string) error {
	reader, err := pm.NewFileReader(inputPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	indexItems := make([]ti.IndexItem, 0)
	for tileId, tileLocation := range reader.TileLocations() {
		indexItems = append(indexItems, ti.IndexItem{
			X:      tileId.X,
			Y:      tileId.Y,
			Z:      tileId.Z,
			Length: uint32(tileLocation.Length),
			Offset: tileLocation.Offset,
		})
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	err = ti.WriteIndex(indexItems, writer)
	if err != nil {
		return err
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	inputPath := flag.String("i", "", "input pmtiles file path")
	outputPath := flag.String("o", "", "output index file path")
	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	err := exportIndex(*inputPath, *outputPath)
	if err != nil {
		log.Fatal(err)
	}
}
