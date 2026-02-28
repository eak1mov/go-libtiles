package mb

import (
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/eak1mov/go-libtiles/tile"
)

type Writer interface {
	io.Closer
	tile.Writer
}

type writerConfig struct {
	Metadata      map[string]string
	Logger        *log.Logger
	Optimizations bool
	Deduplication bool
}

type WriterOption func(*writerConfig)

func WithMetadata(metadata map[string]string) WriterOption {
	return func(c *writerConfig) { c.Metadata = metadata }
}

// WithOptimizations enables or disables SQLite performance optimizations (enabled by default).
// When enabled, it disables journaling and other safety measures, which can lead
// to data corruption in case of crashes or power loss.
func WithOptimizations(enable bool) WriterOption {
	return func(c *writerConfig) { c.Optimizations = enable }
}

func WithDeduplication(enable bool) WriterOption {
	return func(c *writerConfig) { c.Deduplication = enable }
}

// NewWriter creates a new Writer for writing to a MBTiles file.
// It always creates a new file and does not support appending to an existing one.
//
// Finalize() must be called to complete writing, otherwise the output file
// will be left in an invalid state.
//
// Close() should always be called to release database resources.
func NewWriter(filePath string, opts ...WriterOption) (writer Writer, err error) {
	config := writerConfig{
		Logger:        log.New(io.Discard, "", log.LstdFlags),
		Optimizations: true,
		Deduplication: true,
	}
	for _, opt := range opts {
		opt(&config)
	}

	if _, err := os.Stat(filePath); err == nil {
		return nil, fmt.Errorf("libtiles: file already exists: %q", filePath)
	}

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
		CREATE UNIQUE INDEX name ON metadata (name);
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

	if config.Deduplication {
		writer, err = newDedupWriter(db)
	} else {
		writer, err = newFlatWriter(db)
	}
	if err != nil {
		return nil, err
	}

	return writer, nil
}

type flatWriter struct {
	db   *sql.DB
	stmt *sql.Stmt
}

func newFlatWriter(db *sql.DB) (*flatWriter, error) {
	_, err := db.Exec(`
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

	stmt, err := db.Prepare("INSERT INTO tiles (zoom_level, tile_column, tile_row, tile_data) VALUES (?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}

	return &flatWriter{db: db, stmt: stmt}, nil
}

func (w *flatWriter) Close() error {
	return errors.Join(w.stmt.Close(), w.db.Close())
}

func (w *flatWriter) WriteTile(tileID tile.ID, tileData []byte) error {
	x, y, z := tileID.X, tileID.Y, tileID.Z
	y = (1 << z) - 1 - y // XYZ -> TMS

	_, err := w.stmt.Exec(z, x, y, tileData)
	return err
}

func (w *flatWriter) Finalize() error {
	_, err := w.db.Exec("CREATE UNIQUE INDEX tile_index ON tiles (zoom_level, tile_column, tile_row)")
	return err
}

type dedupWriter struct {
	db        *sql.DB
	dataStmt  *sql.Stmt
	indexStmt *sql.Stmt
	dataIDs   map[[16]byte]uint32 // hash -> id
}

func newDedupWriter(db *sql.DB) (*dedupWriter, error) {
	var err error

	_, err = db.Exec(`
		CREATE TABLE map (
			zoom_level INTEGER,
			tile_column INTEGER,
			tile_row INTEGER,
			tile_id INTEGER,
			PRIMARY KEY (zoom_level, tile_column, tile_row)
		) WITHOUT ROWID;
		CREATE TABLE images (tile_id INTEGER PRIMARY KEY, tile_data BLOB);
		CREATE VIEW tiles AS SELECT zoom_level, tile_column, tile_row, tile_data FROM map JOIN images USING (tile_id);
	`)
	if err != nil {
		return nil, err
	}

	dataStmt, err := db.Prepare("INSERT INTO images (tile_id, tile_data) VALUES (?, ?)")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			dataStmt.Close()
		}
	}()

	indexStmt, err := db.Prepare("INSERT INTO map (zoom_level, tile_column, tile_row, tile_id) VALUES (?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}

	return &dedupWriter{
		db:        db,
		dataStmt:  dataStmt,
		indexStmt: indexStmt,
		dataIDs:   make(map[[16]byte]uint32),
	}, nil
}

func (w *dedupWriter) Close() error {
	return errors.Join(w.indexStmt.Close(), w.dataStmt.Close(), w.db.Close())
}

func (w *dedupWriter) WriteTile(tileID tile.ID, tileData []byte) error {
	x, y, z := tileID.X, tileID.Y, tileID.Z
	y = (1 << z) - 1 - y // XYZ -> TMS

	digest := md5.Sum(tileData)
	tileDataID, exists := w.dataIDs[digest]

	if !exists {
		tileDataID = uint32(len(w.dataIDs))
		w.dataIDs[digest] = tileDataID

		if _, err := w.dataStmt.Exec(tileDataID, tileData); err != nil {
			return err
		}
	}

	_, err := w.indexStmt.Exec(z, x, y, tileDataID)
	return err
}

func (w *dedupWriter) Finalize() error {
	return nil
}
