package kernel

import (
	"context"

	"babel-runtime/internal/core/types"
)

type Engine interface {
	Accept(ctx context.Context, env types.InboundEnvelope) (types.ExecutionTicket, error)
	Resume(ctx context.Context, executionID string) error
}
