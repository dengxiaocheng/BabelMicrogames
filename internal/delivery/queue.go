package delivery

import (
	"context"
	"time"

	"babel-runtime/internal/core/types"
)

type Dispatcher interface {
	Enqueue(ctx context.Context, plan types.DeliveryPlan) (types.DeliveryJob, error)
}

type QueueDispatcher struct {
	Now func() time.Time
}

func (d QueueDispatcher) Enqueue(ctx context.Context, plan types.DeliveryPlan) (types.DeliveryJob, error) {
	_ = ctx
	return types.DeliveryJob{
		JobID:          plan.PlanID + ":job",
		PlanID:         plan.PlanID,
		RuntimeID:      plan.RuntimeID,
		ExecutionID:    plan.ExecutionID,
		RecipientID:    plan.RecipientID,
		Transport:      plan.Transport,
		FrameID:        plan.FrameID,
		Payload:        plan.Payload,
		Status:         types.DeliveryQueued,
		EnqueuedAtUnix: d.now().Unix(),
	}, nil
}

func (d QueueDispatcher) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

var _ Dispatcher = (*QueueDispatcher)(nil)
