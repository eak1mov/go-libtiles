package zoom

import "slices"

type Levels []uint32

func LevelIndex(levels Levels, zoom uint32) int {
	return slices.IndexFunc(levels, func(zlevel uint32) bool {
		return zlevel > zoom
	}) - 1
}

func LevelsFromMask(mask uint64) Levels {
	levels := make([]uint32, 0)
	for z := uint32(0); z <= 30; z++ {
		if mask&(1<<z) != 0 {
			levels = append(levels, z)
		}
	}
	return levels
}

func LevelsToMask(levels Levels) uint64 {
	mask := uint64(0)
	for _, z := range levels {
		mask |= 1 << z
	}
	return mask
}
