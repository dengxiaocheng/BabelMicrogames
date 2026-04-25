package kernel

import (
	"context"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/repository"
)

type RecoverySupervisor struct {
	Repo           repository.Repository
	Engine         Engine
	MaxResumeSteps int
	Now            func() time.Time
}

func (s RecoverySupervisor) ResumeStalled(ctx context.Context, limit int) (types.RecoveryReport, error) {
	if s.Repo == nil || s.Engine == nil {
		return types.RecoveryReport{}, nil
	}

	executions, err := s.Repo.ListExecutions(ctx)
	if err != nil {
		return types.RecoveryReport{}, err
	}

	report := types.RecoveryReport{}
	nowUnix := s.now().Unix()
	processed := 0

	for _, execution := range executions {
		if limit > 0 && processed >= limit {
			break
		}
		if isTerminalStage(execution.Stage) {
			continue
		}
		if execution.LeaseExpiresAtUnix != 0 && execution.LeaseExpiresAtUnix > nowUnix {
			continue
		}

		processed++
		if err := s.resumeUntilTerminal(ctx, execution.ExecutionID); err != nil {
			report.FailedExecutions++
			continue
		}
		report.ResumedExecutions++
	}

	return report, nil
}

func (s RecoverySupervisor) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s RecoverySupervisor) maxResumeSteps() int {
	if s.MaxResumeSteps > 0 {
		return s.MaxResumeSteps
	}
	return 8
}

func (s RecoverySupervisor) resumeUntilTerminal(ctx context.Context, executionID string) error {
	for step := 0; step < s.maxResumeSteps(); step++ {
		execution, err := s.loadExecution(ctx, executionID)
		if err != nil {
			return err
		}
		if isTerminalStage(execution.Stage) {
			return nil
		}
		if err := s.Engine.Resume(ctx, executionID); err != nil {
			return err
		}
	}
	return nil
}

func (s RecoverySupervisor) loadExecution(ctx context.Context, executionID string) (types.ExecutionRecord, error) {
	var execution types.ExecutionRecord
	err := s.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		loaded, err := tx.LoadExecution(ctx, executionID)
		if err != nil {
			return err
		}
		execution = loaded
		return nil
	})
	return execution, err
}
