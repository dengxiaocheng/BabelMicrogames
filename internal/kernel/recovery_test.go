package kernel_test

import (
	"context"
	"testing"
	"time"

	"babel-runtime/internal/agent"
	"babel-runtime/internal/core/types"
	"babel-runtime/internal/kernel"
	"babel-runtime/internal/mode"
	"babel-runtime/internal/repository"
)

func TestRecoverySupervisorResumesExpiredExecution(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:   "runtime-1",
		ModeID:      types.ModeFreeChat,
		HeadVersion: 1,
		Status:      types.RuntimeStatusActive,
	})
	repo.SeedExecution(types.ExecutionRecord{
		ExecutionID:        "exec-1",
		RuntimeID:          "runtime-1",
		IdempotencyKey:     "idem-1",
		ModeID:             types.ModeFreeChat,
		Stage:              types.ExecutionPlanned,
		CommandType:        "chat.user_text",
		CommandText:        "你好",
		LeaseExpiresAtUnix: 10,
		AcceptedAtUnix:     1,
	})

	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeFreeChat,
		commandType: "chat.user_text",
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
	}
	supervisor := kernel.RecoverySupervisor{
		Repo:   repo,
		Engine: engine,
		Now: func() time.Time {
			return time.Unix(100, 0)
		},
	}

	report, err := supervisor.ResumeStalled(context.Background(), 10)
	if err != nil {
		t.Fatalf("ResumeStalled returned error: %v", err)
	}
	if report.ResumedExecutions != 1 {
		t.Fatalf("expected 1 resumed execution, got %d", report.ResumedExecutions)
	}
	execution := repo.Snapshot().Executions["exec-1"]
	if execution.Stage != types.ExecutionCommitted {
		t.Fatalf("expected execution committed, got %q", execution.Stage)
	}
}

func TestRecoverySupervisorSkipsLeasedExecution(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID: "runtime-1",
		ModeID:    types.ModeFreeChat,
		Status:    types.RuntimeStatusActive,
	})
	repo.SeedExecution(types.ExecutionRecord{
		ExecutionID:        "exec-1",
		RuntimeID:          "runtime-1",
		IdempotencyKey:     "idem-1",
		ModeID:             types.ModeFreeChat,
		Stage:              types.ExecutionPlanned,
		LeaseExpiresAtUnix: 200,
		AcceptedAtUnix:     1,
	})

	router, err := mode.NewStaticRouter(fakeModule{
		id:          types.ModeFreeChat,
		commandType: "chat.user_text",
	})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:   repo,
		Router: router,
	}
	supervisor := kernel.RecoverySupervisor{
		Repo:   repo,
		Engine: engine,
		Now: func() time.Time {
			return time.Unix(100, 0)
		},
	}

	report, err := supervisor.ResumeStalled(context.Background(), 10)
	if err != nil {
		t.Fatalf("ResumeStalled returned error: %v", err)
	}
	if report.ResumedExecutions != 0 {
		t.Fatalf("expected 0 resumed executions, got %d", report.ResumedExecutions)
	}
	if repo.Snapshot().Executions["exec-1"].Stage != types.ExecutionPlanned {
		t.Fatalf("expected execution to remain planned")
	}
}

func TestRecoverySupervisorCompletesAwaitingArtifactExecution(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntime(types.RuntimeRecord{
		RuntimeID:     "runtime-1",
		ModeID:        types.ModeFreeChat,
		SchemaVersion: 1,
		HeadVersion:   0,
		Status:        types.RuntimeStatusActive,
	})
	snapshot, err := types.EncodeRuntimeSnapshot(
		"runtime-1",
		types.ModeFreeChat,
		0,
		1,
		types.FreeChatState{RuntimeID: "runtime-1"},
		time.Unix(1, 0),
	)
	if err != nil {
		t.Fatalf("EncodeRuntimeSnapshot returned error: %v", err)
	}
	repo.SeedRuntimeSnapshot(snapshot)
	repo.SeedExecution(types.ExecutionRecord{
		ExecutionID:        "exec-1",
		RuntimeID:          "runtime-1",
		IdempotencyKey:     "idem-1",
		ModeID:             types.ModeFreeChat,
		Stage:              types.ExecutionAwaitingArtifacts,
		CommandType:        "chat.user_text",
		CommandText:        "你好",
		LeaseExpiresAtUnix: 10,
		AcceptedAtUnix:     1,
	})
	err = repo.RunExecutionTx(context.Background(), func(ctx context.Context, tx repository.ExecutionTx) error {
		return tx.SaveAgentTask(ctx, types.AgentTaskRecord{
			TaskID:        "exec-1:reply",
			ExecutionID:   "exec-1",
			RuntimeID:     "runtime-1",
			TaskType:      "free_chat.reply",
			Input:         "free chat[1]: 你好",
			ArtifactType:  "assistant_text",
			Status:        types.AgentTaskQueued,
			CreatedAtUnix: 1,
		})
	})
	if err != nil {
		t.Fatalf("RunExecutionTx returned error: %v", err)
	}

	router, err := mode.NewStaticRouter(mode.FreeChatModule{})
	if err != nil {
		t.Fatalf("NewStaticRouter returned error: %v", err)
	}
	engine := kernel.SimpleEngine{
		Repo:       repo,
		Router:     router,
		Supervisor: &agent.SimpleSupervisor{},
	}
	supervisor := kernel.RecoverySupervisor{
		Repo:   repo,
		Engine: engine,
		Now: func() time.Time {
			return time.Unix(100, 0)
		},
	}

	report, err := supervisor.ResumeStalled(context.Background(), 10)
	if err != nil {
		t.Fatalf("ResumeStalled returned error: %v", err)
	}
	if report.ResumedExecutions != 1 {
		t.Fatalf("expected 1 resumed execution, got %d", report.ResumedExecutions)
	}
	execution := repo.Snapshot().Executions["exec-1"]
	if execution.Stage != types.ExecutionCommitted {
		t.Fatalf("expected execution committed, got %q", execution.Stage)
	}
}
