package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/mb"
	"github.com/eak1mov/go-libtiles/tile"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
)

func exportTiles(inputPath string, outputIndexPath string, outputTilesPath string) error {
	reader, err := mb.NewReader(inputPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	indexFile, err := os.Create(outputIndexPath)
	if err != nil {
		return err
	}
	defer indexFile.Close()
	indexWriter := bufio.NewWriter(indexFile)

	tilesFile, err := os.Create(outputTilesPath)
	if err != nil {
		return err
	}
	defer tilesFile.Close()
	tilesWriter := bufio.NewWriter(tilesFile)
	tilesOffset := uint64(0)

	bar := progressbar.New(-1)

	err = reader.VisitTiles(func(tileID tile.ID, tileData []byte) error {
		indexItem := index.Item{
			X:      tileID.X,
			Y:      tileID.Y,
			Z:      tileID.Z,
			Length: uint32(len(tileData)),
			Offset: tilesOffset,
		}

		if err := binary.Write(indexWriter, binary.LittleEndian, indexItem); err != nil {
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

func main() {
	inputPath := flag.String("i", "", "input mbtiles file path")
	outputIndexPath := flag.String("o", "", "output index file path")
	outputTilesPath := flag.String("t", "", "output tiles file path")
	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	if err := exportTiles(*inputPath, *outputIndexPath, *outputTilesPath); err != nil {
		log.Fatal(err)
	}
}
