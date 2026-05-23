package sparse

import (
	"cmp"
	"slices"
	"sort"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/eak1mov/go-libtiles/wt/index"
	"github.com/eak1mov/go-libtiles/wt/index/internal/morton"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
)

func findTile(block *fbs.SparseBlock, tileID tile.ID) *fbs.LocationItem {
	tileCode := morton.Encode(tileID)

	locations := fbs.SparseLocations{}
	block.SparseLocations(&locations, int(tileID.Z))

	idx, found := sort.Find(locations.TilesLength(), func(i int) int {
		locationItem := fbs.LocationItem{}
		locations.Tiles(&locationItem, i)
		return cmp.Compare(tileCode, locationItem.TileCode())
	})

	if !found {
		return nil
	}

	locationItem := fbs.LocationItem{}
	locations.Tiles(&locationItem, idx)
	return &locationItem
}

func findLink(block *fbs.SparseBlock, tileID tile.ID) *fbs.LinkItem {
	tileCode := morton.Encode(tileID)

	locations := fbs.SparseLocations{}
	block.SparseLocations(&locations, int(tileID.Z))

	idx, found := sort.Find(locations.LinksLength(), func(i int) int {
		linkItem := fbs.LinkItem{}
		locations.Links(&linkItem, i)
		return cmp.Compare(tileCode, linkItem.TileCode())
	})

	if !found {
		return nil
	}

	linkItem := fbs.LinkItem{}
	locations.Links(&linkItem, idx)
	return &linkItem
}

func resolveLink(block *fbs.SparseBlock, tileID tile.ID) tile.ID {
	return morton.Decode(findLink(block, tileID).LinkCode(), tileID.Z)
}

func querySparse(block *fbs.SparseBlock, tileID tile.ID) (fbs.Location, error) {
	if int(tileID.Z) >= block.SparseLocationsLength() {
		return fbs.Location{}, index.ErrInvalidIndex
	}

	for findTile(block, tileID) == nil {
		pTileID := tileID
		for findTile(block, pTileID) == nil && findLink(block, pTileID) == nil {
			if pTileID.Z == 0 {
				return fbs.Location{}, index.ErrInvalidIndex
			}
			pTileID = parent(pTileID)
		}
		if findLink(block, pTileID) != nil {
			tileID = changeRoot(tileID, pTileID, resolveLink(block, pTileID))
		}
	}

	fbsLocation := fbs.Location{}
	findTile(block, tileID).Location(&fbsLocation)
	return fbsLocation, nil
}

func queryDense(block *fbs.SparseBlock, tileID tile.ID) (fbs.Location, error) {
	tileCode := morton.Encode(tileID)

	denseLocations := fbs.DenseLocations{}
	if !block.DenseLocations(&denseLocations, int(tileID.Z)) {
		return fbs.Location{}, index.ErrInvalidIndex
	}

	location := fbs.Location{}
	if !denseLocations.Locations(&location, int(tileCode)) {
		return fbs.Location{}, index.ErrInvalidIndex
	}

	return location, nil
}

