package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/eak1mov/go-libtiles/mb"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/eak1mov/go-libtiles/tile"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
)

func headerMetadata(metadata map[string]string) (pm.HeaderMetadata, error) {
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

	E7 := 10000000.0
	boundsValue, boundsFound := metadata["bounds"]
	if boundsFound {
		var coords [4]float64
		if _, err := fmt.Sscanf(boundsValue, "%f,%f,%f,%f", &coords[0], &coords[1], &coords[2], &coords[3]); err != nil {
			return header, err
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
			return header, err
		}
		header.CenterLonE7 = int32(centerLon * E7)
		header.CenterLatE7 = int32(centerLat * E7)
	}

	minzoomValue, minzoomFound := metadata["minzoom"]
	if minzoomFound {
		if _, err := fmt.Sscanf(minzoomValue, "%d", &header.MinZoom); err != nil {
			return header, err
		}
	}

	maxzoomValue, maxzoomFound := metadata["maxzoom"]
	if maxzoomFound {
		if _, err := fmt.Sscanf(maxzoomValue, "%d", &header.MaxZoom); err != nil {
			return header, err
		}
	}

	return header, nil
}

func importTiles(inputPath string, outputPath string) error {
	reader, err := mb.NewReader(inputPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	metadata, err := reader.ReadMetadata()
	if err != nil {
		return err
	}

	headerMetadata, err := headerMetadata(metadata)
	if err != nil {
		return err
	}

	jsonValue, jsonFound := metadata["json"]
	var jsonMetadata []byte
	if jsonFound {
		jsonMetadata = []byte(jsonValue)
	}

	writer, err := pm.NewWriter(
		outputPath,
		pm.WithMetadata(jsonMetadata),
		pm.WithHeaderMetadata(headerMetadata),
		pm.WithLogger(slog.Default()),
	)
	if err != nil {
		return err
	}
	defer writer.Close()

	bar := progressbar.New(-1)
	err = reader.VisitTiles(func(tileID tile.ID, tileData []byte) error {
		defer bar.Add(1)
		return writer.WriteTile(tileID, tileData)
	})
	bar.Finish()

	if err != nil {
		return err
	}

	if err := writer.Finalize(); err != nil {
		return err
	}

	return nil
}

func main() {
	inputPath := flag.String("i", "", "input mbtiles file path")
	outputPath := flag.String("o", "", "output pmtiles file path")
	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	if err := importTiles(*inputPath, *outputPath); err != nil {
		log.Fatal(err)
	}
}
