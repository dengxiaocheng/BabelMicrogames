package delivery_test

import (
	"context"
	"testing"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/delivery"
)

func TestQueueDispatcherReturnsQueuedJob(t *testing.T) {
	dispatcher := delivery.QueueDispatcher{
		Now: func() time.Time { return time.Unix(50, 0) },
	}
	job, err := dispatcher.Enqueue(context.Background(), types.DeliveryPlan{
		PlanID:      "plan-1",
		RuntimeID:   "runtime-1",
		ExecutionID: "exec-1",
		RecipientID: "user-1",
		Transport:   "wechat",
		Payload:     "hello",
	})
	if err != nil {
		t.Fatalf("Enqueue returned error: %v", err)
	}
	if job.Status != types.DeliveryQueued {
		t.Fatalf("expected queued status, got %q", job.Status)
	}
	if job.JobID != "plan-1:job" {
		t.Fatalf("unexpected job id %q", job.JobID)
	}
}
