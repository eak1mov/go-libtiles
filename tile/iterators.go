package tile

import (
	"errors"
	"iter"
)

var errVisitCancelled = errors.New("visit cancelled")

// IterTiles returns an iterator over all tiles in the tileset.
// It yields tile IDs and their data. Iteration may panic on unrecoverable errors.
// TODO(eak1mov): more robust iterator interface?
func IterTiles(r Visitor) iter.Seq2[ID, []byte] {
	return func(yield func(ID, []byte) bool) {
		err := r.VisitTiles(func(tileID ID, tileData []byte) error {
			if !yield(tileID, tileData) {
				return errVisitCancelled
			}
			return nil
		})
		if err != nil && err != errVisitCancelled {
			panic(err)
		}
	}
}

func IterLocations(r LocationVisitor) iter.Seq2[ID, Location] {
	return func(yield func(ID, Location) bool) {
		err := r.VisitLocations(func(tileID ID, location Location) error {
			if !yield(tileID, location) {
				return errVisitCancelled
			}
			return nil
		})
		if err != nil && err != errVisitCancelled {
			panic(err)
		}
	}
}
