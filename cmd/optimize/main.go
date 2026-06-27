package main

import (
	"bufio"
	"cmp"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/eak1mov/go-libtiles/cmd/internal"
	"github.com/eak1mov/go-libtiles/index"
	"github.com/eak1mov/go-libtiles/pm"
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt"
	"github.com/ulikunitz/xz"
)

var (
	inputPath   = flag.String("i", "", "Input path")
	outputPath  = flag.String("o", "", "Output path")
	logsPath    = flag.String("l", "", "Logs path (e.g. tiles-2025-12-31.txt.xz)")
	format      = flag.String("f", "", "Format (pmtiles, wtiles)")
	disableLogs = flag.Bool("q", false, "Disable debug logs")
)

var logger *log.Logger

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <path> -o <path> -l <path> [-f <format>]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *disableLogs {
		logger = log.New(io.Discard, "", log.LstdFlags)
	} else {
		logger = log.Default()
	}

	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	inputFormat := internal.DeduceFormat(*format, *inputPath)

	var reader tile.LocationVisitor
	var doImport func(tile.LocationVisitor) error

	switch inputFormat {
	case "pmtiles":
		r, err := pm.NewFileReader(*inputPath)
		if err != nil {
			return fmt.Errorf("failed to create reader: %w", err)
		}
		defer r.Close()

		reader = r
		pmHeaderMetadata := r.HeaderMetadata()
		pmJsonMetadata, err := r.ReadMetadata()
		if err != nil {
			return fmt.Errorf("failed to read metadata: %w", err)
		}

		inputFile, err := os.Open(*inputPath)
		if err != nil {
			return err
		}
		defer inputFile.Close()

		doImport = func(newIndex tile.LocationVisitor) error {
			return pm.Import(
				*outputPath,
				newIndex,
				inputFile,
				pm.WithHeaderMetadata(pmHeaderMetadata),
				pm.WithMetadata(pmJsonMetadata),
				pm.WithLogger(logger),
			)
		}
	case "wtiles":
		r, err := wt.NewFileReader(*inputPath)
		if err != nil {
			return fmt.Errorf("failed to create reader: %w", err)
		}
		defer r.Close()

		reader = r
		wtHeaderMetadata := r.HeaderMetadata()
		wtMetadata, err := r.ReadMetadata()
		if err != nil {
			return fmt.Errorf("failed to read metadata: %w", err)
		}

		inputFile, err := os.Open(*inputPath)
		if err != nil {
			return err
		}
		defer inputFile.Close()

		doImport = func(newIndex tile.LocationVisitor) error {
			// TODO: forward index format from input file
			return wt.Import(
				*outputPath,
				newIndex,
				inputFile,
				wt.WithHeaderMetadata(wtHeaderMetadata),
				wt.WithMetadata(wtMetadata),
				wt.WithLogger(logger),
			)
		}
	case "index":
		inputFile, err := os.Open(*inputPath)
		if err != nil {
			return fmt.Errorf("failed to create reader: %w", err)
		}
		defer inputFile.Close()

		reader = index.NewDecoder(bufio.NewReader(inputFile))

		doImport = func(newIndex tile.LocationVisitor) error {
			outputFile, err := os.Create(*outputPath)
			if err != nil {
				return fmt.Errorf("failed to create writer: %w", err)
			}
			defer outputFile.Close()

			writer := bufio.NewWriter(outputFile)

			if err := index.NewEncoder(writer).EncodeFrom(newIndex); err != nil {
				return err
			}

			return writer.Flush()
		}
	default:
		return fmt.Errorf("invalid format: %q", inputFormat)
	}

	logger.Println("Reading index...")

	indexItems, err := index.Collect(reader)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	logger.Println("Reading logs...")

	logsData, err := readLogs(*logsPath)
	if err != nil {
		return fmt.Errorf("failed to read logs: %w", err)
	}

	logger.Println("Optimizing...")

	optimize(indexItems, logsData)
	logsData = nil

	logger.Println("Writing tiles...")

	if err := doImport(index.ItemsVisitor(indexItems)); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	logger.Println("Done!")

	return nil
}

// Read logs in OSM tile_logs format.
// Log data: https://planet.openstreetmap.org/tile_logs/
// Format description: https://github.com/openstreetmap/tilelog
func readLogs(inputPath string) (map[tile.ID]uint64, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var reader io.Reader = bufio.NewReaderSize(f, 64*1024)

	if strings.HasSuffix(inputPath, ".xz") {
		if reader, err = xz.NewReader(reader); err != nil {
			return nil, err
		}
	}

	result := make(map[tile.ID]uint64)

	scanner := bufio.NewScanner(reader)
	lineRegexp := regexp.MustCompile(`^(\d+)/(\d+)/(\d+)\s+(\d+)$`)

	for scanner.Scan() {
		matches := lineRegexp.FindSubmatch(scanner.Bytes())
		if matches == nil {
			return nil, fmt.Errorf("failed to parse line: %q", scanner.Text())
		}

		z, _ := strconv.Atoi(string(matches[1]))
		x, _ := strconv.Atoi(string(matches[2]))
		y, _ := strconv.Atoi(string(matches[3]))
		count, _ := strconv.Atoi(string(matches[4]))

		tileID := tile.ID{X: uint32(x), Y: uint32(y), Z: uint32(z)}
		result[tileID] = uint64(count)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

type dataChunk struct {
	Location     tile.Location
	RequestCount uint64
}

// Optimization strategy is based on research from https://github.com/babanov1403/tiles
func optimize(indexItems []index.Item, logsData map[tile.ID]uint64) {
	chunks := make([]dataChunk, 0)
	offsetToChunkIdx := make(map[uint64]int)

	for _, item := range indexItems {
		chunkIdx, found := offsetToChunkIdx[item.Offset]

		if !found {
			chunkIdx = len(chunks)
			offsetToChunkIdx[item.Offset] = chunkIdx

			chunks = append(chunks, dataChunk{
				Location:     item.TileLocation(),
				RequestCount: 0,
			})
		}

		chunks[chunkIdx].RequestCount += logsData[item.TileID()]
	}

	slices.SortFunc(chunks, func(a, b dataChunk) int {
		return cmp.Or(
			-cmp.Compare(a.RequestCount, b.RequestCount),
			cmp.Compare(a.Location.Offset, b.Location.Offset),
		)
	})

	newOffsets := make(map[uint64]uint64, len(chunks))
	currentOffset := uint64(0)

	for _, chunk := range chunks {
		newOffsets[chunk.Location.Offset] = currentOffset
		currentOffset += chunk.Location.Length
	}

	slices.SortStableFunc(indexItems, func(a, b index.Item) int {
		return cmp.Compare(newOffsets[a.Offset], newOffsets[b.Offset])
	})
}
