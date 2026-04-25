package timecore

import (
	"fmt"

	"babel-runtime/internal/core/types"
)

var segmentOrder = []string{
	SegmentDawn,
	SegmentMorning,
	SegmentNoon,
	SegmentAfternoon,
	SegmentDusk,
	SegmentNight,
}

type SimpleCore struct{}

func (SimpleCore) Current(clock types.WorldClock) SegmentRule {
	next := SegmentDawn
	for i, segment := range segmentOrder {
		if segment == clock.Segment {
			next = segmentOrder[(i+1)%len(segmentOrder)]
			break
		}
	}
	return SegmentRule{
		Segment:     clock.Segment,
		NextSegment: next,
	}
}

func (SimpleCore) ValidateAction(clock types.WorldClock, action types.ActionSpec) error {
	if clock.Segment == "" {
		return fmt.Errorf("missing segment")
	}
	if action.ActionType == "" {
		return fmt.Errorf("missing action type")
	}
	return nil
}

func (s SimpleCore) Advance(clock types.WorldClock) types.WorldClock {
	rule := s.Current(clock)
	next := clock
	next.AbsoluteTick++
	next.Segment = rule.NextSegment
	if rule.NextSegment == SegmentDawn {
		next.DayIndex++
	}
	return next
}

