package pm

import (
	"bufio"
	"cmp"
	"crypto/md5"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/eak1mov/go-libtiles/tile"
)

// Writer implements tile.Writer interface for PMTiles format.
type Writer struct {
	logger *slog.Logger
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
	Logger         *slog.Logger
}

type WriterOption func(*writerConfig)

func WithHeaderMetadata(headerMetadata HeaderMetadata) WriterOption {
	return func(c *writerConfig) { c.HeaderMetadata = headerMetadata }
}

func WithMetadata(metadata []byte) WriterOption {
	return func(c *writerConfig) { c.Metadata = metadata }
}

func WithLogger(logger *slog.Logger) WriterOption {
	return func(c *writerConfig) { c.Logger = logger }
}

// NewWriter creates a new Writer for writing to a PMTiles file.
// It applies given options and initializes file for writing tiles.
func NewWriter(filePath string, opts ...WriterOption) (*Writer, error) {
	config := writerConfig{
		Logger: slog.New(slog.DiscardHandler),
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
		header.MetadataLength = uint64(len(config.Metadata))
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

func (w *Writer) WriteTile(tileID tile.ID, tileData []byte) error {
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

func (w *Writer) Finalize() error {
	if w.tileWriter == nil {
		panic("libtiles: finalize called twice")
	}

	w.logger.Debug("libtiles: flush")
	if err := w.tileWriter.Flush(); err != nil {
		return err
	}
	w.header.TileDataLength = w.tileOffset
	w.tileWriter = nil

	w.logger.Debug("libtiles: sort")
	slices.SortFunc(w.entries, func(a, b spec.Entry) int {
		return cmp.Compare(a.TileCode, b.TileCode)
	})

	w.logger.Debug("libtiles: compact")
	w.entries = spec.CompactEntries(w.entries)

	w.logger.Debug("libtiles: serialize")
	rootBytes, leavesBytes := spec.SerializeAll(w.entries, w.header.InternalCompression)

	w.logger.Debug("libtiles: write leaves")
	leavesOffset, err := w.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if _, err := w.file.Write(leavesBytes); err != nil {
		return err
	}
	w.header.LeafDirectoryOffset = uint64(leavesOffset)
	w.header.LeafDirectoryLength = uint64(len(leavesBytes))

	w.logger.Debug("libtiles: write root")
	if _, err := w.file.Seek(spec.RootDirOffset, io.SeekStart); err != nil {
		return err
	}
	if _, err := w.file.Write(rootBytes); err != nil {
		return err
	}
	w.header.RootOffset = spec.RootDirOffset
	w.header.RootLength = uint64(len(rootBytes))

	w.logger.Debug("libtiles: write header")
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	headerData := spec.SerializeHeader(&w.header)
	if _, err := w.file.Write(headerData); err != nil {
		return err
	}

	w.logger.Debug("libtiles: flush")
	if err := w.file.Close(); err != nil {
		return err
	}
	w.file = nil

	w.logger.Debug("libtiles: done!")
	return nil
}

func (w *Writer) Close() error {
	if w.file == nil {
		return nil
	}
	return w.file.Close()
}
