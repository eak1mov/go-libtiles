package sparse

import (
	"cmp"
	"slices"

	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
	flatbuffers "github.com/google/flatbuffers/go"
)

type sparseLocations struct {
	Tiles []locationItem
	Links []linkItem
}

type locationItem struct {
	TileCode uint32
	Location uint64
}

type linkItem struct {
	TileCode uint32
	LinkCode uint32
}

func readDense(block *fbs.SparseBlock) [][]packed.Location {
	result := make([][]packed.Location, block.DenseLocationsLength())

	for z := range len(result) {
		denseLocations := fbs.DenseLocations{}
		block.DenseLocations(&denseLocations, z)

		result[z] = make([]packed.Location, denseLocations.LocationsLength())

		for tileCode := range len(result[z]) {
			fbsLocation := fbs.Location{}
			denseLocations.Locations(&fbsLocation, tileCode)

			result[z][tileCode] = packed.ReadFbs(fbsLocation)
		}
	}

	return result
}

func readSparse(block *fbs.SparseBlock) []sparseLocations {
	result := make([]sparseLocations, block.SparseLocationsLength())

	for z := range len(result) {
		locations := fbs.SparseLocations{}
		block.SparseLocations(&locations, z)

		result[z].Tiles = make([]locationItem, locations.TilesLength())
		result[z].Links = make([]linkItem, locations.LinksLength())

		for i := range locations.TilesLength() {
			item := fbs.LocationItem{}
			locations.Tiles(&item, i)

			fbsLocation := fbs.Location{}
			item.Location(&fbsLocation)

			result[z].Tiles[i] = locationItem{
				TileCode: item.TileCode(),
				Location: uint64(packed.ReadFbs(fbsLocation)),
			}
		}

		for i := range locations.LinksLength() {
			item := fbs.LinkItem{}
			locations.Links(&item, i)

			result[z].Links[i] = linkItem{
				TileCode: item.TileCode(),
				LinkCode: item.LinkCode(),
			}
		}
	}

	return result
}

func writeDense(blockLocations [][]packed.Location) []byte {
	builder := flatbuffers.NewBuilder(0)

	offsets := make([]flatbuffers.UOffsetT, len(blockLocations))

	for z, locations := range slices.Backward(blockLocations) {
		fbs.DenseLocationsStartLocationsVector(builder, len(locations))
		for _, location := range slices.Backward(locations) {
			// TODO(eak1mov): try to use builder.PrependUint64(uint64(location))
			v0, v1 := packed.WriteFbs(packed.Location(location))
			fbs.CreateLocation(builder, v0, v1)
		}
		locationsOffset := builder.EndVector(len(locations))

		fbs.DenseLocationsStart(builder)
		fbs.DenseLocationsAddLocations(builder, locationsOffset)
		offsets[z] = fbs.DenseLocationsEnd(builder)
	}

	fbs.SparseBlockStartDenseLocationsVector(builder, len(offsets))
	for _, offset := range slices.Backward(offsets) {
		builder.PrependUOffsetT(offset)
	}
	denseLocationsOffset := builder.EndVector(len(offsets))

	fbs.SparseBlockStart(builder)
	fbs.SparseBlockAddBlockType(builder, fbs.BlockTypeDense)
	fbs.SparseBlockAddDenseLocations(builder, denseLocationsOffset)
	builder.Finish(fbs.SparseBlockEnd(builder))

	return builder.FinishedBytes()
}

func writeSparse(block []sparseLocations) []byte {
	builder := flatbuffers.NewBuilder(0)

	offsets := make([]flatbuffers.UOffsetT, len(block))

	for z, locations := range slices.Backward(block) {
		fbs.SparseLocationsStartTilesVector(builder, len(locations.Tiles))
		for _, locationItem := range slices.Backward(locations.Tiles) {
			v0, v1 := packed.WriteFbs(packed.Location(locationItem.Location))
			fbs.CreateLocationItem(builder, locationItem.TileCode, v0, v1)
		}
		tilesOffset := builder.EndVector(len(locations.Tiles))

		fbs.SparseLocationsStartLinksVector(builder, len(locations.Links))
		for _, linkItem := range slices.Backward(locations.Links) {
			fbs.CreateLinkItem(builder, linkItem.TileCode, linkItem.LinkCode)
		}
		linksOffset := builder.EndVector(len(locations.Links))

		fbs.SparseLocationsStart(builder)
		fbs.SparseLocationsAddTiles(builder, tilesOffset)
		fbs.SparseLocationsAddLinks(builder, linksOffset)
		offsets[z] = fbs.SparseLocationsEnd(builder)
	}

	fbs.SparseBlockStartSparseLocationsVector(builder, len(offsets))
	for _, offset := range slices.Backward(offsets) {
		builder.PrependUOffsetT(offset)
	}
	sparseLocationsOffset := builder.EndVector(len(offsets))

	fbs.SparseBlockStart(builder)
	fbs.SparseBlockAddBlockType(builder, fbs.BlockTypeSparse)
	fbs.SparseBlockAddSparseLocations(builder, sparseLocationsOffset)
	builder.Finish(fbs.SparseBlockEnd(builder))

	return builder.FinishedBytes()
}

func validateDense(blockLocations [][]packed.Location, zoomCount uint32) bool {
	if len(blockLocations) != int(zoomCount) {
		return false
	}
	for z, locations := range blockLocations {
		if len(locations) != int(tilesCountOnZoom(uint32(z))) {
			return false
		}
	}
	return true
}

func validateSparse(block []sparseLocations, zoomCount uint32) bool {
	if len(block) != int(zoomCount) {
		return false
	}
	for _, locations := range block {
		if !slices.IsSortedFunc(locations.Tiles, func(a, b locationItem) int {
			return cmp.Compare(a.TileCode, b.TileCode)
		}) {
			return false
		}
		if !slices.IsSortedFunc(locations.Links, func(a, b linkItem) int {
			return cmp.Compare(a.TileCode, b.TileCode)
		}) {
			return false
		}
	}
	return true
}
