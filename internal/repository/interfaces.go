package repository

import (
	"context"
	"errors"

	"babel-runtime/internal/core/types"
)

var (
	ErrRuntimeNotFound   = errors.New("runtime not found")
	ErrSnapshotNotFound  = errors.New("runtime snapshot not found")
	ErrExecutionNotFound = errors.New("execution not found")
)

type ExecutionTx interface {
	// ExecutionTx is the only place where one execution stage is allowed to
	// atomically move runtime state, task state, artifact state, and output state together.
	LoadRuntime(ctx context.Context, runtimeID string) (types.RuntimeRecord, error)
	SaveRuntime(ctx context.Context, runtime types.RuntimeRecord) error
	LoadRuntimeSnapshot(ctx context.Context, runtimeID string) (types.RuntimeSnapshot, bool, error)
	SaveRuntimeSnapshot(ctx context.Context, snapshot types.RuntimeSnapshot) error
	FindExecutionByIdempotency(ctx context.Context, runtimeID, idempotencyKey string) (types.ExecutionRecord, bool, error)
	LoadExecution(ctx context.Context, executionID string) (types.ExecutionRecord, error)
	SaveExecution(ctx context.Context, execution types.ExecutionRecord) error
	ListAgentTasksByExecution(ctx context.Context, executionID string) ([]types.AgentTaskRecord, error)
	SaveAgentTask(ctx context.Context, task types.AgentTaskRecord) error
	ListAgentArtifactsByExecution(ctx context.Context, executionID string) ([]types.AgentArtifact, error)
	SaveAgentArtifact(ctx context.Context, artifact types.AgentArtifact) error
	AppendEvent(ctx context.Context, event types.RuntimeEvent) error
	SaveProjectionFrame(ctx context.Context, frame types.ProjectionFrame) error
	SaveDeliveryJob(ctx context.Context, job types.DeliveryJob) error
}

type Repository interface {
	RunExecutionTx(ctx context.Context, fn func(ctx context.Context, tx ExecutionTx) error) error
	ListExecutions(ctx context.Context) ([]types.ExecutionRecord, error)
	ListAgentTasks(ctx context.Context, executionID string) ([]types.AgentTaskRecord, error)
	ListAgentArtifacts(ctx context.Context, executionID string) ([]types.AgentArtifact, error)
	ListProjectionFrames(ctx context.Context, runtimeID string) ([]types.ProjectionFrame, error)
	ListDeliveryJobs(ctx context.Context, runtimeID string) ([]types.DeliveryJob, error)
}
