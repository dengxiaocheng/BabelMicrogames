package mode_test

import (
	"context"
	"testing"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/mode"
)

func TestFreeChatModuleRequestsArtifactThenAppliesIt(t *testing.T) {
	module := mode.FreeChatModule{
		Now: func() time.Time {
			return time.Unix(1700000500, 0)
		},
	}

	snapshot, err := types.EncodeRuntimeSnapshot(
		"chat-1",
		types.ModeFreeChat,
		2,
		1,
		types.FreeChatState{
			RuntimeID:         "chat-1",
			TurnCount:         2,
			LastUserText:      "之前的问题",
			LastAssistantText: "之前的回复",
		},
		time.Unix(1700000000, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}

	result, err := module.Execute(context.Background(), types.ModeExecutionInput{
		Runtime: types.RuntimeRecord{
			RuntimeID:     "chat-1",
			ModeID:        types.ModeFreeChat,
			SchemaVersion: 1,
			HeadVersion:   2,
		},
		Execution: types.ExecutionRecord{
			ExecutionID:    "exec-1",
			RuntimeID:      "chat-1",
			IdempotencyKey: "idem-1",
			ModeID:         types.ModeFreeChat,
			Stage:          types.ExecutionPlanned,
		},
		Snapshot: &snapshot,
		Command: types.ModeCommand{
			CommandType: "chat.user_text",
			Text:        "你好",
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.NextStage != types.ExecutionAwaitingArtifacts {
		t.Fatalf("expected awaiting artifacts stage, got %q", result.NextStage)
	}
	if len(result.AgentTasks) != 1 {
		t.Fatalf("expected 1 agent task, got %d", len(result.AgentTasks))
	}
	if result.AgentTasks[0].Input != "free chat[3]: 你好" {
		t.Fatalf("unexpected task input %q", result.AgentTasks[0].Input)
	}

	result, err = module.Execute(context.Background(), types.ModeExecutionInput{
		Runtime: types.RuntimeRecord{
			RuntimeID:     "chat-1",
			ModeID:        types.ModeFreeChat,
			SchemaVersion: 1,
			HeadVersion:   2,
		},
		Execution: types.ExecutionRecord{
			ExecutionID:    "exec-1",
			RuntimeID:      "chat-1",
			IdempotencyKey: "idem-1",
			ModeID:         types.ModeFreeChat,
			Stage:          types.ExecutionAwaitingArtifacts,
		},
		Snapshot: &snapshot,
		Artifacts: []types.AgentArtifact{
			{
				ArtifactID:   "artifact-1",
				ExecutionID:  "exec-1",
				ArtifactType: "assistant_text",
				Body:         "free chat[3]: 你好",
			},
		},
		Command: types.ModeCommand{
			CommandType: "chat.user_text",
			Text:        "你好",
		},
	})
	if err != nil {
		t.Fatalf("Execute with artifact returned error: %v", err)
	}
	updated, err := types.DecodeRuntimeSnapshot[types.FreeChatState](*result.Snapshot)
	if err != nil {
		t.Fatalf("DecodeRuntimeSnapshot returned error: %v", err)
	}
	if updated.TurnCount != 3 {
		t.Fatalf("expected turn count 3, got %d", updated.TurnCount)
	}
	if updated.LastAssistantText != "free chat[3]: 你好" {
		t.Fatalf("unexpected assistant text %q", updated.LastAssistantText)
	}
}
