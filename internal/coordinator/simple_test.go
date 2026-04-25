package coordinator

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/llm"
	"babel-runtime/internal/multiplayer"
	"babel-runtime/internal/render"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/solo"
	"babel-runtime/internal/store"
	"babel-runtime/internal/testkit/fakellm"
	"babel-runtime/internal/testkit/fakestore"
	"babel-runtime/internal/timecore"
)

func TestHandleSoloInputPersistsCheckpoint(t *testing.T) {
	fs := fakestore.NewStore()
	fs.Tx.SoloSessions["solo-1"] = types.SoloSession{
		SessionID:    "solo-1",
		StateVersion: 1,
		WorldClock:   types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 1},
		PlayerState:  types.PlayerState{ActorID: "u1", Stats: types.Stats{Stamina: 3}},
	}

	coord := SimpleCoordinator{
		Store: fs,
		SoloRuntime: solo.SimpleRuntime{
			Settlement: settlement.SimpleEngine{
				Deps: settlement.Dependencies{TimeCore: timecore.SimpleCore{}},
			},
		},
		SoloRenderer: render.SimpleRenderer{
			LLM: &fakellm.Renderer{
				Response: llm.SoloRenderResponse{
					SceneText: "你继续向前。",
				},
			},
		},
	}

	err := coord.HandleSoloInput(context.Background(), types.SoloInput{
		UserID:         "u1",
		RuntimeID:      "solo-1",
		IdempotencyKey: "evt-1",
		Text:           "搬砖",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fs.Tx.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(fs.Tx.Events))
	}
	if len(fs.Tx.Checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(fs.Tx.Checkpoints))
	}
	if len(fs.Tx.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(fs.Tx.Frames))
	}
	saved := fs.Tx.SoloSessions["solo-1"]
	if saved.StateVersion != 2 {
		t.Fatalf("expected saved state version 2, got %d", saved.StateVersion)
	}
	if saved.PlayerState.Stats.Stamina != 2 {
		t.Fatalf("expected saved stamina 2, got %d", saved.PlayerState.Stats.Stamina)
	}
}

func TestHandleRoomInputClosesTurnWhenReady(t *testing.T) {
	fs := fakestore.NewStore()
	fs.Tx.Rooms["room-1"] = types.MultiplayerRoom{
		RoomID:       "room-1",
		StateVersion: 5,
		Status:       types.RuntimeStatusActive,
		WorldClock:   types.WorldClock{DayIndex: 1, Segment: timecore.SegmentMorning, AbsoluteTick: 3},
		HiddenState: map[string]types.PrivateState{
			"u1": {ActorID: "worker-1"},
			"u2": {ActorID: "worker-2"},
		},
		PendingTurn: &types.PendingTurn{
			TurnID:           "turn-1",
			RequiredPlayers:  []string{"u1", "u2"},
			SubmittedActions: map[string]types.PendingAction{},
		},
	}

	coord := SimpleCoordinator{
		Store: fs,
		RoomRuntime: multiplayer.SimpleRuntime{
			Settlement: settlement.SimpleEngine{
				Deps: settlement.Dependencies{TimeCore: timecore.SimpleCore{}},
			},
		},
	}

	if err := coord.HandleRoomInput(context.Background(), types.RoomInput{
		UserID:         "u1",
		RuntimeID:      "room-1",
		IdempotencyKey: "evt-room-1",
		Text:           "搬石头",
	}); err != nil {
		t.Fatalf("first room input: %v", err)
	}
	if got := len(fs.Tx.Checkpoints); got != 0 {
		t.Fatalf("expected no checkpoint before turn ready, got %d", got)
	}
	if fs.Tx.Rooms["room-1"].PendingTurn == nil {
		t.Fatalf("expected pending turn after first submit")
	}

	if err := coord.HandleRoomInput(context.Background(), types.RoomInput{
		UserID:         "u2",
		RuntimeID:      "room-1",
		IdempotencyKey: "evt-room-2",
		Text:           "拉绳索",
	}); err != nil {
		t.Fatalf("second room input: %v", err)
	}

	room := fs.Tx.Rooms["room-1"]
	if room.StateVersion != 6 {
		t.Fatalf("expected state version 6 after close, got %d", room.StateVersion)
	}
	if room.PendingTurn != nil {
		t.Fatalf("expected pending turn cleared, got %#v", room.PendingTurn)
	}
	if room.LastCommittedTurnID != "turn-1" {
		t.Fatalf("expected last committed turn turn-1, got %q", room.LastCommittedTurnID)
	}
	if got := len(fs.Tx.Checkpoints); got != 1 {
		t.Fatalf("expected one checkpoint after close, got %d", got)
	}
}

var _ store.Store = (*fakestore.Store)(nil)
