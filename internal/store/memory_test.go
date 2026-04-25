package store_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/store"
)

func TestMemoryStorePersistsTxWrites(t *testing.T) {
	s := store.NewMemoryStore()
	s.SeedSoloSession(types.SoloSession{
		SessionID:    "solo-1",
		StateVersion: 3,
	})
	s.SeedRoom(types.MultiplayerRoom{
		RoomID:       "room-1",
		StateVersion: 8,
	})

	err := s.RunTx(context.Background(), func(ctx context.Context, tx store.TxStore) error {
		session, err := tx.LoadSoloSession(ctx, "solo-1")
		if err != nil {
			return err
		}
		if session.StateVersion != 3 {
			t.Fatalf("expected session version 3, got %d", session.StateVersion)
		}

		room, err := tx.LoadRoom(ctx, "room-1")
		if err != nil {
			return err
		}
		if room.StateVersion != 8 {
			t.Fatalf("expected room version 8, got %d", room.StateVersion)
		}

		if err := tx.AppendEvent(ctx, types.InputEvent{EventID: "evt-1"}); err != nil {
			return err
		}
		if err := tx.SaveCheckpoint(ctx, types.RuntimeCheckpoint{CheckpointID: "cp-1"}); err != nil {
			return err
		}
		return tx.SaveRenderFrame(ctx, types.RenderFrame{FrameID: "frame-1"})
	})
	if err != nil {
		t.Fatalf("RunTx returned error: %v", err)
	}

	snapshot := s.Snapshot()
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
