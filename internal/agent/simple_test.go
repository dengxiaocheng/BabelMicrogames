package agent_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"babel-runtime/internal/agent"
	"babel-runtime/internal/core/types"
)

func TestSimpleSupervisorStartsAndCollectsTasks(t *testing.T) {
	supervisor := &agent.SimpleSupervisor{
		MemoryRoot: t.TempDir(),
		Now: func() time.Time {
			return time.Unix(1700001000, 0)
		},
	}
	execution := types.ExecutionRecord{
		ExecutionID: "exec-1",
		RuntimeID:   "runtime-1",
	}

	err := supervisor.StartTasks(context.Background(), execution, []types.AgentTaskSpec{
		{
			TaskID:       "task-1",
			TaskType:     "consult.reply",
			RuntimeID:    "runtime-1",
			Input:        "consult[1]: 帮我评估架构",
			ArtifactType: "assistant_text",
		},
	})
	if err != nil {
		t.Fatalf("StartTasks returned error: %v", err)
	}

	artifacts, err := supervisor.Collect(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if artifacts[0].Body != "consult[1]: 帮我评估架构" {
		t.Fatalf("unexpected artifact body %q", artifacts[0].Body)
	}
	memoryDir := filepath.Join(supervisor.MemoryRoot, "runtime-1")
	recentSummary, err := os.ReadFile(filepath.Join(memoryDir, "recent_summary.md"))
	if err != nil {
		t.Fatalf("ReadFile recent_summary.md returned error: %v", err)
	}
	if string(recentSummary) != "consult[1]: 帮我评估架构\n" {
		t.Fatalf("unexpected recent summary %q", string(recentSummary))
	}
	stateBody, err := os.ReadFile(filepath.Join(memoryDir, "state.json"))
	if err != nil {
		t.Fatalf("ReadFile state.json returned error: %v", err)
	}
	if len(stateBody) == 0 {
		t.Fatalf("expected non-empty state.json")
	}
	primaryContext, err := os.ReadFile(filepath.Join(memoryDir, "primary_context.md"))
	if err != nil {
		t.Fatalf("ReadFile primary_context.md returned error: %v", err)
	}
	if !strings.Contains(string(primaryContext), "preferred_worker: `claude_code`") {
		t.Fatalf("expected primary context to advertise claude_code worker, got %q", string(primaryContext))
	}
	manifestBody, err := os.ReadFile(filepath.Join(memoryDir, "session_manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile session_manifest.json returned error: %v", err)
	}
	if !strings.Contains(string(manifestBody), "\"primary_document\": \"primary_context.md\"") {
		t.Fatalf("expected session manifest to point at primary_context.md, got %q", string(manifestBody))
	}

	artifacts, err = supervisor.Collect(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("second Collect returned error: %v", err)
	}
	if len(artifacts) != 0 {
		t.Fatalf("expected no artifacts after collection, got %d", len(artifacts))
	}
}

func TestSimpleSupervisorIgnoresOperationalMemoryWriteFailure(t *testing.T) {
	errFile := filepath.Join(t.TempDir(), "memory-root-file")
	if err := os.WriteFile(errFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var memoryErr error
	supervisor := &agent.SimpleSupervisor{
		MemoryRoot: errFile,
		OnMemoryWriteError: func(err error) {
			memoryErr = err
		},
	}

	err := supervisor.StartTasks(context.Background(), types.ExecutionRecord{
		ExecutionID: "exec-1",
		RuntimeID:   "runtime-1",
	}, []types.AgentTaskSpec{
		{
			TaskID:       "task-1",
			TaskType:     "free_chat.reply",
			RuntimeID:    "runtime-1",
			Input:        "free chat[1]: 你好",
			ArtifactType: "assistant_text",
		},
	})
	if err != nil {
		t.Fatalf("StartTasks returned error: %v", err)
	}

	artifacts, err := supervisor.Collect(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	if memoryErr == nil {
		t.Fatalf("expected memory write error callback")
	}
}
