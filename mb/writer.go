package mb

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/eak1mov/go-libtiles/tile"
)

// Writer implements tile.Writer interface for MBTiles format.
type Writer struct {
	db     *sql.DB
	stmt   *sql.Stmt
	logger *log.Logger
}

type writerConfig struct {
	Metadata      map[string]string
	Logger        *log.Logger
	Optimizations bool
}

type WriterOption func(*writerConfig)

func WithMetadata(metadata map[string]string) WriterOption {
	return func(c *writerConfig) { c.Metadata = metadata }
}

func WithLogger(logger *log.Logger) WriterOption {
	return func(c *writerConfig) { c.Logger = logger }
}

// WithOptimizations enables or disables SQLite performance optimizations (enabled by default).
// When enabled, it disables journaling and other safety measures, which can lead
// to data corruption in case of crashes or power loss.
func WithOptimizations(enable bool) WriterOption {
	return func(c *writerConfig) { c.Optimizations = enable }
}

// NewWriter creates a new Writer for writing to a MBTiles file.
// It always creates a new file and does not support appending to an existing one.
//
// Finalize() must be called to complete writing, otherwise the output file
// will be left in an invalid state.
//
// Close() should always be called to release database resources.
func NewWriter(filePath string, opts ...WriterOption) (*Writer, error) {
	config := writerConfig{
		Logger:        log.New(io.Discard, "", log.LstdFlags),
		Optimizations: true,
	}
	for _, opt := range opts {
		opt(&config)
	}

	if _, err := os.Stat(filePath); err == nil {
		return nil, fmt.Errorf("libtiles: file already exists: %q", filePath)
	}

	var err error
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	if config.Optimizations {
		_, err = db.Exec(`
			PRAGMA synchronous = OFF;
			PRAGMA journal_mode = MEMORY;
		`)
		if err != nil {
			return nil, err
		}
	}

	_, err = db.Exec(`
		CREATE TABLE metadata (name TEXT, value TEXT);
		CREATE TABLE tiles (
			zoom_level INTEGER,
			tile_column INTEGER,
			tile_row INTEGER,
			tile_data BLOB
		);
	`)
	if err != nil {
		return nil, err
	}

	for k, v := range config.Metadata {
		_, err = db.Exec("INSERT INTO metadata (name, value) VALUES (?, ?)", k, v)
		if err != nil {
			return nil, err
		}
	}

	stmt, err := db.Prepare("INSERT INTO tiles (zoom_level, tile_column, tile_row, tile_data) VALUES (?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}

	return &Writer{db: db, stmt: stmt, logger: config.Logger}, nil
}

func (w *Writer) Close() error {
	return errors.Join(w.stmt.Close(), w.db.Close())
}

func (w *Writer) WriteTile(tileID tile.ID, tileData []byte) error {
	x, y, z := tileID.X, tileID.Y, tileID.Z
	y = (1 << z) - 1 - y // XYZ -> TMS

	_, err := w.stmt.Exec(z, x, y, tileData)
	return err
}

func (w *Writer) Finalize() error {
	w.logger.Println("libtiles: creating index")
	_, err := w.db.Exec("CREATE UNIQUE INDEX tile_index ON tiles (zoom_level, tile_column, tile_row)")

	w.logger.Println("libtiles: done!")
	return err
}
