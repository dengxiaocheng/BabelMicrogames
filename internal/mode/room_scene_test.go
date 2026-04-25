package mode_test

import (
	"context"
	"testing"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/mode"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/timecore"
)

func TestRoomSceneModuleSubmitsPendingActionWithoutClosingTurn(t *testing.T) {
	module := mode.RoomSceneModule{
		Settlement: settlement.SimpleEngine{
			Deps: settlement.Dependencies{
				TimeCore: timecore.SimpleCore{},
			},
		},
		Now: func() time.Time {
			return time.Unix(1700000300, 0)
		},
	}

	snapshot, err := types.EncodeRuntimeSnapshot(
		"room-1",
		types.ModeRoomScene,
		5,
		1,
		types.MultiplayerRoom{
			RoomID:        "room-1",
			SchemaVersion: 1,
			StateVersion:  5,
			Status:        types.RuntimeStatusActive,
			WorldClock:    types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 3},
			PendingTurn: &types.PendingTurn{
				TurnID:           "turn-1",
				RequiredPlayers:  []string{"u1", "u2"},
				SubmittedActions: map[string]types.PendingAction{},
			},
		},
		time.Unix(1700000000, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}

	result, err := module.Execute(context.Background(), types.ModeExecutionInput{
		Runtime: types.RuntimeRecord{
			RuntimeID:   "room-1",
			ModeID:      types.ModeRoomScene,
			HeadVersion: 5,
		},
		Execution: types.ExecutionRecord{
			ExecutionID:    "exec-1",
			RuntimeID:      "room-1",
			ActorID:        "u1",
			IdempotencyKey: "idem-1",
			ModeID:         types.ModeRoomScene,
			Stage:          types.ExecutionPlanned,
		},
		Snapshot: &snapshot,
		Command: types.ModeCommand{
			CommandType: "room.user_text",
			Text:        "搬石头",
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Snapshot == nil {
		t.Fatalf("expected updated snapshot")
	}
	updated, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](*result.Snapshot)
	if err != nil {
		t.Fatalf("DecodeRuntimeSnapshot returned error: %v", err)
	}
	if updated.StateVersion != 5 {
		t.Fatalf("expected state version to remain 5 before turn close, got %d", updated.StateVersion)
	}
	if updated.PendingTurn == nil {
		t.Fatalf("expected pending turn to remain")
	}
	if _, ok := updated.PendingTurn.SubmittedActions["u1"]; !ok {
		t.Fatalf("expected submitted action for u1")
	}
}

func TestRoomSceneModuleClosesTurnWhenReady(t *testing.T) {
	module := mode.RoomSceneModule{
		Settlement: settlement.SimpleEngine{
			Deps: settlement.Dependencies{
				TimeCore: timecore.SimpleCore{},
			},
		},
		Now: func() time.Time {
			return time.Unix(1700000400, 0)
		},
	}

	snapshot, err := types.EncodeRuntimeSnapshot(
		"room-1",
		types.ModeRoomScene,
		5,
		1,
		types.MultiplayerRoom{
			RoomID:        "room-1",
			SchemaVersion: 1,
			StateVersion:  5,
			Status:        types.RuntimeStatusActive,
			WorldClock:    types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 3},
			HiddenState: map[string]types.PrivateState{
				"u1": {ActorID: "worker-1"},
				"u2": {ActorID: "worker-2"},
			},
			PendingTurn: &types.PendingTurn{
				TurnID:          "turn-1",
				RequiredPlayers: []string{"u1", "u2"},
				SubmittedActions: map[string]types.PendingAction{
					"u1": {
						ActionID:   "existing-1",
						UserInput:  "先占位",
						ActionType: "free_text",
					},
				},
			},
		},
		time.Unix(1700000000, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}

	result, err := module.Execute(context.Background(), types.ModeExecutionInput{
		Runtime: types.RuntimeRecord{
			RuntimeID:   "room-1",
			ModeID:      types.ModeRoomScene,
			HeadVersion: 5,
		},
		Execution: types.ExecutionRecord{
			ExecutionID:    "exec-2",
			RuntimeID:      "room-1",
			ActorID:        "u2",
			IdempotencyKey: "idem-2",
			ModeID:         types.ModeRoomScene,
			Stage:          types.ExecutionPlanned,
		},
		Snapshot: &snapshot,
		Command: types.ModeCommand{
			CommandType: "room.user_text",
			Text:        "拉绳索",
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	updated, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](*result.Snapshot)
	if err != nil {
		t.Fatalf("DecodeRuntimeSnapshot returned error: %v", err)
	}
	if updated.StateVersion != 6 {
		t.Fatalf("expected state version 6 after close, got %d", updated.StateVersion)
	}
	if updated.PendingTurn != nil {
		t.Fatalf("expected pending turn cleared after close")
	}
	if updated.LastCommittedTurnID != "turn-1" {
		t.Fatalf("expected last committed turn turn-1, got %q", updated.LastCommittedTurnID)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 room events, got %d", len(result.Events))
	}
}
