package mode

import (
	"context"
	"fmt"

	"babel-runtime/internal/core/types"
)

type Module interface {
	ModeID() types.ModeID
	BuildCommand(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (types.ModeCommand, error)
	Execute(ctx context.Context, input types.ModeExecutionInput) (types.ModeExecutionResult, error)
}

type Router interface {
	Resolve(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (Module, error)
}

type StaticRouter struct {
	modules map[types.ModeID]Module
}

func NewStaticRouter(modules ...Module) (*StaticRouter, error) {
	router := &StaticRouter{modules: map[types.ModeID]Module{}}
	for _, module := range modules {
		if err := router.Register(module); err != nil {
			return nil, err
		}
	}
	return router, nil
}

func (r *StaticRouter) Register(module Module) error {
	if module == nil {
		return fmt.Errorf("nil mode module")
	}
	if r.modules == nil {
		r.modules = map[types.ModeID]Module{}
	}
	modeID := module.ModeID()
	if modeID == "" {
		return fmt.Errorf("missing mode id")
	}
	if _, exists := r.modules[modeID]; exists {
		return fmt.Errorf("duplicate mode module: %s", modeID)
	}
	r.modules[modeID] = module
	return nil
}

func (r StaticRouter) Resolve(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (Module, error) {
	_ = ctx
	modeID := runtime.ModeID
	if modeID == "" && env.RouteHint != "" {
		modeID = types.ModeID(env.RouteHint)
	}
	if modeID == "" {
		return nil, fmt.Errorf("unable to resolve mode")
	}
	module, ok := r.modules[modeID]
	if !ok {
		return nil, fmt.Errorf("unregistered mode: %s", modeID)
	}
	return module, nil
}

var _ Router = (*StaticRouter)(nil)
