package repository

import (
	"context"
	"sort"
	"sync"

	"babel-runtime/internal/core/types"
)

type MemoryRepository struct {
	mu       sync.Mutex
	snapshot MemorySnapshot
}

type MemorySnapshot struct {
	Runtimes       map[string]types.RuntimeRecord
	Snapshots      map[string]types.RuntimeSnapshot
	Executions     map[string]types.ExecutionRecord
	AgentTasks     map[string]types.AgentTaskRecord
	AgentArtifacts map[string]types.AgentArtifact
	ExecutionByKey map[string]string
	Events         []types.RuntimeEvent
	Frames         []types.ProjectionFrame
	DeliveryJobs   []types.DeliveryJob
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		snapshot: MemorySnapshot{
			Runtimes:       map[string]types.RuntimeRecord{},
			Snapshots:      map[string]types.RuntimeSnapshot{},
			Executions:     map[string]types.ExecutionRecord{},
			AgentTasks:     map[string]types.AgentTaskRecord{},
			AgentArtifacts: map[string]types.AgentArtifact{},
			ExecutionByKey: map[string]string{},
		},
	}
}

func NewMemoryRepositoryFromSnapshot(snapshot MemorySnapshot) *MemoryRepository {
	return &MemoryRepository{
		snapshot: cloneMemorySnapshot(snapshot),
	}
}

func (r *MemoryRepository) SeedRuntime(runtime types.RuntimeRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.Runtimes[runtime.RuntimeID] = runtime
}

func (r *MemoryRepository) SeedExecution(execution types.ExecutionRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.Executions[execution.ExecutionID] = cloneExecutionRecord(execution)
	if execution.RuntimeID != "" && execution.IdempotencyKey != "" {
		r.snapshot.ExecutionByKey[idempotencyIndex(execution.RuntimeID, execution.IdempotencyKey)] = execution.ExecutionID
	}
}

func (r *MemoryRepository) SeedRuntimeSnapshot(snapshot types.RuntimeSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.Snapshots[snapshot.RuntimeID] = snapshot
}

func (r *MemoryRepository) Snapshot() MemorySnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneMemorySnapshot(r.snapshot)
}

func (r *MemoryRepository) RunExecutionTx(ctx context.Context, fn func(ctx context.Context, tx ExecutionTx) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	working := cloneMemorySnapshot(r.snapshot)
	tx := &memoryTx{snapshot: &working}
	if err := fn(ctx, tx); err != nil {
		return err
	}
	r.snapshot = working
	return nil
}

func (r *MemoryRepository) ListExecutions(ctx context.Context) ([]types.ExecutionRecord, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	executions := make([]types.ExecutionRecord, 0, len(r.snapshot.Executions))
	for _, execution := range r.snapshot.Executions {
		executions = append(executions, execution)
	}
	sort.Slice(executions, func(i, j int) bool {
		if executions[i].AcceptedAtUnix == executions[j].AcceptedAtUnix {
			return executions[i].ExecutionID < executions[j].ExecutionID
		}
		return executions[i].AcceptedAtUnix < executions[j].AcceptedAtUnix
	})
	return executions, nil
}

func (r *MemoryRepository) ListAgentTasks(ctx context.Context, executionID string) ([]types.AgentTaskRecord, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	return listAgentTasks(r.snapshot.AgentTasks, executionID), nil
}

func (r *MemoryRepository) ListAgentArtifacts(ctx context.Context, executionID string) ([]types.AgentArtifact, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	return listAgentArtifacts(r.snapshot.AgentArtifacts, executionID), nil
}

func (r *MemoryRepository) ListProjectionFrames(ctx context.Context, runtimeID string) ([]types.ProjectionFrame, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	frames := make([]types.ProjectionFrame, 0)
	for _, frame := range r.snapshot.Frames {
		if runtimeID == "" || frame.RuntimeID == runtimeID {
			frames = append(frames, frame)
		}
	}
	return append([]types.ProjectionFrame(nil), frames...), nil
}

