package mode_test

import (
	"context"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/mode"
)

type fakeModule struct {
	id types.ModeID
}

func (m fakeModule) ModeID() types.ModeID {
	return m.id
}

func (m fakeModule) BuildCommand(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (types.ModeCommand, error) {
	_ = ctx
	_ = runtime
	return types.ModeCommand{CommandType: "text", Text: env.Text}, nil
}

func (m fakeModule) Execute(ctx context.Context, input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
	_ = ctx
	_ = input
	return types.ModeExecutionResult{}, nil
}

func TestStaticRouterResolvesRuntimeMode(t *testing.T) {
	router, err := mode.NewStaticRouter(fakeModule{id: types.ModeFreeChat})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}

	module, err := router.Resolve(context.Background(), types.RuntimeRecord{
		RuntimeID: "runtime-1",
		ModeID:    types.ModeFreeChat,
	}, types.InboundEnvelope{})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got := module.ModeID(); got != types.ModeFreeChat {
		t.Fatalf("expected free_chat module, got %q", got)
	}
}

func TestStaticRouterFallsBackToRouteHint(t *testing.T) {
	router, err := mode.NewStaticRouter(fakeModule{id: types.ModeProjectConsult})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}

	module, err := router.Resolve(context.Background(), types.RuntimeRecord{}, types.InboundEnvelope{
		RouteHint: string(types.ModeProjectConsult),
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got := module.ModeID(); got != types.ModeProjectConsult {
		t.Fatalf("expected project_consult module, got %q", got)
	}
}
