// Package sparse provides low-level implementation of WebTiles Sparse index format.
package sparse

import (
	"crypto/md5"
	"slices"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/eak1mov/go-libtiles/wt/index"
	"github.com/eak1mov/go-libtiles/wt/index/block"
	"github.com/eak1mov/go-libtiles/wt/index/internal/morton"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
)

const MaxZoom = 24

func QueryBlock(tileID tile.ID, blockRange block.ZoomRange, blockData []byte) (tile.Location, error) {
	nextZ := min(blockRange.End(), tileID.Z)

	nextTileID := parentN(tileID, tileID.Z-nextZ)
	blockTileID := parentN(tileID, tileID.Z-blockRange.Start)
	innerTileID := subtractTileIDs(nextTileID, blockTileID)

	blockFbs := fbs.GetRootAsSparseBlock(blockData, 0)

	fbsLocation, err := func() (fbs.Location, error) {
		switch blockFbs.BlockType() {
		case fbs.BlockTypeDense:
			return queryDense(blockFbs, innerTileID)
		case fbs.BlockTypeSparse:
			return querySparse(blockFbs, innerTileID)
		default:
			return fbs.Location{}, index.ErrInvalidIndex
		}
	}()
	if err != nil {
		return tile.Location{}, err
	}

	tileLocation := packed.Unpack(packed.ReadFbs(fbsLocation))

	return tileLocation, nil
}

func Query(header *fbs.IndexHeader, tileID tile.ID, indexAccess index.FileAccessFunc) (tile.Location, error) {
	if tileID.Z > uint32(header.MaxZoom()) {
		return tile.Location{}, nil
	}

	blockLevels := block.LevelsMask(header.BlockLevelsMask())
	location := tile.Location{Offset: header.RootOffset(), Length: header.RootSize()}

	for blockRange := range blockLevels.Ranges() {
		if tileID.Z < blockRange.Start {
			break
		}
		if location.Length == 0 {
			return location, nil
		}
		blockData, err := indexAccess(location.Offset, location.Length)
		if err != nil {
			return tile.Location{}, err
		}
		nextLocation, err := QueryBlock(tileID, blockRange, blockData)
		if err != nil {
			return tile.Location{}, err
		}
		location = nextLocation
	}

	return location, nil
}

func Read(header *fbs.IndexHeader, indexData []byte) (index.Map, error) {
	blockLevels := block.LevelsMask(header.BlockLevelsMask())
	rootLocation := tile.Location{Offset: header.RootOffset(), Length: header.RootSize()}

	result := make(index.Map, len(indexData)/packed.LocationLength)

	result[tile.ID{X: 0, Y: 0, Z: 0}] = packed.Pack(rootLocation)

	for bIdx, blockRange := range slices.Collect(blockLevels.Ranges()) {
		blockZoomCount := blockRange.Count
		if bIdx != blockLevels.RangesCount()-1 {
			blockZoomCount++ // location of the next block (blockRoot)
		}

		for blockCode := range tilesCountOnZoom(blockRange.Start) {
			blockTileID := morton.Decode(blockCode, blockRange.Start)

			blockRoot, blockRootFound := result[blockTileID]
			if blockRootFound {
				delete(result, blockTileID)
			}

			if blockRoot.Length() == 0 {
				continue
			}

			blockData := indexData[blockRoot.Offset():]
			blockFbs := fbs.GetRootAsSparseBlock(blockData, 0)
			var blockLocations [][]packed.Location

			switch blockFbs.BlockType() {
			case fbs.BlockTypeDense:
				blockLocations = readDense(blockFbs)
				if !validateDense(blockLocations, blockZoomCount) {
					return nil, index.ErrInvalidIndex
				}
			case fbs.BlockTypeSparse:
				sparseLocations := readSparse(blockFbs)
				if !validateSparse(sparseLocations, blockZoomCount) {
					return nil, index.ErrInvalidIndex
				}
				denseLocations, err := sparseToDense(sparseLocations)
				if err != nil {
					return nil, err
				}
				blockLocations = denseLocations
			default:
				return nil, index.ErrInvalidIndex
			}

			for innerZ, locations := range blockLocations {
				for innerCode, tileLocation := range locations {
					if tileLocation.Length() == 0 {
						continue
					}

					innerTileID := morton.Decode(uint32(innerCode), uint32(innerZ))
					tileID := combineTileIDs(blockTileID, innerTileID)

					result[tileID] = tileLocation
				}
			}
		}
	}

	return result, nil
}

