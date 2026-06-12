package block

import (
	"iter"
	"math/bits"
)

// [Start, Start+Count)
type ZoomRange struct {
	Start uint32
	Count uint32
}

func (r ZoomRange) End() uint32 {
	return r.Start + r.Count
}

type LevelsMask uint32

func NewLevelsMask(zooms ...uint32) LevelsMask {
	mask := LevelsMask(0)
	for _, zoom := range zooms {
		mask |= 1 << zoom
	}
	return mask
}

func (m LevelsMask) zoomCount() int {
	return bits.OnesCount32(uint32(m))
}

func (m LevelsMask) zooms() iter.Seq[uint32] {
	return func(yield func(uint32) bool) {
		mask := uint32(m)
		for mask != 0 {
			zoom := uint32(bits.TrailingZeros32(mask))
			if !yield(zoom) {
				return
			}
			mask &^= 1 << zoom
		}
	}
}

func (m LevelsMask) RangesCount() int {
	return m.zoomCount() - 1
}

func (m LevelsMask) Ranges() iter.Seq[ZoomRange] {
	return func(yield func(ZoomRange) bool) {
		first := true
		var prevZoom uint32
		for nextZoom := range m.zooms() {
			if !first && !yield(ZoomRange{Start: prevZoom, Count: nextZoom - prevZoom}) {
				return
			}
			first = false
			prevZoom = nextZoom
		}
	}
}

func (m LevelsMask) FindRange(zoom uint32) ZoomRange {
	for r := range m.Ranges() {
		if r.Start <= zoom && zoom < r.End() {
			return r
		}
	}
	panic("invalid zoom or LevelsMask")
}
