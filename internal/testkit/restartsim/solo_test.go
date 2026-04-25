package restartsim_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/testkit/harness"
	"babel-runtime/internal/testkit/restartsim"
	"babel-runtime/internal/timecore"
)

func TestSoloRestartSimulatorRestoresProgress(t *testing.T) {
	original := harness.NewSoloHarness()
	original.SeedSession(types.SoloSession{
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

	if err := original.SubmitText(context.Background(), "solo-1", "user-1", "继续干活"); err != nil {
		t.Fatalf("first SubmitText returned error: %v", err)
	}

	restored := restartsim.CaptureSolo(original).Restore()
	if err := restored.SubmitText(context.Background(), "solo-1", "user-1", "继续前进"); err != nil {
		t.Fatalf("second SubmitText returned error: %v", err)
	}

	snapshot := restored.Snapshot()
	session := snapshot.SoloSessions["solo-1"]
	if session.StateVersion != 3 {
		t.Fatalf("expected restored state version 3, got %d", session.StateVersion)
	}
	if session.PlayerState.Stats.Stamina != 2 {
		t.Fatalf("expected stamina 2 after two steps, got %d", session.PlayerState.Stats.Stamina)
	}
	if len(snapshot.Events) != 2 {
		t.Fatalf("expected 2 events after restore, got %d", len(snapshot.Events))
	}
	if len(snapshot.Checkpoints) != 2 {
		t.Fatalf("expected 2 checkpoints after restore, got %d", len(snapshot.Checkpoints))
	}
}
