package timecore

import "babel-runtime/internal/core/types"

const (
	SegmentDawn      = "dawn"
	SegmentMorning   = "morning"
	SegmentNoon      = "noon"
	SegmentAfternoon = "afternoon"
	SegmentDusk      = "dusk"
	SegmentNight     = "night"
)

type SegmentRule struct {
	Segment            string           `json:"segment"`
	AllowedActionTags  []string         `json:"allowed_action_tags"`
	DefaultModifiers   []types.Modifier `json:"default_modifiers,omitempty"`
	NextSegment        string           `json:"next_segment"`
}

