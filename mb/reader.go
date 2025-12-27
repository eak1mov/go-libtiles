// Package mb provides API for reading tiles and metadata in MBTiles format.
//
// Note: User must properly initialize the sqlite3 library generic driver
// (e.g. import _ "github.com/mattn/go-sqlite3") before using this package.
package mb

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/eak1mov/go-libtiles/tile"
)

// Reader implements tile.Reader interface for MBTiles format.
type Reader struct {
	db   *sql.DB
	stmt *sql.Stmt
}

// NewReader creates a new Reader for the given MBTiles file path.
//
// The returned Reader must be closed after use to release database resources.
func NewReader(filePath string) (*Reader, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", filePath))
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare("SELECT tile_data FROM tiles WHERE zoom_level = ? AND tile_column = ? AND tile_row = ?")
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Reader{db: db, stmt: stmt}, nil
}

func (r *Reader) Close() error {
	return errors.Join(r.stmt.Close(), r.db.Close())
}

func (r *Reader) ReadMetadata() (map[string]string, error) {
	metadata := make(map[string]string)

	rows, err := r.db.Query("SELECT name, value FROM metadata")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		metadata[name] = value
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return metadata, nil
}

func (r *Reader) ReadTile(tileID tile.ID) ([]byte, error) {
	x, y, z := tileID.X, tileID.Y, tileID.Z
	y = (1 << z) - 1 - y // XYZ -> TMS

	var tileData []byte
	if err := r.stmt.QueryRow(z, x, y).Scan(&tileData); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return make([]byte, 0), nil
		}
		return nil, err
	}

	return tileData, nil
}

func (r *Reader) VisitTiles(visitor func(tile.ID, []byte) error) error {
	rows, err := r.db.Query("SELECT zoom_level, tile_column, tile_row, tile_data FROM tiles")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var x, y, z uint32
		var tileData []byte

		if err := rows.Scan(&z, &x, &y, &tileData); err != nil {
			return err
		}

		y = (1 << z) - 1 - y // TMS -> XYZ

		if err := visitor(tile.ID{X: x, Y: y, Z: z}, tileData); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}
