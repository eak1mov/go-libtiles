package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/eak1mov/go-libtiles/cmd/internal"
	"github.com/eak1mov/go-libtiles/mb"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt"
	"github.com/eak1mov/go-libtiles/xyz"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
)

var (
	inputPath    = flag.String("i", "", "Input path")
	inputFormat  = flag.String("if", "", "Input format (mbtiles, pmtiles, wtiles, xyz)")
	outputPath   = flag.String("o", "", "Output path")
	outputFormat = flag.String("of", "", "Output format (mbtiles, pmtiles, wtiles, xyz)")
	deduplicate  = flag.Bool("d", true, "Deduplicate tiles (for mbtiles format)")
	disableLogs  = flag.Bool("q", false, "Disable debug logs")
)

var logger = log.Default()

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <path> -o <path> [-if <format> | -of <format>]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *disableLogs {
		logger = log.New(io.Discard, "", log.LstdFlags)
	}

	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	inputFormat := internal.DeduceFormat(*inputFormat, *inputPath)
	outputFormat := internal.DeduceFormat(*outputFormat, *outputPath)

	var err error
	var reader tile.Visitor
	switch inputFormat {
	case "mbtiles":
		reader, err = mb.NewReader(*inputPath)
	case "pmtiles":
		reader, err = pm.NewFileReader(*inputPath)
	case "wtiles":
		reader, err = wt.NewFileReader(*inputPath)
	case "xyz", "":
		reader, err = xyz.NewReader(*inputPath)
	default:
		return fmt.Errorf("invalid input format: %q", inputFormat)
	}
	if err != nil {
		return err
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	var mbMetadata map[string]string
	var pmHeaderMetadata pm.HeaderMetadata
	var pmJsonMetadata []byte

	switch inputFormat {
	case "mbtiles":
		mbMetadata, err = reader.(*mb.Reader).ReadMetadata()
		if err != nil {
			return err
		}
	case "pmtiles":
		pmHeaderMetadata = reader.(*pm.Reader).HeaderMetadata()
		pmJsonMetadata, err = reader.(*pm.Reader).ReadMetadata()
		if err != nil {
			return err
		}
	}

	switch {
	case inputFormat == "mbtiles" && outputFormat == "pmtiles":
		pmHeaderMetadata, err = metadataMbToPm(mbMetadata)
		if err != nil {
			return fmt.Errorf("failed to convert metadata: %s", err)
		}
		jsonValue, found := mbMetadata["json"]
		if found {
			pmJsonMetadata = []byte(jsonValue)
		}
	case inputFormat == "pmtiles" && outputFormat == "mbtiles":
		mbMetadata, err = metadataPmToMb(&pmHeaderMetadata)
		if err != nil {
			return fmt.Errorf("failed to convert metadata: %s", err)
		}
		mbMetadata["name"] = filepath.Base(*inputPath)
	}

	var writer tile.Writer
	switch outputFormat {
	case "mbtiles":
		writer, err = mb.NewWriter(
			*outputPath,
			mb.WithMetadata(mbMetadata),
			mb.WithDeduplication(*deduplicate),
		)
	case "pmtiles":
		writer, err = pm.NewWriter(
			*outputPath,
			pm.WithMetadata(pmJsonMetadata),
			pm.WithHeaderMetadata(pmHeaderMetadata),
			pm.WithLogger(logger),
		)
	case "wtiles":
		writer, err = wt.NewWriter(
			*outputPath,
			wt.WithLogger(logger),
		)
	case "xyz", "":
		writer, err = xyz.NewWriter(*outputPath)
	default:
		return fmt.Errorf("invalid output format: %q", outputFormat)
	}
	if err != nil {
		return err
	}
	if closer, ok := writer.(io.Closer); ok {
		defer closer.Close()
	}

	bar := progressbar.DefaultBytes(-1)
	defer bar.Close()

	err = reader.VisitTiles(func(tileID tile.ID, tileData []byte) error {
		err := writer.WriteTile(tileID, tileData)
		bar.Add(len(tileData))
		return err
	})
	if err != nil {
		return err
	}

	return writer.Finalize()
}

func metadataMbToPm(metadata map[string]string) (pm.HeaderMetadata, error) {
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

func metadataPmToMb(pmMetadata *pm.HeaderMetadata) (map[string]string, error) {
	mbMetadata := make(map[string]string)

	switch pmMetadata.TileType {
	case spec.TileTypeMvt:
		mbMetadata["format"] = "pbf"
	case spec.TileTypePng:
		mbMetadata["format"] = "png"
	case spec.TileTypeJpeg:
		mbMetadata["format"] = "jpg"
	case spec.TileTypeWebp:
		mbMetadata["format"] = "webp"
	case spec.TileTypeAvif:
		mbMetadata["format"] = "avif"
	}

	return mbMetadata, nil
}