func (r *MemoryRepository) ListDeliveryJobs(ctx context.Context, runtimeID string) ([]types.DeliveryJob, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	jobs := make([]types.DeliveryJob, 0)
	for _, job := range r.snapshot.DeliveryJobs {
		if runtimeID == "" || job.RuntimeID == runtimeID {
			jobs = append(jobs, job)
		}
	}
	return append([]types.DeliveryJob(nil), jobs...), nil
}

type memoryTx struct {
	snapshot *MemorySnapshot
}

func (t *memoryTx) LoadRuntime(ctx context.Context, runtimeID string) (types.RuntimeRecord, error) {
	_ = ctx
	runtime, ok := t.snapshot.Runtimes[runtimeID]
	if !ok {
		return types.RuntimeRecord{}, ErrRuntimeNotFound
	}
	return runtime, nil
}

func (t *memoryTx) SaveRuntime(ctx context.Context, runtime types.RuntimeRecord) error {
	_ = ctx
	t.snapshot.Runtimes[runtime.RuntimeID] = runtime
	return nil
}

func (t *memoryTx) LoadRuntimeSnapshot(ctx context.Context, runtimeID string) (types.RuntimeSnapshot, bool, error) {
	_ = ctx
	snapshot, ok := t.snapshot.Snapshots[runtimeID]
	if !ok {
		return types.RuntimeSnapshot{}, false, nil
	}
	return snapshot, true, nil
}

func (t *memoryTx) SaveRuntimeSnapshot(ctx context.Context, snapshot types.RuntimeSnapshot) error {
	_ = ctx
	t.snapshot.Snapshots[snapshot.RuntimeID] = snapshot
	return nil
}

func (t *memoryTx) FindExecutionByIdempotency(ctx context.Context, runtimeID, idempotencyKey string) (types.ExecutionRecord, bool, error) {
	_ = ctx
	executionID, ok := t.snapshot.ExecutionByKey[idempotencyIndex(runtimeID, idempotencyKey)]
	if !ok {
		return types.ExecutionRecord{}, false, nil
	}
	execution, ok := t.snapshot.Executions[executionID]
	if !ok {
		return types.ExecutionRecord{}, false, ErrExecutionNotFound
	}
	return execution, true, nil
}

func (t *memoryTx) LoadExecution(ctx context.Context, executionID string) (types.ExecutionRecord, error) {
	_ = ctx
	execution, ok := t.snapshot.Executions[executionID]
	if !ok {
		return types.ExecutionRecord{}, ErrExecutionNotFound
	}
	return execution, nil
}

func (t *memoryTx) SaveExecution(ctx context.Context, execution types.ExecutionRecord) error {
	_ = ctx
	t.snapshot.Executions[execution.ExecutionID] = cloneExecutionRecord(execution)
	if execution.RuntimeID != "" && execution.IdempotencyKey != "" {
		t.snapshot.ExecutionByKey[idempotencyIndex(execution.RuntimeID, execution.IdempotencyKey)] = execution.ExecutionID
	}
	return nil
}

func (t *memoryTx) ListAgentTasksByExecution(ctx context.Context, executionID string) ([]types.AgentTaskRecord, error) {
	_ = ctx
	return listAgentTasks(t.snapshot.AgentTasks, executionID), nil
}

func (t *memoryTx) SaveAgentTask(ctx context.Context, task types.AgentTaskRecord) error {
	_ = ctx
	t.snapshot.AgentTasks[task.TaskID] = task
	return nil
}

func (t *memoryTx) ListAgentArtifactsByExecution(ctx context.Context, executionID string) ([]types.AgentArtifact, error) {
	_ = ctx
	return listAgentArtifacts(t.snapshot.AgentArtifacts, executionID), nil
}

func (t *memoryTx) SaveAgentArtifact(ctx context.Context, artifact types.AgentArtifact) error {
	_ = ctx
	t.snapshot.AgentArtifacts[artifact.ArtifactID] = artifact
	return nil
}

func (t *memoryTx) AppendEvent(ctx context.Context, event types.RuntimeEvent) error {
	_ = ctx
	t.snapshot.Events = append(t.snapshot.Events, event)
	return nil
}

