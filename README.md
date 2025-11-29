# go-libtiles

[![GoDoc](https://pkg.go.dev/badge/github.com/eak1mov/go-libtiles?status.svg)](https://pkg.go.dev/github.com/eak1mov/go-libtiles)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

go-libtiles is a Go library for working with tile storage formats.
It is designed to be easy to use, efficient and to integrate smoothly into Go applications that work with map tiles.

## Features

- **PMTiles Support**: Complete implementation of [PMTiles v3](https://github.com/protomaps/PMTiles/blob/main/spec/v3/spec.md) specification.
- **MBTiles Support**: Read tiles and metadata from [MBTiles 1.3](https://github.com/mapbox/mbtiles-spec/blob/master/1.3/spec.md) format.
- **XYZ Directory Support**: Read and write tiles in standard [XYZ directory structure](https://wiki.openstreetmap.org/wiki/Slippy_map_tilenames) (`/zoom/x/y.png`).
- **Format Conversion**: Convert between MBTiles, PMTiles and custom index formats.
- **Modular Design**: Clean separation between low-level format handling and high-level APIs.
- **High Performance**: Optimized for large tile datasets.

## Installation
Requires Go 1.24 or later, with minimal dependencies.

```bash
go get github.com/eak1mov/go-libtiles
```

## Quick Start

### Writing PMTiles
```go
import (
    "github.com/eak1mov/go-libtiles/pm"
    "github.com/eak1mov/go-libtiles/tile"
)

func main() {
    writer, err := pm.NewWriter("output.pmtiles")
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

### Reading PMTiles
```go
import (
    "github.com/eak1mov/go-libtiles/pm"
    "github.com/eak1mov/go-libtiles/tile"
)

func main() {
    reader, err := pm.NewFileReader("input.pmtiles")
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

### Reading MBTiles
```go
import (
    "github.com/eak1mov/go-libtiles/mb"
    "github.com/eak1mov/go-libtiles/tile"
    _ "github.com/mattn/go-sqlite3" // Note: import sqlite3 driver
)

func main() {
    reader, err := mb.NewReader("input.mbtiles")
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
# Build tools
go build ./tools/mb_export_index
go build ./tools/pm_export_index
go build ./tools/pm_import_index
go build ./tools/mb_to_pm

# Export tile index and tiles from MBTiles:
mb_export_index -i input.mbtiles -o output.index -t output.tiles

# Export tile index from PMTiles:
pm_export_index -i input.pmtiles -o output.index

# Import from tile index and tiles file to PMTiles:
pm_import_index -i tiles.index -t tiles.dat -o output.pmtiles

# Convert MBTiles to PMTiles:
mb_to_pm -i input.mbtiles -o output.pmtiles
```

## Project Structure

- `mb`: API for reading MBTiles format.
- `pm`: high-level API for reading and writing tiles, metadata and headers in PMTiles format.
- `pm/spec`: low-level implementation of the PMTiles v3 specification, including serialization/deserialization of headers and directories.
- `xyz`: API for reading and writing standard directory-based tile structures (individual files with paths like `/z/x/y.ext`).

```
libtiles/
├── tile/              # Common tile interfaces and types
│   ├── tile.go        #   Tile ID, Reader, Writer interfaces
├── mb/                # MBTiles API
├── pm/                # High-level PMTiles API
│   ├── reader.go      #   Reader (access to tiles from pmtiles file)
│   ├── writer.go      #   Writer (write tiles to pmtiles file)
├── pm/spec/           # Low-level implementation of PMTiles specification
│   ├── header.go      #   Header serialization and deserialization
│   ├── directory.go   #   Tile directory management
│   ├── tileid.go      #   Tile coordinate encoding
├── xyz/               # XYZ directory format API
├── index/             # Utilities for custom index formats
└── tools/             # Command-line conversion tools
```

## Testing
```bash
# Put your tile index files to `testdata/input.tar.gz`:
tar -czf testdata/input.tar.gz small.index medium.index large.index

# Run tests:
go test ./...
```

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments
- `mb` package is based on the [MBTiles specification](https://github.com/mapbox/mbtiles-spec/blob/master/1.3/spec.md) by [Mapbox](https://github.com/mapbox/mbtiles-spec).
- `pm/spec` package is based on the [PMTiles specification](https://github.com/protomaps/PMTiles/blob/main/spec/v3/spec.md) by [Protomaps](https://github.com/protomaps/PMTiles).
