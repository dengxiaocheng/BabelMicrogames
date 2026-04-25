package agent

import (
	"context"
	"sync"
	"time"

	"babel-runtime/internal/core/types"
)

type Supervisor interface {
	StartTasks(ctx context.Context, execution types.ExecutionRecord, tasks []types.AgentTaskSpec) error
	Collect(ctx context.Context, executionID string) ([]types.AgentArtifact, error)
}

type SimpleSupervisor struct {
	mu                 sync.Mutex
	pending            map[string]map[string]types.AgentTaskSpec
	MemoryRoot         string
	OnMemoryWriteError func(error)
	Now                func() time.Time
}

func (s *SimpleSupervisor) StartTasks(ctx context.Context, execution types.ExecutionRecord, tasks []types.AgentTaskSpec) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pending == nil {
		s.pending = map[string]map[string]types.AgentTaskSpec{}
	}
	if _, ok := s.pending[execution.ExecutionID]; !ok {
		s.pending[execution.ExecutionID] = map[string]types.AgentTaskSpec{}
	}
	for _, task := range tasks {
		s.pending[execution.ExecutionID][task.TaskID] = task
	}
	return nil
}

func (s *SimpleSupervisor) Collect(ctx context.Context, executionID string) ([]types.AgentArtifact, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pending[executionID]) == 0 {
		return nil, nil
	}

	// The current scaffold supervisor deterministically turns queued task input
	// into artifacts so the runtime can exercise the full task/artifact lifecycle.
	artifacts := make([]types.AgentArtifact, 0, len(s.pending[executionID]))
	for _, task := range s.pending[executionID] {
		artifactType := task.ArtifactType
		if artifactType == "" {
			artifactType = "assistant_text"
		}
		artifacts = append(artifacts, types.AgentArtifact{
			ArtifactID:   task.TaskID + ":artifact",
			TaskID:       task.TaskID,
			ExecutionID:  executionID,
			RuntimeID:    task.RuntimeID,
			ArtifactType: artifactType,
			Body:         task.Input,
		})
	}
	tasks := make([]types.AgentTaskSpec, 0, len(s.pending[executionID]))
	for _, task := range s.pending[executionID] {
		tasks = append(tasks, task)
	}
	delete(s.pending, executionID)
	// Operational memory is best-effort derived output. A write failure should be
	// visible to operators but must not suppress artifact return to the kernel.
	if err := writeOperationalMemory(s.MemoryRoot, executionID, tasks, artifacts, s.now()); err != nil && s.OnMemoryWriteError != nil {
		s.OnMemoryWriteError(err)
	}
	return artifacts, nil
}

func (s *SimpleSupervisor) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

var _ Supervisor = (*SimpleSupervisor)(nil)
