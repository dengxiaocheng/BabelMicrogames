package eventlog_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/eventlog"
	"babel-runtime/internal/testkit/harness"
	"babel-runtime/internal/timecore"
)

func TestReplayerReplaySolo(t *testing.T) {
	h := harness.NewSoloHarness()
	h.SeedSession(types.SoloSession{
		SessionID:    "solo-1",
		UserID:       "user-1",
		Status:       types.RuntimeStatusActive,
		StateVersion: 1,
		WorldClock:   types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 1},
		PlayerState: types.PlayerState{
			ActorID: "worker-1",
			Stats:   types.Stats{Stamina: 4},
		},
	})
	if err := h.SubmitText(context.Background(), "solo-1", "user-1", "继续干活"); err != nil {
		t.Fatalf("SubmitText returned error: %v", err)
	}

	replay, err := (eventlog.Replayer{Store: h.Store}).ReplaySolo(context.Background(), "solo-1")
	if err != nil {
		t.Fatalf("ReplaySolo returned error: %v", err)
	}
	if replay.Session.StateVersion != 2 {
		t.Fatalf("expected state version 2, got %d", replay.Session.StateVersion)
	}
	if len(replay.History.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(replay.History.Events))
	}
	if len(replay.History.Checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(replay.History.Checkpoints))
	}
	if len(replay.History.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(replay.History.Frames))
	}
}

func TestReplayerReplayRoom(t *testing.T) {
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
			TurnID:           "turn-1",
			RequiredPlayers:  []string{"u1", "u2"},
			SubmittedActions: map[string]types.PendingAction{},
		},
	})
	if err := h.SubmitText(context.Background(), "room-1", "u1", "搬石头"); err != nil {
		t.Fatalf("first SubmitText returned error: %v", err)
	}
	if err := h.SubmitText(context.Background(), "room-1", "u2", "拉绳索"); err != nil {
		t.Fatalf("second SubmitText returned error: %v", err)
	}

	replay, err := (eventlog.Replayer{Store: h.Store}).ReplayRoom(context.Background(), "room-1")
	if err != nil {
		t.Fatalf("ReplayRoom returned error: %v", err)
	}
	if replay.Room.StateVersion != 5 {
		t.Fatalf("expected state version 5, got %d", replay.Room.StateVersion)
	}
	if len(replay.History.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(replay.History.Events))
	}
	if len(replay.History.Checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(replay.History.Checkpoints))
	}
}
