package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/eak1mov/go-libtiles/mb"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/xyz"
	"github.com/google/subcommands"
	"github.com/schollz/progressbar/v3"
)

type convertCmd struct {
	inputFormat  string
	inputPath    string
	outputFormat string
	outputPath   string
}

func (c *convertCmd) Name() string     { return "convert" }
func (c *convertCmd) Synopsis() string { return "convert between tile storage formats" }
func (c *convertCmd) Usage() string {
	return "tileutils convert -i <path> -o <path> [-if <format> | -of <format>]\n"
}
func (c *convertCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.inputPath, "i", "", "Input path")
	f.StringVar(&c.inputFormat, "if", "", "Input format (mbtiles, pmtiles, xyz)")
	f.StringVar(&c.outputPath, "o", "", "Output path")
	f.StringVar(&c.outputFormat, "of", "", "Output format (mbtiles, pmtiles, xyz)")
}

func convertMetadata(metadata map[string]string) (pm.HeaderMetadata, error) {
	header := pm.HeaderMetadata{}

	formatValue, formatFound := metadata["format"]
	if formatFound {
		switch formatValue {
		case "pbf":
			header.TileType = spec.TileTypeMvt
			header.TileCompression = spec.CompressionGzip
		case "png":
			header.TileType = spec.TileTypePng
			header.TileCompression = spec.CompressionNone
		case "jpg":
			header.TileType = spec.TileTypeJpeg
			header.TileCompression = spec.CompressionNone
		case "webp":
			header.TileType = spec.TileTypeWebp
			header.TileCompression = spec.CompressionNone
		case "avif":
			header.TileType = spec.TileTypeAvif
			header.TileCompression = spec.CompressionNone
		}
	}

	const E7 = 10000000.0
	boundsValue, boundsFound := metadata["bounds"]
	if boundsFound {
		var coords [4]float64
		if _, err := fmt.Sscanf(boundsValue, "%f,%f,%f,%f", &coords[0], &coords[1], &coords[2], &coords[3]); err != nil {
			return pm.HeaderMetadata{}, err
		}
		header.MinLonE7 = int32(coords[0] * E7)
		header.MinLatE7 = int32(coords[1] * E7)
		header.MaxLonE7 = int32(coords[2] * E7)
		header.MaxLatE7 = int32(coords[3] * E7)
	} else {
		header.MinLonE7 = int32(-180 * E7)
		header.MinLatE7 = int32(-85 * E7)
		header.MaxLonE7 = int32(180 * E7)
		header.MaxLatE7 = int32(85 * E7)
	}

	centerValue, centerFound := metadata["center"]
	if centerFound {
		var centerLat float64
		var centerLon float64
		if _, err := fmt.Sscanf(centerValue, "%f,%f,%d", &centerLon, &centerLat, &header.CenterZoom); err != nil {
			return pm.HeaderMetadata{}, err
		}
		header.CenterLonE7 = int32(centerLon * E7)
		header.CenterLatE7 = int32(centerLat * E7)
	}

	minzoomValue, minzoomFound := metadata["minzoom"]
	if minzoomFound {
		if _, err := fmt.Sscanf(minzoomValue, "%d", &header.MinZoom); err != nil {
			return pm.HeaderMetadata{}, err
		}
	}

	maxzoomValue, maxzoomFound := metadata["maxzoom"]
	if maxzoomFound {
		if _, err := fmt.Sscanf(maxzoomValue, "%d", &header.MaxZoom); err != nil {
			return pm.HeaderMetadata{}, err
		}
	}

	return header, nil
}

func (c *convertCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	inputFormat := deduceFormat(c.inputFormat, c.inputPath)
	outputFormat := deduceFormat(c.outputFormat, c.outputPath)

	var err error
	var reader tile.Visitor
	switch inputFormat {
	case "mbtiles":
		reader, err = mb.NewReader(c.inputPath)
	case "pmtiles":
		reader, err = pm.NewFileReader(c.inputPath)
	case "xyz", "":
		reader, err = xyz.NewReader(c.inputPath)
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

	var pmHeaderMetadata pm.HeaderMetadata
	var pmJsonMetadata []byte
	if inputFormat == "mbtiles" && outputFormat == "pmtiles" {
		metadata, err := reader.(*mb.Reader).ReadMetadata()
		if err != nil {
			log.Println(err)
			return subcommands.ExitFailure
		}
		pmHeaderMetadata, err = convertMetadata(metadata)
		if err != nil {
			log.Println("failed to convert metadata:", err)
			return subcommands.ExitFailure
		}
		jsonValue, found := metadata["json"]
		if found {
			pmJsonMetadata = []byte(jsonValue)
		}
	}

	var writer tile.Writer
	switch outputFormat {
	case "mbtiles":
		writer, err = mb.NewWriter(c.outputPath, mb.WithLogger(log.Default()))
	case "pmtiles":
		writer, err = pm.NewWriter(
			c.outputPath,
			pm.WithMetadata(pmJsonMetadata),
			pm.WithHeaderMetadata(pmHeaderMetadata),
			pm.WithLogger(log.Default()),
		)
	case "xyz", "":
		writer, err = xyz.NewWriter(c.outputPath)
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

	bar := progressbar.NewOptions(-1, progressbar.OptionShowIts(), progressbar.OptionShowCount())
	err = reader.VisitTiles(func(tileID tile.ID, tileData []byte) error {
		err := writer.WriteTile(tileID, tileData)
		bar.Add(1)
		return err
	})
	bar.Finish()
	fmt.Println()

	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	if err := writer.Finalize(); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
