package morton

import "github.com/eak1mov/go-libtiles/tile"

func interleave(x uint32) uint32 {
	x = (x | (x << 8)) & 0x00FF00FF
	x = (x | (x << 4)) & 0x0F0F0F0F
	x = (x | (x << 2)) & 0x33333333
	x = (x | (x << 1)) & 0x55555555
	return x
}

func deinterleave(x uint32) uint32 {
	x = x & 0x55555555
	x = (x | (x >> 1)) & 0x33333333
	x = (x | (x >> 2)) & 0x0F0F0F0F
	x = (x | (x >> 4)) & 0x00FF00FF
	x = (x | (x >> 8)) & 0x0000FFFF
	return x
}

func Encode(tileID tile.ID) uint32 {
	return interleave(tileID.X) | (interleave(tileID.Y) << 1)
}

func Decode(tileCode, z uint32) tile.ID {
	return tile.ID{
		X: deinterleave(tileCode),
		Y: deinterleave(tileCode >> 1),
		Z: z,
	}
}
