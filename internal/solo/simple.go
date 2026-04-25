package solo

import (
	"context"
	"fmt"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/settlement"
)

type Renderer interface {
	RenderSolo(ctx context.Context, req types.SoloRenderEnvelope) (types.RenderFrame, error)
}

type SimpleRuntime struct {
	Settlement settlement.Engine
}

func (r SimpleRuntime) Step(ctx context.Context, session types.SoloSession, action types.PendingAction) (types.StepResult, error) {
	_ = ctx
	if r.Settlement == nil {
		return types.StepResult{}, fmt.Errorf("missing settlement engine")
	}
	delta, err := r.Settlement.StepSolo(types.SoloStepInput{
		Session: session,
		Action:  action,
	})
	if err != nil {
		return types.StepResult{}, err
	}
	return types.StepResult{
		RuntimeID:    session.SessionID,
		StateVersion: session.StateVersion + 1,
		Delta:        delta,
		Checkpoint:   "solo.simulate",
	}, nil
}

