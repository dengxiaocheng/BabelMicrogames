package harness_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/testkit/harness"
	"babel-runtime/internal/timecore"
)

func TestMultiplayerHarnessSubmitAndCloseTurn(t *testing.T) {
	h := harness.NewMultiplayerHarness()
	h.SeedRoom(types.MultiplayerRoom{
		RoomID:       "room-1",
		Status:       types.RuntimeStatusActive,
		StateVersion: 4,
		WorldClock:   types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 1},
		HiddenState: map[string]types.PrivateState{
			"u1": {ActorID: "worker-1"},
			"u2": {ActorID: "worker-2"},
		},
		PendingTurn: &types.PendingTurn{
			TurnID:          "turn-1",
			RequiredPlayers: []string{"u1", "u2"},
			SubmittedActions: map[string]types.PendingAction{},
		},
	})

	if err := h.SubmitText(context.Background(), "room-1", "u1", "搬石头"); err != nil {
		t.Fatalf("first SubmitText returned error: %v", err)
	}
	snapshot := h.Snapshot()
	if got := len(snapshot.Events); got != 1 {
		t.Fatalf("expected 1 event after first submit, got %d", got)
	}
	if snapshot.Rooms["room-1"].PendingTurn == nil {
		t.Fatalf("expected pending turn to remain after first submit")
	}

	if err := h.SubmitText(context.Background(), "room-1", "u2", "拉绳索"); err != nil {
		t.Fatalf("second SubmitText returned error: %v", err)
	}
	snapshot = h.Snapshot()
	room := snapshot.Rooms["room-1"]
	if room.StateVersion != 5 {
		t.Fatalf("expected state version 5 after closing turn, got %d", room.StateVersion)
	}
	if room.PendingTurn != nil {
		t.Fatalf("expected pending turn cleared after close, got %#v", room.PendingTurn)
	}
	if room.LastCommittedTurnID != "turn-1" {
		t.Fatalf("expected committed turn turn-1, got %q", room.LastCommittedTurnID)
	}
	if got := len(snapshot.Checkpoints); got != 1 {
		t.Fatalf("expected 1 checkpoint after close, got %d", got)
	}
}

func TestMultiplayerHarnessSweepRecovery(t *testing.T) {
	h := harness.NewMultiplayerHarness()
	h.SeedRoom(types.MultiplayerRoom{
		RoomID:  "room-2",
		Status:  types.RuntimeStatusRecovering,
		PendingTurn: &types.PendingTurn{
			TurnID: "turn-2",
		},
	})

	report, err := h.SweepRecovery(context.Background())
	if err != nil {
		t.Fatalf("SweepRecovery returned error: %v", err)
	}
	if report.RecoveredRuntimes != 1 {
		t.Fatalf("expected 1 recoverable runtime, got %d", report.RecoveredRuntimes)
	}
}
