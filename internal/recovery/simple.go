package recovery

import (
	"context"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/store"
)

type Snapshotter interface {
	Snapshot() store.MemorySnapshot
}

type SimpleSupervisor struct {
	Store Snapshotter
}

func (s SimpleSupervisor) Sweep(ctx context.Context) (Report, error) {
	_ = ctx

	if s.Store == nil {
		return Report{}, nil
	}

	snapshot := s.Store.Snapshot()
	report := Report{}

	for _, session := range snapshot.SoloSessions {
		if session.Status != types.RuntimeStatusActive && session.Status != types.RuntimeStatusRecovering {
			continue
		}
		if session.PendingAction != nil || session.LastRenderFrameID == "" {
			report.RecoveredRuntimes++
		}
	}

	for _, room := range snapshot.Rooms {
		if room.Status != types.RuntimeStatusActive && room.Status != types.RuntimeStatusRecovering {
			continue
		}
		if room.PendingTurn != nil {
			report.RecoveredRuntimes++
		}
	}

	for _, checkpoint := range snapshot.Checkpoints {
		if checkpoint.Status == types.CheckpointCommitted || checkpoint.Status == types.CheckpointFailed {
			continue
		}
		report.StaleCheckpoints++
		switch checkpoint.ResumePolicy {
		case types.ResumeRetryStep:
			report.RetriedSteps++
		case types.ResumeRerenderOnly, types.ResumeRedeliver:
			report.RedeliveredFrames++
		}
	}

	return report, nil
}
