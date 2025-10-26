# go-libtiles

<!--
[![GoDoc](https://godoc.org/github.com/eak1mov/go-libtiles?status.svg)](https://godoc.org/github.com/eak1mov/go-libtiles)
[![GoDoc](https://pkg.go.dev/badge/github.com/eak1mov/go-libtiles?status.svg)](https://pkg.go.dev/github.com/eak1mov/go-libtiles)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
-->

go-libtiles is a Go library for working with tile storage formats.
It is designed to be easy to use, efficient and to integrate smoothly into Go applications that work with map tiles.

## Features

- **PMTiles Support**: Complete implementation of PMTiles v3 specification.
- **Modular Design**: Clean separation between low-level format handling and high-level APIs.
- **Format Conversion**: Convert between MBTiles, PMTiles and custom index formats.
- **High Performance**: Optimized for large tile datasets.

## Installation
Requires Go 1.24 or later, with minimal dependencies.

```bash
go get github.com/eak1mov/go-libtiles
```

## Quick Start

### Writing PMTiles
```go
import "github.com/eak1mov/go-libtiles/pm"

func main() {
    writer, err := pm.NewWriter("output.pmtiles")
    if err != nil {
        // handle error
    }
    defer writer.Close()

    tileId := pm.TileId{X: 1, Y: 2, Z: 3}
    tileData := []byte("example tile data")
    if err := writer.WriteTile(tileId, tileData); err != nil {
        // handle error
    }

    if err := writer.Finalize(); err != nil {
        // handle error
    }
}
```

### Reading PMTiles
```go
import "github.com/eak1mov/go-libtiles/pm"

func main() {
    reader, err := pm.NewFileReader("input.pmtiles")
    if err != nil {
        // handle error
    }
    defer reader.Close()

    tileId := pm.TileId{X: 1, Y: 2, Z: 3}
    tileData, err := reader.ReadTile(tileId)
    if err != nil {
        // handle error
    }
    fmt.Printf("Read tile %v, size: %d bytes\n", tileId, len(tileData))

    // Iterate over all tiles
    for tileId, tileData := range reader.Tiles() {
        fmt.Printf("Tile %v: %d bytes\n", tileId, len(tileData))
    }
}
```

### Command Line Tools (examples)

```bash
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

- `pm`: high-level API for reading and writing tiles, metadata and headers in PMTiles format.
- `pm/spec`: low-level implementation of the PMTiles v3 specification, including serialization/deserialization of headers and directories.

```
libtiles/
├── pm/                # High-level PMTiles API
│   ├── reader.go      #   Reader (access to tiles from pmtiles file)
│   ├── writer.go      #   Writer (write tiles to pmtiles file)
├── pm/spec/           # Low-level implementation of PMTiles specification
│   ├── header.go      #   Header serialization and deserialization
│   ├── directory.go   #   Tile directory management
│   ├── tileid.go      #   Tile coordinate encoding
├── tileindex/         # Custom index format utilities
└── tileutils/         # Command-line conversion tools
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
- `pm/spec` package is based on the [PMTiles specification](https://github.com/protomaps/PMTiles/blob/main/spec/v3/spec.md) by [Protomaps](https://github.com/protomaps/PMTiles).
- Uses Hilbert curve implementation from [google/hilbert](https://github.com/google/hilbert).
