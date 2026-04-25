package timecore

import (
	"testing"

	"babel-runtime/internal/core/types"
)

func TestAdvanceWrapsToNextDay(t *testing.T) {
	core := SimpleCore{}
	clock := types.WorldClock{DayIndex: 3, Segment: SegmentNight, AbsoluteTick: 99}
	next := core.Advance(clock)
	if next.DayIndex != 4 {
		t.Fatalf("expected day 4, got %d", next.DayIndex)
	}
	if next.Segment != SegmentDawn {
		t.Fatalf("expected dawn, got %s", next.Segment)
	}
	if next.AbsoluteTick != 100 {
		t.Fatalf("expected tick 100, got %d", next.AbsoluteTick)
	}
}

