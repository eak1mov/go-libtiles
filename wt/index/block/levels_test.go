package block_test

import (
	"slices"
	"testing"

	"github.com/eak1mov/go-libtiles/wt/index/block"
	"github.com/google/go-cmp/cmp"
)

func TestNewLevelsMask(t *testing.T) {
	if got, want := block.NewLevelsMask(0, 2, 3), 0b1101; uint32(got) != uint32(want) {
		t.Errorf("NewLevelsMask(0, 2, 3) = %b, want = %b", got, want)
	}
}

func TestLevelsMaskRanges(t *testing.T) {
	levels := []uint32{0, 2, 3}
	got := slices.Collect(block.NewLevelsMask(levels...).Ranges())
	want := []block.ZoomRange{{Start: 0, Count: 2}, {Start: 2, Count: 1}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LevelsMask(%v).Ranges() mismatch (-want +got):\n%s", levels, diff)
	}
}

func TestLevelsMaskFindRange(t *testing.T) {
	testCases := []struct {
		Levels []uint32
		Zoom   uint32
		Range  block.ZoomRange
	}{
		{Levels: []uint32{0, 10, 20}, Zoom: 0, Range: block.ZoomRange{Start: 0, Count: 10}},
		{Levels: []uint32{0, 10, 20}, Zoom: 1, Range: block.ZoomRange{Start: 0, Count: 10}},
		{Levels: []uint32{0, 10, 20}, Zoom: 9, Range: block.ZoomRange{Start: 0, Count: 10}},
		{Levels: []uint32{0, 10, 20}, Zoom: 10, Range: block.ZoomRange{Start: 10, Count: 10}},
		{Levels: []uint32{0, 10, 20}, Zoom: 11, Range: block.ZoomRange{Start: 10, Count: 10}},
		{Levels: []uint32{0, 10, 20}, Zoom: 19, Range: block.ZoomRange{Start: 10, Count: 10}},
	}
	for _, tc := range testCases {
		mask := block.NewLevelsMask(tc.Levels...)
		if got, want := mask.FindRange(tc.Zoom), tc.Range; got != want {
			t.Errorf("FindRange(%v, %v) = %v, want = %v", tc.Levels, tc.Zoom, got, want)
		}
	}
}
