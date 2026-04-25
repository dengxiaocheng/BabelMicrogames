package repository_test

import (
	"context"
	"errors"
	"testing"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/repository"
)

func TestMemoryRepositoryCommitsSuccessfulTx(t *testing.T) {
	repo := repository.NewMemoryRepository()

	err := repo.RunExecutionTx(context.Background(), func(ctx context.Context, tx repository.ExecutionTx) error {
		if err := tx.SaveRuntime(ctx, types.RuntimeRecord{
			RuntimeID: "runtime-1",
			ModeID:    types.ModeFreeChat,
			Status:    types.RuntimeStatusActive,
		}); err != nil {
			return err
		}
		if err := tx.SaveRuntimeSnapshot(ctx, types.RuntimeSnapshot{
			RuntimeID: "runtime-1",
			ModeID:    types.ModeFreeChat,
			Version:   3,
			State:     []byte(`{"topic":"hello"}`),
		}); err != nil {
			return err
		}
		if err := tx.SaveExecution(ctx, types.ExecutionRecord{
			ExecutionID:    "exec-1",
			RuntimeID:      "runtime-1",
			IdempotencyKey: "idem-1",
			ModeID:         types.ModeFreeChat,
			Stage:          types.ExecutionPlanned,
		}); err != nil {
			return err
		}
		return tx.AppendEvent(ctx, types.RuntimeEvent{
			EventID:     "evt-1",
			RuntimeID:   "runtime-1",
			ExecutionID: "exec-1",
			Kind:        "execution.planned",
		})
	})
	if err != nil {
		t.Fatalf("RunExecutionTx returned error: %v", err)
	}

	snapshot := repo.Snapshot()
	if len(snapshot.Runtimes) != 1 {
		t.Fatalf("expected 1 runtime, got %d", len(snapshot.Runtimes))
	}
	if len(snapshot.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshot.Snapshots))
	}
	if len(snapshot.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(snapshot.Executions))
	}
	if len(snapshot.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(snapshot.Events))
	}
	if len(snapshot.Frames) != 0 {
		t.Fatalf("expected 0 frames, got %d", len(snapshot.Frames))
	}
}

func TestMemoryRepositoryRollsBackFailedTx(t *testing.T) {
	repo := repository.NewMemoryRepository()
	boom := errors.New("boom")

	err := repo.RunExecutionTx(context.Background(), func(ctx context.Context, tx repository.ExecutionTx) error {
		if err := tx.SaveRuntime(ctx, types.RuntimeRecord{
			RuntimeID: "runtime-1",
			ModeID:    types.ModeFreeChat,
		}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got %v", err)
	}

	snapshot := repo.Snapshot()
	if len(snapshot.Runtimes) != 0 {
		t.Fatalf("expected rollback to leave 0 runtimes, got %d", len(snapshot.Runtimes))
	}
}

func TestMemoryRepositoryFindsExecutionByIdempotency(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedExecution(types.ExecutionRecord{
		ExecutionID:    "exec-1",
		RuntimeID:      "runtime-1",
		IdempotencyKey: "idem-1",
		Stage:          types.ExecutionCommitted,
	})

	err := repo.RunExecutionTx(context.Background(), func(ctx context.Context, tx repository.ExecutionTx) error {
		execution, ok, err := tx.FindExecutionByIdempotency(ctx, "runtime-1", "idem-1")
		if err != nil {
			return err
		}
		if !ok {
			t.Fatalf("expected execution by idempotency key")
		}
		if execution.ExecutionID != "exec-1" {
			t.Fatalf("expected exec-1, got %q", execution.ExecutionID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunExecutionTx returned error: %v", err)
	}
}

func TestMemoryRepositoryLoadsSnapshot(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedRuntimeSnapshot(types.RuntimeSnapshot{
		RuntimeID: "runtime-1",
		ModeID:    types.ModeSoloScene,
		Version:   4,
		State:     []byte(`{"session_id":"runtime-1"}`),
	})

	err := repo.RunExecutionTx(context.Background(), func(ctx context.Context, tx repository.ExecutionTx) error {
		snapshot, ok, err := tx.LoadRuntimeSnapshot(ctx, "runtime-1")
		if err != nil {
			return err
		}
		if !ok {
			t.Fatalf("expected snapshot to exist")
		}
		if snapshot.Version != 4 {
			t.Fatalf("expected version 4, got %d", snapshot.Version)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunExecutionTx returned error: %v", err)
	}
}

func TestMemoryRepositoryListsExecutionsInAcceptedOrder(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SeedExecution(types.ExecutionRecord{
		ExecutionID:    "exec-2",
		RuntimeID:      "runtime-1",
		IdempotencyKey: "idem-2",
		AcceptedAtUnix: 20,
	})
	repo.SeedExecution(types.ExecutionRecord{
		ExecutionID:    "exec-1",
		RuntimeID:      "runtime-1",
		IdempotencyKey: "idem-1",
		AcceptedAtUnix: 10,
	})

	executions, err := repo.ListExecutions(context.Background())
	if err != nil {
		t.Fatalf("ListExecutions returned error: %v", err)
	}
	if len(executions) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(executions))
	}
	if executions[0].ExecutionID != "exec-1" {
		t.Fatalf("expected exec-1 first, got %q", executions[0].ExecutionID)
	}
}

func TestMemoryRepositoryListsFramesAndJobsByRuntime(t *testing.T) {
	repo := repository.NewMemoryRepository()
	err := repo.RunExecutionTx(context.Background(), func(ctx context.Context, tx repository.ExecutionTx) error {
		if err := tx.SaveAgentTask(ctx, types.AgentTaskRecord{
			TaskID:      "task-1",
			ExecutionID: "exec-1",
			RuntimeID:   "runtime-1",
			TaskType:    "consult.reply",
			Status:      types.AgentTaskQueued,
		}); err != nil {
			return err
		}
		if err := tx.SaveAgentArtifact(ctx, types.AgentArtifact{
			ArtifactID:   "artifact-1",
			TaskID:       "task-1",
			ExecutionID:  "exec-1",
			RuntimeID:    "runtime-1",
			ArtifactType: "assistant_text",
			Body:         "consult[1]: hello",
		}); err != nil {
			return err
		}
		if err := tx.SaveProjectionFrame(ctx, types.ProjectionFrame{
			FrameID:     "frame-1",
			RuntimeID:   "runtime-1",
			ExecutionID: "exec-1",
			Body:        "hello",
		}); err != nil {
			return err
		}
		return tx.SaveDeliveryJob(ctx, types.DeliveryJob{
			JobID:       "job-1",
			RuntimeID:   "runtime-1",
			ExecutionID: "exec-1",
			Status:      types.DeliveryQueued,
		})
	})
	if err != nil {
		t.Fatalf("RunExecutionTx returned error: %v", err)
	}

	frames, err := repo.ListProjectionFrames(context.Background(), "runtime-1")
	if err != nil {
		t.Fatalf("ListProjectionFrames returned error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	tasks, err := repo.ListAgentTasks(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("ListAgentTasks returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	artifacts, err := repo.ListAgentArtifacts(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("ListAgentArtifacts returned error: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}
	jobs, err := repo.ListDeliveryJobs(context.Background(), "runtime-1")
	if err != nil {
		t.Fatalf("ListDeliveryJobs returned error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
}
