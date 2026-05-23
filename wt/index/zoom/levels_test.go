package zoom

import (
	"testing"
)

func TestLevelIndex(t *testing.T) {
	testCases := []struct {
		ZoomLevels Levels
		Zoom       uint32
		LevelIdx   int
	}{
		{ZoomLevels: Levels{0, 10, 20}, Zoom: 0, LevelIdx: 0},
		{ZoomLevels: Levels{0, 10, 20}, Zoom: 1, LevelIdx: 0},
		{ZoomLevels: Levels{0, 10, 20}, Zoom: 9, LevelIdx: 0},
		{ZoomLevels: Levels{0, 10, 20}, Zoom: 10, LevelIdx: 1},
		{ZoomLevels: Levels{0, 10, 20}, Zoom: 11, LevelIdx: 1},
		{ZoomLevels: Levels{0, 10, 20}, Zoom: 19, LevelIdx: 1},
	}
	for _, tc := range testCases {
		if got, want := LevelIndex(tc.ZoomLevels, tc.Zoom), tc.LevelIdx; got != want {
			t.Errorf("LevelIndex(%v, %v) = %v, want = %v", tc.ZoomLevels, tc.Zoom, got, want)
		}
	}
}
