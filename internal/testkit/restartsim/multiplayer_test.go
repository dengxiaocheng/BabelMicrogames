package restartsim_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/testkit/harness"
	"babel-runtime/internal/testkit/restartsim"
	"babel-runtime/internal/timecore"
)

func TestMultiplayerRestartSimulatorRestoresRoomProgress(t *testing.T) {
	original := harness.NewMultiplayerHarness()
	original.SeedRoom(types.MultiplayerRoom{
		RoomID:       "room-1",
		Status:       types.RuntimeStatusActive,
		StateVersion: 9,
		WorldClock:   types.WorldClock{DayIndex: 2, Segment: timecore.SegmentMorning, AbsoluteTick: 8},
		HiddenState: map[string]types.PrivateState{
			"u1": {ActorID: "worker-1"},
			"u2": {ActorID: "worker-2"},
		},
		PendingTurn: &types.PendingTurn{
			TurnID:           "turn-9",
			RequiredPlayers:  []string{"u1", "u2"},
			SubmittedActions: map[string]types.PendingAction{},
		},
	})

	if err := original.SubmitText(context.Background(), "room-1", "u1", "先搬一轮"); err != nil {
		t.Fatalf("first SubmitText returned error: %v", err)
	}

	restored := restartsim.CaptureMultiplayer(original).Restore()
	if err := restored.SubmitText(context.Background(), "room-1", "u2", "补上第二个动作"); err != nil {
		t.Fatalf("second SubmitText returned error: %v", err)
	}

	snapshot := restored.Snapshot()
	room := snapshot.Rooms["room-1"]
	if room.StateVersion != 10 {
		t.Fatalf("expected restored state version 10, got %d", room.StateVersion)
	}
	if room.PendingTurn != nil {
		t.Fatalf("expected turn cleared after restored close, got %#v", room.PendingTurn)
	}
	if room.LastCommittedTurnID != "turn-9" {
		t.Fatalf("expected committed turn turn-9, got %q", room.LastCommittedTurnID)
	}
	if len(snapshot.Events) != 2 {
		t.Fatalf("expected 2 events after restore, got %d", len(snapshot.Events))
	}
	if len(snapshot.Checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint after restored close, got %d", len(snapshot.Checkpoints))
	}
}
