package plain

import (
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/eak1mov/go-libtiles/wt/index"
	"github.com/eak1mov/go-libtiles/wt/index/internal/morton"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
	"github.com/eak1mov/go-libtiles/wt/index/zoom"
)

const MaxZoom = 15

func calcBlockLength(zoomCount uint32) uint64 {
	// = 4^0 + 4^1 + ... + 4^(zoomCount-1)
	return ((1 << (2 * zoomCount)) - 1) / 3
}

type BlockLocation struct {
	Block tile.Location
	Inner tile.Location
}

func QueryBlock(tileID tile.ID, blockLevels zoom.Levels) BlockLocation {
	blockIdx := zoom.LevelIndex(blockLevels, tileID.Z)
	blockZ := blockLevels[blockIdx]
	innerZ := tileID.Z - blockZ
	innerZCount := blockLevels[blockIdx+1] - blockLevels[blockIdx]

	blockTileID := tile.ID{
		X: tileID.X >> innerZ,
		Y: tileID.Y >> innerZ,
		Z: tileID.Z - innerZ,
	}
	innerTileID := tile.ID{
		X: tileID.X & ((1 << innerZ) - 1),
		Y: tileID.Y & ((1 << innerZ) - 1),
		Z: innerZ,
	}

	blockCode := uint64(morton.Encode(blockTileID))
	blockLength := calcBlockLength(innerZCount)
	blockOffset := calcBlockLength(blockZ) + blockCode*blockLength

	innerCode := uint64(morton.Encode(innerTileID))
	innerLength := uint64(1)
	innerOffset := calcBlockLength(innerZ) + innerCode*innerLength

	return BlockLocation{
		Block: tile.Location{
			Offset: blockOffset * packed.LocationLength,
			Length: blockLength * packed.LocationLength,
		},
		Inner: tile.Location{
			Offset: innerOffset * packed.LocationLength,
			Length: innerLength * packed.LocationLength,
		},
	}
}

func Query(header *fbs.IndexHeader, tileID tile.ID, indexAccess index.FileAccessFunc) (tile.Location, error) {
	if tileID.Z > uint32(header.MaxZoom()) {
		return tile.Location{}, nil
	}

	blockLevels := zoom.LevelsFromMask(header.BlockLevelsMask())
	location := QueryBlock(tileID, blockLevels)

	blockData, err := indexAccess(location.Block.Offset, location.Block.Length)
	if err != nil {
		return tile.Location{}, err
	}

	locationData := blockData[location.Inner.Offset:][:location.Inner.Length]
	tileLocation := packed.Unpack(packed.Read(locationData))

	return tileLocation, nil
}

func Write(header *fbs.IndexHeader, indexMap index.Map) ([]byte, error) {
	maxZoom := uint32(0)
	for tileID := range indexMap {
		maxZoom = max(maxZoom, tileID.Z)
	}

	var blockLevels zoom.Levels
	if maxZoom <= 8 {
		blockLevels = []uint32{0, maxZoom + 1}
	} else {
		blockLevels = []uint32{0, maxZoom / 2, maxZoom + 1}
	}

	header.MutateMagic(fbs.IndexMagicValue)
	header.MutateFormat(fbs.IndexFormatPlain)
	header.MutateMaxZoom(uint64(maxZoom))
	header.MutateBlockLevelsMask(zoom.LevelsToMask(blockLevels))

	result := make([]byte, packed.LocationLength*calcBlockLength(maxZoom+1))

	for tileID, tileLocation := range indexMap {
		location := QueryBlock(tileID, blockLevels)
		locationOffset := location.Block.Offset + location.Inner.Offset
		locationLength := location.Inner.Length
		locationData := result[locationOffset:][:locationLength]

		packed.Write(locationData, tileLocation)
	}

	return result, nil
}

func Read(header *fbs.IndexHeader, indexData []byte) (index.Map, error) {
	maxZoom := uint32(header.MaxZoom())
	blockLevels := zoom.LevelsFromMask(header.BlockLevelsMask())

	result := make(index.Map, len(indexData)/packed.LocationLength)

	for z := range maxZoom + 1 {
		for x := range uint32(1 << z) {
			for y := range uint32(1 << z) {
				tileID := tile.ID{X: x, Y: y, Z: z}

				location := QueryBlock(tileID, blockLevels)
				locationOffset := location.Block.Offset + location.Inner.Offset
				locationLength := location.Inner.Length
				locationData := indexData[locationOffset:][:locationLength]

				tileLocation := packed.Read(locationData)

				if tileLocation.Length() != 0 {
					result[tileID] = tileLocation
				}
			}
		}
	}

	return result, nil
}
