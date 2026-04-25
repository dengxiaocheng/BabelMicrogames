package harness_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/testkit/harness"
	"babel-runtime/internal/timecore"
)

func TestSoloHarnessSubmitText(t *testing.T) {
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

	snapshot := h.Snapshot()
	if len(snapshot.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(snapshot.Events))
	}
	if len(snapshot.Checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(snapshot.Checkpoints))
	}
	if len(snapshot.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(snapshot.Frames))
	}
}

func TestSoloHarnessSweepRecovery(t *testing.T) {
	h := harness.NewSoloHarness()
	h.SeedSession(types.SoloSession{
		SessionID: "solo-2",
		UserID:    "user-2",
		Status:    types.RuntimeStatusRecovering,
		PendingAction: &types.PendingAction{
			ActionID: "pending-1",
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
