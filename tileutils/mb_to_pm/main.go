package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/pm/spec"
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
		_, err := fmt.Sscanf(boundsValue, "%f,%f,%f,%f", &coords[0], &coords[1], &coords[2], &coords[3])
		if err != nil {
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
		_, err := fmt.Sscanf(centerValue, "%f,%f,%d", &centerLon, &centerLat, &header.CenterZoom)
		if err != nil {
			return header, err
		}
		header.CenterLonE7 = int32(centerLon * E7)
		header.CenterLatE7 = int32(centerLat * E7)
	}

	minzoomValue, minzoomFound := metadata["minzoom"]
	if minzoomFound {
		_, err := fmt.Sscanf(minzoomValue, "%d", &header.MinZoom)
		if err != nil {
			return header, err
		}
	}

	maxzoomValue, maxzoomFound := metadata["maxzoom"]
	if maxzoomFound {
		_, err := fmt.Sscanf(maxzoomValue, "%d", &header.MaxZoom)
		if err != nil {
			return header, err
		}
	}

	return header, nil
}

func importTiles(inputPath string, outputPath string) error {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", inputPath))
	if err != nil {
		return err
	}
	defer db.Close()

	metadata := make(map[string]string)
	{
		rows, err := db.Query("SELECT name, value FROM metadata")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var name, value string
			if err := rows.Scan(&name, &value); err != nil {
				return err
			}
			metadata[name] = value
		}

		if err = rows.Err(); err != nil {
			return err
		}
	}

	headerMetadata, err := headerMetadata(metadata)
	if err != nil {
		return err
	}

	jsonValue, jsonFound := metadata["json"]
	var jsonMetadata []byte = nil
	if jsonFound {
		jsonMetadata, _ = spec.Compress([]byte(jsonValue), spec.CompressionGzip)
	}

	writerParams := pm.WriterParams{
		Metadata:       jsonMetadata,
		HeaderMetadata: headerMetadata,
		Logger:         slog.Default(),
	}
	writer, err := pm.NewWriterParams(outputPath, writerParams)
	if err != nil {
		return err
	}
	defer writer.Close()

	{
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
			tileId := pm.TileId{X: x, Y: y, Z: z}

			err = writer.WriteTile(tileId, tileData)
			if err != nil {
				return err
			}

			bar.Add(1)
		}

		if err = rows.Err(); err != nil {
			return err
		}

		bar.Finish()
	}

	err = writer.Finalize()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	inputPath := flag.String("i", "", "input mbtiles file path")
	outputPath := flag.String("o", "", "output pmtiles file path")
	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	err := importTiles(*inputPath, *outputPath)
	if err != nil {
		log.Fatal(err)
	}
}
