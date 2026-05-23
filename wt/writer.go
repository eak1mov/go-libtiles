package wt

import (
	"bufio"
	"crypto/md5"
	"io"
	"log"
	"os"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/eak1mov/go-libtiles/wt/index"
	"github.com/eak1mov/go-libtiles/wt/index/formats/basic"
	"github.com/eak1mov/go-libtiles/wt/index/formats/plain"
	"github.com/eak1mov/go-libtiles/wt/index/formats/sparse"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
)

const (
	ErrInvalidTile tile.Error = "libtiles: invalid tile"
	ErrInvalidZoom tile.Error = "libtiles: invalid zoom for selected index format"
)

const MaxHeaderMetadataLength = int(fbs.HeaderSizeExtended - fbs.HeaderSizeRegular)
const MaxZoom = 24 // TODO(eak1mov): move to fbs?

// Writer implements tile.Writer interface for WebTiles format.
type Writer struct {
	logger *log.Logger
	file   *os.File

	headerData []byte
	header     fbs.Header

	tileWriter *bufio.Writer
	tileOffset uint64

	hashToLocation map[[16]byte]packed.Location
	indexMap       index.Map
	indexFormat    fbs.IndexFormat
}

type writerConfig struct {
	HeaderMetadata []byte
	Metadata       []byte
	IndexFormat    fbs.IndexFormat
	Logger         *log.Logger
}

type WriterOption func(*writerConfig)

func WithHeaderMetadata(headerMetadata []byte) WriterOption {
	return func(c *writerConfig) { c.HeaderMetadata = headerMetadata }
}

func WithMetadata(metadata []byte) WriterOption {
	return func(c *writerConfig) { c.Metadata = metadata }
}

func WithIndexFormat(indexFormat fbs.IndexFormat) WriterOption {
	return func(c *writerConfig) { c.IndexFormat = indexFormat }
}

// WithLogger sets custom logger, otherwise log messages are discarded.
func WithLogger(logger *log.Logger) WriterOption {
	return func(c *writerConfig) { c.Logger = logger }
}

