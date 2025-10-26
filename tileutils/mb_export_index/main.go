package main

import (
	"bufio"
	"database/sql"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	ti "github.com/eak1mov/go-libtiles/tileindex"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
)

func exportTiles(inputPath string, outputIndexPath string, outputTilesPath string) error {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", inputPath))
	if err != nil {
		return err
	}
	defer db.Close()

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

	rows, err := db.Query("SELECT tile_column, tile_row, zoom_level, tile_data FROM tiles")
	if err != nil {
		return err
	}
	defer rows.Close()

	bar := progressbar.New(-1)

	for rows.Next() {
		var x, y, z uint32
		var tileData []byte

		if err := rows.Scan(&x, &y, &z, &tileData); err != nil {
			return err
		}

		y = (1 << z) - 1 - y
		indexItem := ti.IndexItem{
			X:      x,
			Y:      y,
			Z:      z,
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
	}

	bar.Finish()

	if err = rows.Err(); err != nil {
		return err
	}

	if err = tilesWriter.Flush(); err != nil {
		return err
	}
	if err = indexWriter.Flush(); err != nil {
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

	err := exportTiles(*inputPath, *outputIndexPath, *outputTilesPath)
	if err != nil {
		log.Fatal(err)
	}
}
