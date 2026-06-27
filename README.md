# go-libtiles

[![GoDoc](https://pkg.go.dev/badge/github.com/eak1mov/go-libtiles?status.svg)](https://pkg.go.dev/github.com/eak1mov/go-libtiles)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

go-libtiles is a Go library for working with tile storage formats.
It is designed to be easy to use, efficient and to integrate smoothly into Go applications that work with map tiles.

## Features

- **Format Support**:
  [MBTiles 1.3](https://github.com/mapbox/mbtiles-spec/blob/master/1.3/spec.md),
  [PMTiles v3](https://github.com/protomaps/PMTiles/blob/main/spec/v3/spec.md),
  [WebTiles 0.2](https://github.com/eak1mov/webtiles).
- **[XYZ Directory](https://wiki.openstreetmap.org/wiki/Slippy_map_tilenames) Support**: Read and write tiles to files with paths like `/zoom/x/y.png`.
- **Format Conversion**: Convert between MBTiles, PMTiles, WebTiles and custom index formats.
- **Modular Design**: Clean separation between low-level format handling and high-level APIs.
- **High Performance**: Optimized for large tile datasets.

## Installation
Requires Go 1.26 or later, with minimal dependencies.

```bash
go get github.com/eak1mov/go-libtiles
```

## Quick Start

### Writing Tiles
```go
import (
    "github.com/eak1mov/go-libtiles/mb"
    // "github.com/eak1mov/go-libtiles/pm"
    // "github.com/eak1mov/go-libtiles/wt"
    // "github.com/eak1mov/go-libtiles/xyz"
    "github.com/eak1mov/go-libtiles/tile"
    _ "github.com/mattn/go-sqlite3" // import sqlite3 driver for mbtiles format
)

func main() {
    writer, err := mb.NewWriter("output.mbtiles")
    // writer, err := pm.NewWriter("output.pmtiles")
    // writer, err := wt.NewWriter("output.wtiles")
    // writer, err := xyz.NewWriter("output/{z}/{x}/{y}.png")
    if err != nil {
        // handle error
    }
    defer writer.Close()

    tileID := tile.ID{X: 1, Y: 2, Z: 3}
    tileData := []byte("example tile data")
    if err := writer.WriteTile(tileID, tileData); err != nil {
        // handle error
    }

    if err := writer.Finalize(); err != nil {
        // handle error
    }
}
```

### Reading Tiles
```go
import (
    "github.com/eak1mov/go-libtiles/mb"
    // "github.com/eak1mov/go-libtiles/pm"
    // "github.com/eak1mov/go-libtiles/wt"
    // "github.com/eak1mov/go-libtiles/xyz"
    "github.com/eak1mov/go-libtiles/tile"
    _ "github.com/mattn/go-sqlite3" // import sqlite3 driver for mbtiles format
)

func main() {
    reader, err := mb.NewReader("input.mbtiles")
    // reader, err := pm.NewFileReader("input.pmtiles")
    // reader, err := wt.NewFileReader("input.wtiles")
    // reader, err := xyz.NewReader("input/{z}/{x}/{y}.png")
    if err != nil {
        // handle error
    }
    defer reader.Close()

    tileID := tile.ID{X: 1, Y: 2, Z: 3}
    tileData, err := reader.ReadTile(tileID)
    if err != nil {
        // handle error
    }
    fmt.Printf("Read tile %v, size: %d bytes\n", tileID, len(tileData))

    // Iterate over all tiles
    for tileID, tileData := range tile.IterTiles(reader) {
        fmt.Printf("Tile %v: %d bytes\n", tileID, len(tileData))
    }
}
```

### Command Line Tools (examples)

```bash
# Build
go build ./cmd/convert ./cmd/export ./cmd/import ./cmd/optimize

# Convert MBTiles to PMTiles:
./convert -i input.mbtiles -o output.pmtiles

# Convert MBTiles to WebTiles:
./convert -i input.mbtiles -o output.wtiles

# Convert MBTiles to individual tiles:
./convert -i input.mbtiles -o /home/user/tiles/{z}/{x}/{y}.png

# Export tile index and tiles from MBTiles:
./export -i input.mbtiles -o output.index -t output.tiles

# Export tile index from PMTiles:
./export -i input.pmtiles -o output.index

# Import from tile index and tiles file to PMTiles:
./import -i input.index -t input.tiles -o output.pmtiles

# Optimize tileset based on access logs:
./optimize -i input.pmtiles -o output.pmtiles -l tiles-2025-12-31.txt.xz
```

## Project Structure

```
go-libtiles/
├── cmd/               # Command-line conversion tools
├── tile/              # Common tile interfaces and types
│   ├── tile.go        #   Tile ID, Reader, Writer and other interfaces
├── mb/                # MBTiles API (Reader and Writer)
├── pm/                # PMTiles API (Reader and Writer)
├── pm/spec/           # Low-level implementation of PMTiles specification
│   ├── header.go      #   Header serialization and deserialization
│   ├── directory.go   #   Tile directory management
│   ├── tileid.go      #   Tile coordinate encoding
├── wt/                # WebTiles API (Reader and Writer)
├── wt/index/formats/  # Low-level implementation of WebTiles index formats
│   ├── basic/         #   Basic index format
│   ├── plain/         #   Plain index format
│   ├── sparse/        #   Sparse index format
├── xyz/               # XYZ directory format API
├── index/             # Utilities for custom index formats
```

## Testing
```bash
# download test data:
wget -O "testdata/input.zip" "https://github.com/eak1mov/libtiles-testdata/archive/index-v0.2.0.zip"

# Run tests:
go test ./...
```

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
