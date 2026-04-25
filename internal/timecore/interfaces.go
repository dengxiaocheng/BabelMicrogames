package timecore

import "babel-runtime/internal/core/types"

type Core interface {
	Current(clock types.WorldClock) SegmentRule
	ValidateAction(clock types.WorldClock, action types.ActionSpec) error
	Advance(clock types.WorldClock) types.WorldClock
}

