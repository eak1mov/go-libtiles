package main

import (
	"bufio"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/tile"
)

func exportIndex(inputPath string, outputPath string) error {
	reader, err := pm.NewFileReader(inputPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	indexItems := make([]index.Item, 0)
	for tileID, tileLocation := range tile.IterLocations(reader) {
		indexItems = append(indexItems, index.Item{
			X:      tileID.X,
			Y:      tileID.Y,
			Z:      tileID.Z,
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
	if err := index.WriteAll(indexItems, writer); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	return nil
}

func main() {
	inputPath := flag.String("i", "", "input pmtiles file path")
	outputPath := flag.String("o", "", "output index file path")
	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	if err := exportIndex(*inputPath, *outputPath); err != nil {
		log.Fatal(err)
	}
}
