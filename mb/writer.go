package mb

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/eak1mov/go-libtiles/tile"
)

// Writer implements tile.Writer interface for MBTiles format.
type Writer struct {
	db     *sql.DB
	stmt   *sql.Stmt
	logger *slog.Logger
}

type writerConfig struct {
	Metadata map[string]string
	Logger   *slog.Logger
}

type WriterOption func(*writerConfig)

func WithMetadata(metadata map[string]string) WriterOption {
	return func(c *writerConfig) { c.Metadata = metadata }
}

func WithLogger(logger *slog.Logger) WriterOption {
	return func(c *writerConfig) { c.Logger = logger }
}

// NewWriter creates a new Writer for writing to a MBTiles file.
// It applies given options and initializes database for writing tiles.
func NewWriter(filePath string, opts ...WriterOption) (*Writer, error) {
	config := writerConfig{
		Logger: slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(&config)
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

	return &Writer{db, stmt, config.Logger}, nil
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
	w.logger.Debug("libtiles: creating index")
	_, err := w.db.Exec("CREATE UNIQUE INDEX tile_index ON tiles (zoom_level, tile_column, tile_row)")

	// TODO(eak1mov): run VACUUM?
	// _, err = w.db.Exec("VACUUM")

	w.logger.Debug("libtiles: done!")
	return err
}
