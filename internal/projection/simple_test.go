package projection_test

import (
	"context"
	"testing"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/projection"
)

func TestSimpleProjectorBuildsFrameAndDeliveryPlan(t *testing.T) {
	snapshot, err := types.EncodeRuntimeSnapshot(
		"chat-1",
		types.ModeFreeChat,
		1,
		1,
		types.FreeChatState{
			RuntimeID:         "chat-1",
			LastAssistantText: "free chat[1]: 你好",
		},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}

	projector := projection.SimpleProjector{
		DefaultTransport: "wechat",
		Now: func() time.Time {
			return time.Unix(20, 0)
		},
	}
	result, err := projector.Project(context.Background(), types.ProjectionInput{
		Runtime: types.RuntimeRecord{
			RuntimeID: "chat-1",
			ModeID:    types.ModeFreeChat,
		},
		Execution: types.ExecutionRecord{
			ExecutionID: "exec-1",
			RuntimeID:   "chat-1",
			ActorID:     "user-1",
			Transport:   "wechat",
		},
		Snapshot: &snapshot,
	})
	if err != nil {
		t.Fatalf("Project returned error: %v", err)
	}
	if len(result.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(result.Frames))
	}
	if result.Frames[0].Body != "free chat[1]: 你好" {
		t.Fatalf("unexpected frame body %q", result.Frames[0].Body)
	}
	if len(result.DeliveryPlans) != 1 {
		t.Fatalf("expected 1 delivery plan, got %d", len(result.DeliveryPlans))
	}
	if result.DeliveryPlans[0].Payload != "free chat[1]: 你好" {
		t.Fatalf("unexpected delivery payload %q", result.DeliveryPlans[0].Payload)
	}
}
