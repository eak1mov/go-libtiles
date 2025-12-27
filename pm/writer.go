package pm

import (
	"bufio"
	"cmp"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"slices"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/eak1mov/go-libtiles/tile"
)

// Writer implements tile.Writer interface for PMTiles format.
type Writer struct {
	logger *log.Logger
	file   *os.File
	header spec.Header

	tileWriter *bufio.Writer
	tileOffset uint64

	entries   []spec.Entry
	locations map[[16]byte]uint32 // hash -> entry index
}

type writerConfig struct {
	Metadata       []byte
	HeaderMetadata HeaderMetadata
	Logger         *log.Logger
}

type WriterOption func(*writerConfig)

func WithHeaderMetadata(headerMetadata HeaderMetadata) WriterOption {
	return func(c *writerConfig) { c.HeaderMetadata = headerMetadata }
}

func WithMetadata(metadata []byte) WriterOption {
	return func(c *writerConfig) { c.Metadata = metadata }
}

// WithLogger sets custom logger, otherwise log messages are discarded.
func WithLogger(logger *log.Logger) WriterOption {
	return func(c *writerConfig) { c.Logger = logger }
}

// NewWriter creates a new Writer for writing to a PMTiles file.
// It always creates a new file and does not support appending to an existing one.
//
// Finalize() must be called to complete writing. Failure to do so will result
// in a corrupted file.
//
// On any error during writing, the file may be left in an invalid state.
// Close() should always be called to release file resources.
func NewWriter(filePath string, opts ...WriterOption) (*Writer, error) {
	config := writerConfig{
		Logger: log.New(io.Discard, "", log.LstdFlags),
	}
	for _, opt := range opts {
		opt(&config)
	}

	var err error
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			file.Close()
		}
	}()

	header := spec.Header{
		HeaderMagic:         spec.HeaderMagicV3,
		Clustered:           true,
		InternalCompression: spec.CompressionGzip,
	}
	config.HeaderMetadata.CopyToHeader(&header)

	offset := uint64(spec.HeaderRootDirMaxLength)
	if _, err = file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, err
	}

	if config.Metadata != nil {
		metadata, _ := spec.Compress(config.Metadata, header.InternalCompression)
		if _, err = file.Write(metadata); err != nil {
			return nil, err
		}
		header.MetadataOffset = offset
		header.MetadataLength = uint64(len(metadata))
		offset += header.MetadataLength
	}

	header.TileDataOffset = offset

	return &Writer{
		logger:     config.Logger,
		file:       file,
		header:     header,
		tileWriter: bufio.NewWriter(file),
		tileOffset: 0,
		locations:  make(map[[16]byte]uint32),
	}, nil
}

func (w *Writer) Close() error {
	return w.file.Close()
}

// WriteTile writes a single tile to the PMTiles file.
//
// The caller is responsible for compressing the data according to TileCompression.
func (w *Writer) WriteTile(tileID tile.ID, tileData []byte) error {
	if w.tileWriter == nil {
		return fmt.Errorf("libtiles: write called after finalize")
	}

	if len(tileData) == 0 {
		return nil
	}

	digest := md5.Sum(tileData)
	entryIdx, exists := w.locations[digest]

	if exists {
		entry := spec.Entry{
			TileCode:  spec.EncodeTileID(tileID),
			Offset:    w.entries[entryIdx].Offset,
			Length:    w.entries[entryIdx].Length,
			RunLength: 1,
		}
		w.entries = append(w.entries, entry)
		return nil
	}

	entry := spec.Entry{
		TileCode:  spec.EncodeTileID(tileID),
		Offset:    w.tileOffset,
		Length:    uint32(len(tileData)),
		RunLength: 1,
	}

	if _, err := w.tileWriter.Write(tileData); err != nil {
		return err
	}

	w.tileOffset += uint64(len(tileData))

	w.locations[digest] = uint32(len(w.entries))
	w.entries = append(w.entries, entry)

	return nil
}

// Finalize completes the writing process by flushing buffers, writing headers,
// and creating indexes. It must be called before Close.
//
// After Finalize is called, WriteTile must not be called again.
// If Finalize returns an error, the output file may be left in a corrupted state.
func (w *Writer) Finalize() error {
	if w.tileWriter == nil {
		return fmt.Errorf("libtiles: finalize called twice")
	}

	w.logger.Println("libtiles: flush")
	if err := w.tileWriter.Flush(); err != nil {
		return err
	}
	w.header.TileDataLength = w.tileOffset
	w.tileWriter = nil

	w.logger.Println("libtiles: sort")
	slices.SortFunc(w.entries, func(a, b spec.Entry) int {
		return cmp.Compare(a.TileCode, b.TileCode)
	})

	w.logger.Println("libtiles: compact")
	w.entries = spec.CompactEntries(w.entries)

	w.logger.Println("libtiles: serialize")
	rootBytes, leavesBytes := spec.SerializeAll(w.entries, w.header.InternalCompression)

	w.logger.Println("libtiles: write leaves")
	leavesOffset, err := w.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if _, err := w.file.Write(leavesBytes); err != nil {
		return err
	}
	w.header.LeafDirectoryOffset = uint64(leavesOffset)
	w.header.LeafDirectoryLength = uint64(len(leavesBytes))

	w.logger.Println("libtiles: write root")
	if _, err := w.file.Seek(spec.RootDirOffset, io.SeekStart); err != nil {
		return err
	}
	if _, err := w.file.Write(rootBytes); err != nil {
		return err
	}
	w.header.RootOffset = spec.RootDirOffset
	w.header.RootLength = uint64(len(rootBytes))

	w.logger.Println("libtiles: write header")
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	headerData := spec.SerializeHeader(&w.header)
	if _, err := w.file.Write(headerData); err != nil {
		return err
	}

	w.logger.Println("libtiles: flush")
	if err := w.file.Sync(); err != nil {
		return err
	}

	w.logger.Println("libtiles: done!")
	return nil
}
