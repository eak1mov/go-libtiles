package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

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
	inputPath       = flag.String("i", "", "Input file path")
	inputFormat     = flag.String("if", "", "Input file format (mbtiles, pmtiles, wtiles)")
	outputIndexPath = flag.String("o", "", "Output index file path")
	outputTilesPath = flag.String("t", "", "Output tiles file path")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <path> -o <path> [-t <path> -if <format>]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var err error
	var reader tile.Visitor

	switch internal.DeduceFormat(*inputFormat, *inputPath) {
	case "mbtiles":
		reader, err = mb.NewReader(*inputPath)
	case "pmtiles":
		reader, err = pm.NewFileReader(*inputPath)
	case "wtiles":
		reader, err = wt.NewFileReader(*inputPath)
	default:
		return fmt.Errorf("invalid input format: %q", *inputFormat)
	}
	if err != nil {
		return err
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	if locationReader, ok := reader.(tile.LocationVisitor); ok {
		return exportLocations(locationReader)
	} else {
		return exportTiles(reader)
	}
}

func exportLocations(reader tile.LocationVisitor) error {
	indexFile, err := os.Create(*outputIndexPath)
	if err != nil {
		return err
	}
	defer indexFile.Close()

	bar := progressbar.DefaultBytes(-1)
	defer bar.Close()

	indexWriter := bufio.NewWriter(indexFile)
	indexEncoder := index.NewEncoder(io.MultiWriter(indexWriter, bar))

	if err := indexEncoder.EncodeFrom(reader); err != nil {
		return err
	}

	return indexWriter.Flush()
}

func exportTiles(reader tile.Visitor) error {
	indexFile, err := os.Create(*outputIndexPath)
	if err != nil {
		return err
	}
	defer indexFile.Close()
	indexWriter := bufio.NewWriter(indexFile)
	indexEncoder := index.NewEncoder(indexWriter)

	tilesFile, err := os.Create(*outputTilesPath)
	if err != nil {
		return err
	}
	defer tilesFile.Close()
	tilesWriter := bufio.NewWriter(tilesFile)
	tilesOffset := uint64(0)

	bar := progressbar.DefaultBytes(-1)
	defer bar.Close()

	err = reader.VisitTiles(func(tileID tile.ID, tileData []byte) error {
		indexItem := index.Item{
			X:      tileID.X,
			Y:      tileID.Y,
			Z:      tileID.Z,
			Length: uint32(len(tileData)),
			Offset: tilesOffset,
		}

		if err := indexEncoder.Encode(indexItem); err != nil {
			return err
		}

		if _, err := tilesWriter.Write(tileData); err != nil {
			return err
		}

		tilesOffset += uint64(len(tileData))

		bar.Add(len(tileData))

		return nil
	})

	if err != nil {
		return err
	}

	if err := tilesWriter.Flush(); err != nil {
		return err
	}
	if err := indexWriter.Flush(); err != nil {
		return err
	}

	return nil
}