func (t *memoryTx) SaveProjectionFrame(ctx context.Context, frame types.ProjectionFrame) error {
	_ = ctx
	t.snapshot.Frames = append(t.snapshot.Frames, frame)
	return nil
}

func (t *memoryTx) SaveDeliveryJob(ctx context.Context, job types.DeliveryJob) error {
	_ = ctx
	t.snapshot.DeliveryJobs = append(t.snapshot.DeliveryJobs, job)
	return nil
}

func cloneMemorySnapshot(snapshot MemorySnapshot) MemorySnapshot {
	return MemorySnapshot{
		Runtimes:       cloneRuntimeRecords(snapshot.Runtimes),
		Snapshots:      cloneRuntimeSnapshots(snapshot.Snapshots),
		Executions:     cloneExecutionRecords(snapshot.Executions),
		AgentTasks:     cloneAgentTaskRecords(snapshot.AgentTasks),
		AgentArtifacts: cloneAgentArtifacts(snapshot.AgentArtifacts),
		ExecutionByKey: cloneStringMap(snapshot.ExecutionByKey),
		Events:         append([]types.RuntimeEvent(nil), snapshot.Events...),
		Frames:         append([]types.ProjectionFrame(nil), snapshot.Frames...),
		DeliveryJobs:   append([]types.DeliveryJob(nil), snapshot.DeliveryJobs...),
	}
}

func cloneRuntimeRecords(src map[string]types.RuntimeRecord) map[string]types.RuntimeRecord {
	out := make(map[string]types.RuntimeRecord, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func cloneExecutionRecords(src map[string]types.ExecutionRecord) map[string]types.ExecutionRecord {
	out := make(map[string]types.ExecutionRecord, len(src))
	for key, value := range src {
		out[key] = cloneExecutionRecord(value)
	}
	return out
}

func cloneExecutionRecord(src types.ExecutionRecord) types.ExecutionRecord {
	out := src
	if src.PendingDelivery != nil {
		out.PendingDelivery = append([]types.DeliveryPlan(nil), src.PendingDelivery...)
	}
	return out
}

func cloneAgentTaskRecords(src map[string]types.AgentTaskRecord) map[string]types.AgentTaskRecord {
	out := make(map[string]types.AgentTaskRecord, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func cloneAgentArtifacts(src map[string]types.AgentArtifact) map[string]types.AgentArtifact {
	out := make(map[string]types.AgentArtifact, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func cloneRuntimeSnapshots(src map[string]types.RuntimeSnapshot) map[string]types.RuntimeSnapshot {
	out := make(map[string]types.RuntimeSnapshot, len(src))
	for key, value := range src {
		copied := value
		if value.State != nil {
			copied.State = append([]byte(nil), value.State...)
		}
		out[key] = copied
	}
	return out
}

func cloneStringMap(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func idempotencyIndex(runtimeID, idempotencyKey string) string {
	return runtimeID + "\x00" + idempotencyKey
}

func listAgentTasks(src map[string]types.AgentTaskRecord, executionID string) []types.AgentTaskRecord {
	tasks := make([]types.AgentTaskRecord, 0)
	for _, task := range src {
		if executionID == "" || task.ExecutionID == executionID {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAtUnix == tasks[j].CreatedAtUnix {
			return tasks[i].TaskID < tasks[j].TaskID
		}
		return tasks[i].CreatedAtUnix < tasks[j].CreatedAtUnix
	})
	return append([]types.AgentTaskRecord(nil), tasks...)
}

func listAgentArtifacts(src map[string]types.AgentArtifact, executionID string) []types.AgentArtifact {
	artifacts := make([]types.AgentArtifact, 0)
	for _, artifact := range src {
		if executionID == "" || artifact.ExecutionID == executionID {
			artifacts = append(artifacts, artifact)
		}
	}
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].ArtifactID == artifacts[j].ArtifactID {
			return artifacts[i].TaskID < artifacts[j].TaskID
		}
		return artifacts[i].ArtifactID < artifacts[j].ArtifactID
	})
	return append([]types.AgentArtifact(nil), artifacts...)
}

var _ Repository = (*MemoryRepository)(nil)
