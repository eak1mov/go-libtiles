package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"slices"

	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/mb"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/google/subcommands"
	"github.com/schollz/progressbar/v3"
)

type importCmd struct {
	inputIndexPath string
	inputTilesPath string
	outputFormat   string
	outputPath     string
}

func (c *importCmd) Name() string     { return "import_index" }
func (c *importCmd) Synopsis() string { return "create tileset from exported tile index and data" }
func (c *importCmd) Usage() string {
	return "tileutils import_index -i <path> -t <path> -o <path> [-of <format>]\n"
}
func (c *importCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.inputIndexPath, "i", "", "Input index file path")
	f.StringVar(&c.inputTilesPath, "t", "", "Input tiles file path")
	f.StringVar(&c.outputPath, "o", "", "Output file path")
	f.StringVar(&c.outputFormat, "of", "", "Output file format (mbtiles, pmtiles)")
}

func (c *importCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	indexData, err := os.ReadFile(c.inputIndexPath)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	indexItems, err := index.ReadAll(indexData)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	tilesFile, err := os.Open(c.inputTilesPath)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	defer tilesFile.Close()

	var writer tile.Writer
	switch deduceFormat(c.outputFormat, c.outputPath) {
	case "mbtiles":
		writer, err = mb.NewWriter(c.outputPath)
	case "pmtiles":
		writer, err = pm.NewWriter(c.outputPath, pm.WithLogger(log.Default()))
	default:
		log.Printf("invalid output format: %q", c.outputFormat)
		return subcommands.ExitFailure
	}
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
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
			log.Println(err)
			return subcommands.ExitFailure
		}
		tileID := tile.ID{X: item.X, Y: item.Y, Z: item.Z}
		if err := writer.WriteTile(tileID, tileData); err != nil {
			log.Println(err)
			return subcommands.ExitFailure
		}
		bar.Add(1)
	}

	bar.Finish()
	fmt.Println()

	if err := writer.Finalize(); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
