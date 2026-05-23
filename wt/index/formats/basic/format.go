package basic

import (
	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/eak1mov/go-libtiles/wt/index"
	"github.com/eak1mov/go-libtiles/wt/index/internal/morton"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
)

const MaxZoom = 15

func Size(zoomCount uint32) uint64 {
	// = 4^0 + 4^1 + ... + 4^(zoomCount-1)
	return ((1 << (2 * zoomCount)) - 1) / 3
}

func QueryLocation(tileID tile.ID) tile.Location {
	return tile.Location{
		// size of all previous zooms + location on current zoom
		Offset: (Size(tileID.Z) + uint64(morton.Encode(tileID))) * packed.LocationLength,
		Length: packed.LocationLength,
	}
}

func Query(header *fbs.IndexHeader, tileID tile.ID, indexAccess index.FileAccessFunc) (tile.Location, error) {
	if tileID.Z > uint32(header.MaxZoom()) {
		return tile.Location{}, nil
	}

	location := QueryLocation(tileID)

	locationData, err := indexAccess(location.Offset, location.Length)
	if err != nil {
		return tile.Location{}, err
	}

	tileLocation := packed.Unpack(packed.Read(locationData))

	return tileLocation, nil
}

func Write(header *fbs.IndexHeader, indexMap index.Map) ([]byte, error) {
	maxZoom := uint32(0)
	for tileID := range indexMap {
		maxZoom = max(maxZoom, tileID.Z)
	}

	header.MutateMagic(fbs.IndexMagicValue)
	header.MutateFormat(fbs.IndexFormatBasicPlain)
	header.MutateMaxZoom(uint64(maxZoom))
	header.MutateBlockLevelsMask(uint64(1<<(maxZoom+2) - 1))

	result := make([]byte, packed.LocationLength*Size(maxZoom+1))

	for tileID, tileLocation := range indexMap {
		location := QueryLocation(tileID)
		locationData := result[location.Offset:][:location.Length]
		packed.Write(locationData, tileLocation)
	}

	return result, nil
}

func Read(header *fbs.IndexHeader, indexData []byte) (index.Map, error) {
	maxZoom := uint32(header.MaxZoom())

	result := make(index.Map, len(indexData)/packed.LocationLength)

	for z := range maxZoom + 1 {
		for x := range uint32(1 << z) {
			for y := range uint32(1 << z) {
				tileID := tile.ID{X: x, Y: y, Z: z}

				location := QueryLocation(tileID)
				locationData := indexData[location.Offset:][:location.Length]

				tileLocation := packed.Read(locationData)

				if tileLocation.Length() != 0 {
					result[tileID] = tileLocation
				}
			}
		}
	}

	return result, nil
}