func sparseToDense(block []sparseLocations) ([][]packed.Location, error) {
	zCount := uint32(len(block))

	// z -> (tileCode -> location)
	result := make([][]packed.Location, zCount)

	type TileStatus int
	const (
		StatusNone TileStatus = iota
		StatusUniq
		StatusLink
	)

	// z -> (tileCode -> status)
	tileStatus := make([][]TileStatus, zCount)
	// z -> (tileCode -> tileCode)
	linkCodes := make([][]uint32, zCount)

	for z := range zCount {
		tilesCount := tilesCountOnZoom(z)

		result[z] = make([]packed.Location, tilesCount)
		tileStatus[z] = make([]TileStatus, tilesCount)
		linkCodes[z] = make([]uint32, tilesCount)

		for _, item := range block[z].Tiles {
			tileStatus[z][item.TileCode] = StatusUniq
			result[z][item.TileCode] = packed.Location(item.Location)
		}

		for _, item := range block[z].Links {
			tileStatus[z][item.TileCode] = StatusLink
			linkCodes[z][item.TileCode] = item.LinkCode
		}
	}

	for z := range zCount {
		for tileCode := range tilesCountOnZoom(z) {
			switch tileStatus[z][tileCode] {
			case StatusUniq:
				continue // result is already filled

			// No info about tile => parent's subtree was copied.
			case StatusNone:
				if z == 0 {
					return nil, index.ErrInvalidIndex
				}

				tileID := morton.Decode(tileCode, z)
				parentTileID := parent(tileID)
				parentCode := morton.Encode(parentTileID)

				// `parentID` subtree was copied from `parentLinkID` subtree.
				if tileStatus[z-1][parentCode] != StatusLink {
					return nil, index.ErrInvalidIndex
				}
				parentLinkID := morton.Decode(linkCodes[z-1][parentCode], z-1)

				// Restore transitive/relative links:
				//   tileID -> parentTileID
				//   linkID -> parentLinkID
				//   linkCodes[parentTileID] = parentLinkID
				//   linkCodes[tileID] = linkID
				linkID := changeRoot(tileID, parentTileID, parentLinkID)

				// save explicit link from tileID to linkID
				tileStatus[z][tileCode] = StatusLink
				linkCodes[z][tileCode] = morton.Encode(linkID)

				// Link was just restored.
				fallthrough

			case StatusLink:
				result[z][tileCode] = result[z][linkCodes[z][tileCode]]
			}
		}
	}

	return result, nil
}

func denseToSparse(blockLocations [][]packed.Location) []sparseLocations {
	zCount := uint32(len(blockLocations))

	// z -> (tileCode -> eqClass)
	eqClasses := make([][]uint32, zCount)

	for z, locations := range slices.Backward(blockLocations) {
		tilesCount := tilesCountOnZoom(uint32(z))
		eqClasses[z] = make([]uint32, tilesCount)

		// location -> class
		usedLocations := make(map[packed.Location]uint32, tilesCount)

		for tileCode, location := range locations {
			class, found := usedLocations[location]
			if !found {
				class = uint32(len(usedLocations))
				usedLocations[location] = class
			}
			eqClasses[z][tileCode] = class
		}

		type tileClasses struct {
			tileClass    uint32
			childClasses [4]uint32
		}
		// (class + child classes) -> class
		usedClasses := make(map[tileClasses]uint32, tilesCount)

		for tileCode := range tilesCount {
			tile := tileClasses{
				tileClass:    eqClasses[z][tileCode],
				childClasses: [4]uint32{},
			}
			if uint32(z+1) < zCount {
				for childIdx := range tile.childClasses {
					childCode := childCode(tileCode, uint32(childIdx))
					tile.childClasses[childIdx] = eqClasses[z+1][childCode]
				}
			}
			class, found := usedClasses[tile]
			if !found {
				class = uint32(len(usedClasses))
				usedClasses[tile] = class
			}
			eqClasses[z][tileCode] = class
		}
	}

	// z -> (tileCode -> link status)
	linkStatus := make([][]bool, zCount)
	result := make([]sparseLocations, zCount)

	for z := range zCount {
		tilesCount := tilesCountOnZoom(z)
		linkStatus[z] = make([]bool, tilesCount)

		classToCode := make(map[uint32]uint32, tilesCount)

		for tileCode := range tilesCount {
			eqClass := eqClasses[z][tileCode]

			// If parent is copied, then the entire subtree is copied.
			if z > 0 && linkStatus[z-1][parentCode(tileCode)] {
				linkStatus[z][tileCode] = true
				continue
			}

			if linkCode, found := classToCode[eqClass]; found {
				linkStatus[z][tileCode] = true
				result[z].Links = append(result[z].Links, linkItem{
					TileCode: tileCode,
					LinkCode: linkCode,
				})
			} else {
				linkStatus[z][tileCode] = false
				result[z].Tiles = append(result[z].Tiles, locationItem{
					TileCode: tileCode,
					Location: uint64(blockLocations[z][tileCode]),
				})
			}
		}
	}

	return result
}
