package mode_test

import (
	"context"
	"testing"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/mode"
)

func TestProjectConsultModuleSanitizesConsultPrefix(t *testing.T) {
	module := mode.ProjectConsultModule{
		Now: func() time.Time {
			return time.Unix(1700000600, 0)
		},
	}

	snapshot, err := types.EncodeRuntimeSnapshot(
		"consult-1",
		types.ModeProjectConsult,
		0,
		1,
		types.ProjectConsultState{RuntimeID: "consult-1"},
		time.Unix(1700000000, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}

	command, err := module.BuildCommand(context.Background(), types.RuntimeRecord{}, types.InboundEnvelope{
		Text: "/consult 帮我评估架构",
	})
	if err != nil {
		t.Fatalf("BuildCommand returned error: %v", err)
	}
	if command.Text != "帮我评估架构" {
		t.Fatalf("expected sanitized consult text, got %q", command.Text)
	}

	result, err := module.Execute(context.Background(), types.ModeExecutionInput{
		Runtime: types.RuntimeRecord{
			RuntimeID:     "consult-1",
			ModeID:        types.ModeProjectConsult,
			SchemaVersion: 1,
			HeadVersion:   0,
		},
		Execution: types.ExecutionRecord{
			ExecutionID:    "exec-1",
			RuntimeID:      "consult-1",
			IdempotencyKey: "idem-1",
			ModeID:         types.ModeProjectConsult,
			Stage:          types.ExecutionPlanned,
		},
		Snapshot: &snapshot,
		Command:  command,
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
	if result.AgentTasks[0].Input != "consult[1]: 帮我评估架构" {
		t.Fatalf("unexpected task input %q", result.AgentTasks[0].Input)
	}

	result, err = module.Execute(context.Background(), types.ModeExecutionInput{
		Runtime: types.RuntimeRecord{
			RuntimeID:     "consult-1",
			ModeID:        types.ModeProjectConsult,
			SchemaVersion: 1,
			HeadVersion:   0,
		},
		Execution: types.ExecutionRecord{
			ExecutionID:    "exec-1",
			RuntimeID:      "consult-1",
			IdempotencyKey: "idem-1",
			ModeID:         types.ModeProjectConsult,
			Stage:          types.ExecutionAwaitingArtifacts,
		},
		Snapshot: &snapshot,
		Artifacts: []types.AgentArtifact{
			{
				ArtifactID:   "artifact-1",
				ExecutionID:  "exec-1",
				ArtifactType: "assistant_text",
				Body:         "consult[1]: 帮我评估架构",
			},
		},
		Command: command,
	})
	if err != nil {
		t.Fatalf("Execute with artifact returned error: %v", err)
	}
	updated, err := types.DecodeRuntimeSnapshot[types.ProjectConsultState](*result.Snapshot)
	if err != nil {
		t.Fatalf("DecodeRuntimeSnapshot returned error: %v", err)
	}
	if updated.QueryCount != 1 {
		t.Fatalf("expected query count 1, got %d", updated.QueryCount)
	}
	if updated.LastAssistantText != "consult[1]: 帮我评估架构" {
		t.Fatalf("unexpected assistant text %q", updated.LastAssistantText)
	}
}
