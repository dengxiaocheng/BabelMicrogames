package recovery

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/store"
)

func TestSimpleSupervisorSweep(t *testing.T) {
	mem := store.NewMemoryStore()
	mem.SeedSoloSession(types.SoloSession{
		SessionID: "solo-1",
		Status:    types.RuntimeStatusActive,
		PendingAction: &types.PendingAction{
			ActionID: "a1",
		},
	})
	mem.SeedRoom(types.MultiplayerRoom{
		RoomID:  "room-1",
		Status:  types.RuntimeStatusRecovering,
		PendingTurn: &types.PendingTurn{
			TurnID: "turn-1",
		},
	})
	if err := mem.RunTx(context.Background(), func(ctx context.Context, tx store.TxStore) error {
		if err := tx.SaveCheckpoint(ctx, types.RuntimeCheckpoint{
			CheckpointID: "cp-1",
			Status:       types.CheckpointSimulating,
			ResumePolicy: types.ResumeRetryStep,
		}); err != nil {
			return err
		}
		return tx.SaveCheckpoint(ctx, types.RuntimeCheckpoint{
			CheckpointID: "cp-2",
			Status:       types.CheckpointRendering,
			ResumePolicy: types.ResumeRedeliver,
		})
	}); err != nil {
		t.Fatalf("seed checkpoints: %v", err)
	}

	s := SimpleSupervisor{Store: mem}
	out, err := s.Sweep(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.RecoveredRuntimes != 2 {
		t.Fatalf("expected recoverable runtimes 2, got %d", out.RecoveredRuntimes)
	}
	if out.StaleCheckpoints != 2 {
		t.Fatalf("expected stale checkpoints 2, got %d", out.StaleCheckpoints)
	}
	if out.RetriedSteps != 1 {
		t.Fatalf("expected retried steps 1, got %d", out.RetriedSteps)
	}
	if out.RedeliveredFrames != 1 {
		t.Fatalf("expected redelivered frames 1, got %d", out.RedeliveredFrames)
	}
}
