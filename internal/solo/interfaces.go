package solo

import (
	"context"

	"babel-runtime/internal/core/types"
)

type Runtime interface {
	Step(ctx context.Context, session types.SoloSession, action types.PendingAction) (types.StepResult, error)
}

