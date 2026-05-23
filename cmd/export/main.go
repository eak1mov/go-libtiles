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

func exportTiles(reader tile.Visitor) error {
	indexFile, err := os.Create(*outputIndexPath)
	if err != nil {
		return err
	}
	defer indexFile.Close()
	indexWriter := bufio.NewWriter(indexFile)

	tilesFile, err := os.Create(*outputTilesPath)
	if err != nil {
		return err
	}
	defer tilesFile.Close()
	tilesWriter := bufio.NewWriter(tilesFile)
	tilesOffset := uint64(0)

	bar := progressbar.NewOptions(-1, progressbar.OptionShowIts(), progressbar.OptionShowCount())

	err = reader.VisitTiles(func(tileID tile.ID, tileData []byte) error {
		indexItem := index.Item{
			X:      tileID.X,
			Y:      tileID.Y,
			Z:      tileID.Z,
			Length: uint32(len(tileData)),
			Offset: tilesOffset,
		}

		if err := index.WriteItem(indexWriter, indexItem); err != nil {
			return err
		}

		if _, err := tilesWriter.Write(tileData); err != nil {
			return err
		}

		tilesOffset += uint64(len(tileData))

		bar.Add(1)

		return nil
	})

	bar.Finish()
	fmt.Println()

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

func exportLocations(reader tile.LocationVisitor) error {
	indexFile, err := os.Create(*outputIndexPath)
	if err != nil {
		return err
	}
	defer indexFile.Close()

	indexWriter := bufio.NewWriter(indexFile)

	err = reader.VisitLocations(func(tileID tile.ID, tileLocation tile.Location) error {
		return index.WriteItem(indexWriter, index.Item{
			X:      tileID.X,
			Y:      tileID.Y,
			Z:      tileID.Z,
			Length: uint32(tileLocation.Length),
			Offset: tileLocation.Offset,
		})
	})

	if err != nil {
		return err
	}

	return indexWriter.Flush()
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

	if visitor, ok := reader.(tile.LocationVisitor); ok {
		return exportLocations(visitor)
	} else {
		return exportTiles(reader)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s -i <path> -o <path> [-t <path> -if <format>]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
