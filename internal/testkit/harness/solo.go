package harness

import (
	"context"
	"fmt"

	"babel-runtime/internal/coordinator"
	"babel-runtime/internal/core/types"
	"babel-runtime/internal/llm"
	"babel-runtime/internal/recovery"
	"babel-runtime/internal/render"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/solo"
	"babel-runtime/internal/store"
	"babel-runtime/internal/testkit/fakellm"
	"babel-runtime/internal/timecore"
)

type SoloHarness struct {
	Store       *store.MemoryStore
	Coordinator coordinator.SimpleCoordinator
	Recovery    recovery.SimpleSupervisor
	Renderer    *fakellm.Renderer
}

func NewSoloHarness() *SoloHarness {
	return NewSoloHarnessWithStore(store.NewMemoryStore())
}

func NewSoloHarnessWithStore(mem *store.MemoryStore) *SoloHarness {
	renderer := &fakellm.Renderer{
		Response: llm.SoloRenderResponse{
			SceneText: "默认场景输出",
			Options: []types.RenderOption{
				{OptionID: "wait", Label: "等待", ActionTag: "wait"},
			},
		},
	}

	engine := settlement.SimpleEngine{
		Deps: settlement.Dependencies{
			TimeCore: timecore.SimpleCore{},
		},
	}

	return &SoloHarness{
		Store: mem,
		Coordinator: coordinator.SimpleCoordinator{
			Store:       mem,
			SoloRuntime: solo.SimpleRuntime{Settlement: engine},
			SoloRenderer: render.SimpleRenderer{
				LLM: renderer,
			},
		},
		Recovery: recovery.SimpleSupervisor{
			Store: mem,
		},
		Renderer: renderer,
	}
}

func (h *SoloHarness) SeedSession(session types.SoloSession) {
	h.Store.SeedSoloSession(session)
}

func (h *SoloHarness) SubmitText(ctx context.Context, sessionID, userID, text string) error {
	if h == nil {
		return fmt.Errorf("nil harness")
	}
	return h.Coordinator.HandleSoloInput(ctx, types.SoloInput{
		UserID:         userID,
		RuntimeID:      sessionID,
		IdempotencyKey: sessionID + ":" + text,
		Text:           text,
	})
}

func (h *SoloHarness) SweepRecovery(ctx context.Context) (recovery.Report, error) {
	if h == nil {
		return recovery.Report{}, fmt.Errorf("nil harness")
	}
	return h.Recovery.Sweep(ctx)
}

func (h *SoloHarness) Snapshot() store.MemorySnapshot {
	if h == nil {
		return store.MemorySnapshot{}
	}
	return h.Store.Snapshot()
}
