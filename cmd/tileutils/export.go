package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/mb"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/google/subcommands"
	"github.com/schollz/progressbar/v3"
)

type exportCmd struct {
	inputFormat     string
	inputPath       string
	outputIndexPath string
	outputTilesPath string
}

func (c *exportCmd) Name() string     { return "export_index" }
func (c *exportCmd) Synopsis() string { return "export tile index and data from tileset" }
func (c *exportCmd) Usage() string {
	return "tileutils export_index -i <path> -o <path> [-t <path> -if <format>]\n"
}
func (c *exportCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.inputPath, "i", "", "Input file path")
	f.StringVar(&c.inputFormat, "if", "", "Input file format (mbtiles, pmtiles)")
	f.StringVar(&c.outputIndexPath, "o", "", "Output index file path")
	f.StringVar(&c.outputTilesPath, "t", "", "Output tiles file path")
}

func (c *exportCmd) exportTiles(reader tile.Visitor) error {
	indexFile, err := os.Create(c.outputIndexPath)
	if err != nil {
		return err
	}
	defer indexFile.Close()
	indexWriter := bufio.NewWriter(indexFile)

	tilesFile, err := os.Create(c.outputTilesPath)
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

func (c *exportCmd) exportLocations(reader tile.LocationVisitor) error {
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

	file, err := os.Create(c.outputIndexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return index.WriteAll(indexItems, file)
}

func (c *exportCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	var err error
	var reader tile.Visitor

	switch deduceFormat(c.inputFormat, c.inputPath) {
	case "mbtiles":
		reader, err = mb.NewReader(c.inputPath)
	case "pmtiles":
		reader, err = pm.NewFileReader(c.inputPath)
	default:
		log.Printf("invalid input format: %q", c.inputFormat)
		return subcommands.ExitFailure
	}
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	if visitor, ok := reader.(tile.LocationVisitor); ok {
		err = c.exportLocations(visitor)
	} else {
		err = c.exportTiles(reader)
	}

	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
