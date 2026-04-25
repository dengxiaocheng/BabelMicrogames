package settlement

import (
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/relcore"
	"babel-runtime/internal/timecore"
)

func TestStepSoloAdvancesClockAndConsumesStamina(t *testing.T) {
	engine := SimpleEngine{
		Deps: Dependencies{
			TimeCore: timecore.SimpleCore{},
			RelCore:  relcore.SimpleCore{},
		},
	}

	out, err := engine.StepSolo(types.SoloStepInput{
		Session: types.SoloSession{
			SessionID:  "s1",
			WorldClock: types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 10},
			PlayerState: types.PlayerState{
				ActorID: "p1",
				Stats:   types.Stats{Stamina: 3},
			},
		},
		Action: types.PendingAction{
			ActionID:   "a1",
			ActionType: "work",
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.NextClock.Segment != timecore.SegmentNoon {
		t.Fatalf("expected noon, got %s", out.NextClock.Segment)
	}
	if out.PlayerState == nil || out.PlayerState.Stats.Stamina != 2 {
		t.Fatalf("expected stamina 2, got %#v", out.PlayerState)
	}
}

func TestStepRoomAdvancesClockAndCountsActions(t *testing.T) {
	engine := SimpleEngine{
		Deps: Dependencies{
			TimeCore: timecore.SimpleCore{},
		},
	}

	out, err := engine.StepRoom(types.RoomStepInput{
		Room: types.MultiplayerRoom{
			RoomID:     "r1",
			WorldClock: types.WorldClock{DayIndex: 1, Segment: timecore.SegmentAfternoon, AbsoluteTick: 20},
			SharedState: types.SharedSceneState{
				AggregateMetrics: map[string]int{},
			},
		},
		Actions: []types.PendingAction{
			{ActionID: "a1", ActionType: "work"},
			{ActionID: "a2", ActionType: "rest"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.NextClock.Segment != timecore.SegmentDusk {
		t.Fatalf("expected dusk, got %s", out.NextClock.Segment)
	}
	if out.SharedState == nil || out.SharedState.AggregateMetrics["submitted_actions"] != 2 {
		t.Fatalf("expected submitted_actions=2, got %#v", out.SharedState)
	}
}