// NewWriter creates a new Writer for writing to a WebTiles file.
// It always creates a new file and does not support appending to an existing one.
//
// Finalize() must be called to complete writing. Failure to do so will result
// in a corrupted file.
//
// On any error during writing, the file may be left in an invalid state.
// Close() should always be called to release file resources.
func NewWriter(filePath string, opts ...WriterOption) (*Writer, error) {
	config := writerConfig{
		IndexFormat: fbs.IndexFormatSparse,
		Logger:      log.New(io.Discard, "", log.LstdFlags),
	}
	for _, opt := range opts {
		opt(&config)
	}

	if len(config.HeaderMetadata) > MaxHeaderMetadataLength {
		return nil, tile.Error("libtiles: header metadata is too large")
	}

	_, indexFound := fbs.EnumNamesIndexFormat[config.IndexFormat]
	if !indexFound || config.IndexFormat == fbs.IndexFormatInvalid {
		return nil, tile.Error("libtiles: invalid index format")
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

	headerData := make([]byte, fbs.HeaderSizeExtended)
	header := fbs.Header{}
	header.Init(headerData, 0)

	fileHeader := header.FileHeader(nil)
	fileHeader.MutateSignature(fbs.HeaderSignatureValue)
	fileHeader.MutateVersion(fbs.HeaderVersionV02)

	if len(config.HeaderMetadata) > 0 {
		copy(headerData[fbs.HeaderSizeRegular:], config.HeaderMetadata)
		fileHeader.MutateExtendedOffset(uint64(fbs.HeaderSizeRegular))
		fileHeader.MutateExtendedSize(uint64(len(config.HeaderMetadata)))
	}

	if _, err = file.Seek(int64(fbs.HeaderSizeExtended), io.SeekStart); err != nil {
		return nil, err
	}

	dataOffset := uint64(fbs.HeaderSizeExtended)

	if len(config.Metadata) > 0 {
		if _, err = file.Write(config.Metadata); err != nil {
			return nil, err
		}
		fileHeader.MutateMetadataOffset(uint64(fbs.HeaderSizeExtended))
		fileHeader.MutateMetadataSize(uint64(len(config.Metadata)))
		dataOffset += uint64(len(config.Metadata))
	}

	fileHeader.MutateDataOffset(dataOffset)

	return &Writer{
		logger:         config.Logger,
		file:           file,
		headerData:     headerData,
		header:         header,
		tileWriter:     bufio.NewWriter(file),
		tileOffset:     0,
		hashToLocation: make(map[[16]byte]packed.Location),
		indexMap:       make(index.Map),
		indexFormat:    config.IndexFormat,
	}, nil
}

func (w *Writer) Close() error {
	return w.file.Close()
}

var maxZooms = map[fbs.IndexFormat]uint32{
	fbs.IndexFormatBasicPlain: basic.MaxZoom,
	fbs.IndexFormatPlain:      plain.MaxZoom,
	fbs.IndexFormatSparse:     sparse.MaxZoom,
}

// WriteTile writes a single tile to the WebTiles file.
func (w *Writer) WriteTile(tileID tile.ID, tileData []byte) error {
	if w.tileWriter == nil {
		return tile.Error("libtiles: write called after finalize")
	}

	if !tileID.Valid() || tileID.Z > MaxZoom {
		return ErrInvalidTile
	}

	if tileID.Z > maxZooms[w.indexFormat] {
		return ErrInvalidZoom
	}

	if len(tileData) == 0 {
		return nil
	}

	digest := md5.Sum(tileData)
	location, exists := w.hashToLocation[digest]

	if !exists {
		if _, err := w.tileWriter.Write(tileData); err != nil {
			return err
		}
		location = packed.Pack(tile.Location{
			Offset: w.tileOffset,
			Length: uint64(len(tileData)),
		})
		w.hashToLocation[digest] = location
		w.tileOffset += uint64(len(tileData))
	}

	w.indexMap[tileID] = location
	return nil
}

func writeIndex(header *fbs.IndexHeader, indexMap index.Map, indexFormat fbs.IndexFormat) ([]byte, error) {
	switch indexFormat {
	case fbs.IndexFormatBasicPlain:
		return basic.Write(header, indexMap)
	case fbs.IndexFormatPlain:
		return plain.Write(header, indexMap)
	case fbs.IndexFormatSparse:
		return sparse.Write(header, indexMap)
	default:
		return nil, tile.Error("libtiles: invalid index format")
	}
}

// Finalize completes the writing process by flushing buffers, writing headers,
// and creating indexes. It must be called before Close.
//
// After Finalize is called, WriteTile must not be called again.
// If Finalize returns an error, the output file may be left in a corrupted state.
func (w *Writer) Finalize() error {
	if w.tileWriter == nil {
		return tile.Error("libtiles: finalize called twice")
	}

	fileHeader := w.header.FileHeader(nil)
	indexHeader := w.header.IndexHeader(nil)

	w.logger.Println("libtiles: flush tiles")
	if err := w.tileWriter.Flush(); err != nil {
		return err
	}
	w.tileWriter = nil
	fileHeader.MutateDataSize(w.tileOffset)

	w.logger.Println("libtiles: prepare index")
	indexData, err := writeIndex(indexHeader, w.indexMap, w.indexFormat)
	if err != nil {
		return err
	}
	fileHeader.MutateIndexOffset(fileHeader.DataOffset() + fileHeader.DataSize())
	fileHeader.MutateIndexSize(uint64(len(indexData)))

	w.logger.Println("libtiles: write index")
	if _, err := w.file.Write(indexData); err != nil {
		return err
	}

	w.logger.Println("libtiles: write header")
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := w.file.Write(w.headerData); err != nil {
		return err
	}

	w.logger.Println("libtiles: flush file")
	if err := w.file.Sync(); err != nil {
		return err
	}

	w.logger.Println("libtiles: done!")
	return nil
}
