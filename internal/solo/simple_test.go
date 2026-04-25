package solo

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/timecore"
)

func TestSimpleRuntimeStep(t *testing.T) {
	runtime := SimpleRuntime{
		Settlement: settlement.SimpleEngine{
			Deps: settlement.Dependencies{
				TimeCore: timecore.SimpleCore{},
			},
		},
	}

	session := types.SoloSession{
		SessionID:    "solo-1",
		StateVersion: 7,
		WorldClock:   types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 10},
		PlayerState:  types.PlayerState{ActorID: "u1", Stats: types.Stats{Stamina: 3}},
	}

	out, err := runtime.Step(context.Background(), session, types.PendingAction{
		ActionID:   "act-1",
		ActionType: "work",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.StateVersion != 8 {
		t.Fatalf("expected state version 8, got %d", out.StateVersion)
	}
	if out.Delta.PlayerState == nil || out.Delta.PlayerState.Stats.Stamina != 2 {
		t.Fatalf("expected stamina 2, got %#v", out.Delta.PlayerState)
	}
}

