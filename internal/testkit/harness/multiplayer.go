package harness

import (
	"context"
	"fmt"

	"babel-runtime/internal/coordinator"
	"babel-runtime/internal/core/types"
	"babel-runtime/internal/mapcore"
	"babel-runtime/internal/multiplayer"
	"babel-runtime/internal/recovery"
	"babel-runtime/internal/relcore"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/store"
	"babel-runtime/internal/timecore"
)

type MultiplayerHarness struct {
	Store       *store.MemoryStore
	Coordinator coordinator.SimpleCoordinator
	Recovery    recovery.SimpleSupervisor
}

func NewMultiplayerHarness() *MultiplayerHarness {
	return NewMultiplayerHarnessWithStore(store.NewMemoryStore())
}

func NewMultiplayerHarnessWithStore(mem *store.MemoryStore) *MultiplayerHarness {
	engine := settlement.SimpleEngine{
		Deps: settlement.Dependencies{
			TimeCore: timecore.SimpleCore{},
			MapCore:  mapcore.NewSimpleCore(),
			RelCore:  relcore.SimpleCore{},
		},
	}

	return &MultiplayerHarness{
		Store: mem,
		Coordinator: coordinator.SimpleCoordinator{
			Store: mem,
			RoomRuntime: multiplayer.SimpleRuntime{
				Settlement: engine,
			},
		},
		Recovery: recovery.SimpleSupervisor{
			Store: mem,
		},
	}
}

func (h *MultiplayerHarness) SeedRoom(room types.MultiplayerRoom) {
	h.Store.SeedRoom(room)
}

func (h *MultiplayerHarness) SubmitText(ctx context.Context, roomID, userID, text string) error {
	if h == nil {
		return fmt.Errorf("nil harness")
	}
	return h.Coordinator.HandleRoomInput(ctx, types.RoomInput{
		UserID:         userID,
		RuntimeID:      roomID,
		IdempotencyKey: roomID + ":" + userID + ":" + text,
		Text:           text,
	})
}

func (h *MultiplayerHarness) SweepRecovery(ctx context.Context) (recovery.Report, error) {
	if h == nil {
		return recovery.Report{}, fmt.Errorf("nil harness")
	}
	return h.Recovery.Sweep(ctx)
}

func (h *MultiplayerHarness) Snapshot() store.MemorySnapshot {
	if h == nil {
		return store.MemorySnapshot{}
	}
	return h.Store.Snapshot()
}
