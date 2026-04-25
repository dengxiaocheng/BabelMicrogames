package multiplayer_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/mapcore"
	"babel-runtime/internal/multiplayer"
	"babel-runtime/internal/relcore"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/timecore"
)

func TestSimpleRuntimeCloseTurn(t *testing.T) {
	runtime := multiplayer.SimpleRuntime{
		Settlement: settlement.SimpleEngine{
			Deps: settlement.Dependencies{
				TimeCore: timecore.SimpleCore{},
				MapCore:  mapcore.NewSimpleCore(),
				RelCore: relcore.SimpleCore{},
			},
		},
	}

	updated, result, err := runtime.CloseTurn(context.Background(), types.MultiplayerRoom{
		RoomID:       "room-1",
		StateVersion: 7,
		WorldClock:   types.WorldClock{DayIndex: 1, Segment: "dawn", AbsoluteTick: 1},
		Roster: []types.RoomMember{
			{PlayerID: "p1", DisplayName: "甲"},
			{PlayerID: "p2", DisplayName: "乙"},
		},
		HiddenState: map[string]types.PrivateState{
			"p1": {ActorID: "worker-1"},
			"p2": {ActorID: "worker-2"},
		},
		PendingTurn: &types.PendingTurn{
			TurnID: "turn-1",
			SubmittedActions: map[string]types.PendingAction{
				"p1": {ActionID: "a1", UserInput: "搬石块", ActionType: "work"},
				"p2": {ActionID: "a2", UserInput: "拉绳索", ActionType: "assist"},
			},
		},
	})
	if err != nil {
		t.Fatalf("CloseTurn returned error: %v", err)
	}

	if result.RoomID != "room-1" {
		t.Fatalf("expected room-1, got %q", result.RoomID)
	}
	if result.StateVersion != 8 {
		t.Fatalf("expected state version 8, got %d", result.StateVersion)
	}
	if updated.StateVersion != 8 {
		t.Fatalf("expected updated room state version 8, got %d", updated.StateVersion)
	}
	if updated.PendingTurn != nil {
		t.Fatalf("expected pending turn cleared, got %#v", updated.PendingTurn)
	}
	if result.Delta.NextClock.Segment != timecore.SegmentMorning {
		t.Fatalf("expected advanced clock, got %#v", result.Delta.NextClock)
	}
	if result.Delta.SharedState == nil || result.Delta.SharedState.AggregateMetrics["submitted_actions"] != 2 {
		t.Fatalf("expected shared metric for submitted actions, got %#v", result.Delta.SharedState)
	}
}
