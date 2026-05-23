package sparse

import "github.com/eak1mov/go-libtiles/tile"

func tilesCountOnZoom(zoom uint32) uint32 {
	return 1 << (2 * zoom)
}

func parentN(tileID tile.ID, zDiff uint32) tile.ID {
	return tile.ID{
		X: tileID.X >> zDiff,
		Y: tileID.Y >> zDiff,
		Z: tileID.Z - zDiff,
	}
}

func parent(tileID tile.ID) tile.ID {
	return parentN(tileID, 1)
}

func parentCode(tileCode uint32) uint32 {
	return tileCode >> 2
}

func childCode(tileCode, childIdx uint32) uint32 {
	return (tileCode << 2) + childIdx
}

// Returns a tile in the `newRoot` subtree with the same path as `tileID` in the `oldRoot` subtree.
// Result has higher bits from `newRoot`, and lower bits from `tileID`:
//
// bits      <- higher, lower ->
// oldRoot:  [OOOOOOOO]
// newRoot:  [NNNNNNNN]
// tileID:   [OOOOOOOOCCCCCCCC]
// childID:          [CCCCCCCC]
// result:   [NNNNNNNNCCCCCCCC]
func changeRoot(tileID, oldRoot, newRoot tile.ID) tile.ID {
	// `oldRoot` and `newRoot` must be on the same z level
	debugCheck(oldRoot.Z == newRoot.Z)

	childID := subtractTileIDs(tileID, oldRoot)
	return combineTileIDs(newRoot, childID)
}

// bits     <- higher, lower ->
// rootID:  [RRRRRRRR]
// tileID:  [RRRRRRRRTTTTTTTT]
// result:          [TTTTTTTT]
func subtractTileIDs(tileID, rootID tile.ID) tile.ID {
	zDiff := tileID.Z - rootID.Z

	// `tileID` must be in the `rootID` subtree (same higher bits)
	debugCheck(tileID.Z >= rootID.Z)
	debugCheck(tileID.X>>zDiff == rootID.X)
	debugCheck(tileID.Y>>zDiff == rootID.Y)

	// clean higher bits, keep lower bits
	return tile.ID{
		X: tileID.X & (1<<zDiff - 1),
		Y: tileID.Y & (1<<zDiff - 1),
		Z: zDiff,
	}
}

// bits      <- higher, lower ->
// rootID:   [RRRRRRRR]
// innerID:          [IIIIIIII]
// result:   [RRRRRRRRIIIIIIII]
func combineTileIDs(rootID, innerID tile.ID) tile.ID {
	return tile.ID{
		X: (rootID.X << innerID.Z) + innerID.X,
		Y: (rootID.Y << innerID.Z) + innerID.Y,
		Z: rootID.Z + innerID.Z,
	}
}

// Arguments must be checked by calling function, here we double-check them to
// clarify code and simplify debug.
// TODO(eak1mov): remove these checks?
func debugCheck(condition bool) {
	if !condition {
		panic(tile.Error("libtiles: internal error"))
	}
}