func Write(header *fbs.IndexHeader, indexMap index.Map) ([]byte, error) {
	maxZoom := uint32(0)
	for tileID := range indexMap {
		maxZoom = max(maxZoom, tileID.Z)
	}

	var blockLevels block.LevelsMask
	if maxZoom <= 8 {
		blockLevels = block.NewLevelsMask(0, maxZoom+1)
	} else if maxZoom <= 15 {
		// 9 -> [4]
		// 10..11 -> [5]
		// 12..13 -> [6]
		// 14..15 -> [7]
		blockLevels = block.NewLevelsMask(0, maxZoom/2, maxZoom+1)
	} else {
		// 16..17 -> [5, 10]
		// 18..20 -> [6, 12]
		// 21..24 -> [7, 14]
		blockLevels = block.NewLevelsMask(0, maxZoom/3, maxZoom/3*2, maxZoom+1)
	}

	header.MutateMagic(fbs.IndexMagicValue)
	header.MutateFormat(fbs.IndexFormatSparse)
	header.MutateMaxZoom(uint64(maxZoom))
	header.MutateBlockLevelsMask(uint64(blockLevels))

	result := make([]byte, 0, len(indexMap)*packed.LocationLength)

	// blockIdx -> [blockTileID]
	usedBlocks := make([]map[tile.ID]bool, blockLevels.RangesCount())
	for i := range usedBlocks {
		usedBlocks[i] = make(map[tile.ID]bool)
	}

	for tileID := range indexMap {
		for bIdx, blockRange := range slices.Collect(blockLevels.Ranges()) {
			if tileID.Z < blockRange.Start {
				break
			}
			blockTileID := parentN(tileID, tileID.Z-blockRange.Start)
			usedBlocks[bIdx][blockTileID] = true
		}
	}

	hashToLocation := make(map[[16]byte]packed.Location)

	for bIdx, blockRange := range slices.Backward(slices.Collect(blockLevels.Ranges())) {
		blockZoomCount := blockRange.Count
		if bIdx != blockLevels.RangesCount()-1 {
			blockZoomCount++ // location of the next block (blockRoot)
		}

		for blockTileID := range usedBlocks[bIdx] {
			blockLocations := make([][]packed.Location, blockZoomCount)

			for innerZ := range blockZoomCount {
				innerTilesCount := tilesCountOnZoom(innerZ)
				blockLocations[innerZ] = make([]packed.Location, innerTilesCount)

				for innerCode := range innerTilesCount {
					innerTileID := morton.Decode(innerCode, innerZ)
					tileID := combineTileIDs(blockTileID, innerTileID)

					if location, found := indexMap[tileID]; found {
						blockLocations[innerZ][innerCode] = location
					}
				}
			}

			denseData := writeDense(blockLocations)
			denseDataHash := md5.Sum(denseData)

			if location, found := hashToLocation[denseDataHash]; found {
				indexMap[blockTileID] = location
				continue
			}

			sparseData := writeSparse(denseToSparse(blockLocations))
			sparseDataHash := md5.Sum(sparseData)

			if location, found := hashToLocation[sparseDataHash]; found {
				indexMap[blockTileID] = location
				continue
			}

			var blockData []byte
			if len(denseData) <= len(sparseData) {
				blockData = denseData
			} else {
				blockData = sparseData
			}

			blockRoot := packed.Pack(tile.Location{
				Offset: uint64(len(result)),
				Length: uint64(len(blockData)),
			})
			indexMap[blockTileID] = blockRoot
			hashToLocation[denseDataHash] = blockRoot
			hashToLocation[sparseDataHash] = blockRoot

			result = append(result, blockData...)
		}
	}

	rootLocation, _ := indexMap[tile.ID{X: 0, Y: 0, Z: 0}]
	header.MutateRootOffset(rootLocation.Offset())
	header.MutateRootSize(rootLocation.Length())

	return result, nil
}
