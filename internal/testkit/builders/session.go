package builders

import (
	"babel-runtime/internal/core/types"
	"babel-runtime/internal/timecore"
)

func SoloSession(sessionID, actorID string) types.SoloSession {
	return types.SoloSession{
		SessionID:  sessionID,
		UserID:     actorID,
		Status:     types.RuntimeStatusActive,
		WorldClock: types.WorldClock{DayIndex: 1, Segment: timecore.SegmentDawn, AbsoluteTick: 0},
		PlayerState: types.PlayerState{
			ActorID: actorID,
			Stats:   types.Stats{Stamina: 5, Spirit: 5, Satiety: 5, Health: 5},
			Location: types.ActorLocation{
				ZoneID: "camp",
			},
		},
		EventFlags: map[string]types.FlagValue{},
	}
}

