package mode

import (
	"context"
	"fmt"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/corehost"
	"babel-runtime/internal/settlement"
)

type SoloSceneModule struct {
	Settlement settlement.Engine
	Host       corehost.SceneHost
	Now        func() time.Time
}

func (m SoloSceneModule) ModeID() types.ModeID {
	return types.ModeSoloScene
}

func (m SoloSceneModule) BuildCommand(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (types.ModeCommand, error) {
	_ = ctx
	_ = runtime
	if env.Text == "" {
		return types.ModeCommand{}, fmt.Errorf("missing user text")
	}
	return types.ModeCommand{
		CommandType: "scene.user_text",
		Text:        env.Text,
	}, nil
}

func (m SoloSceneModule) Execute(ctx context.Context, input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
	_ = ctx
	if m.Host == nil && m.Settlement == nil {
		return types.ModeExecutionResult{}, fmt.Errorf("missing settlement engine")
	}
	if input.Snapshot == nil {
		return types.ModeExecutionResult{}, fmt.Errorf("missing runtime snapshot")
	}

	session, err := types.DecodeRuntimeSnapshot[types.SoloSession](*input.Snapshot)
	if err != nil {
		return types.ModeExecutionResult{}, err
	}
	if session.SessionID == "" {
		return types.ModeExecutionResult{}, fmt.Errorf("invalid solo session snapshot")
	}

	action := types.PendingAction{
		ActionID:       input.Execution.ExecutionID,
		UserInput:      input.Command.Text,
		ActionType:     "free_text",
		IdempotencyKey: input.Execution.IdempotencyKey,
	}

	updated, err := m.sceneHost().StepSolo(ctx, corehost.SoloStepInput{
		Runtime:      input.Runtime,
		Requirements: input.Requirements,
		Session:      session,
		Action:       action,
	})
	if err != nil {
		return types.ModeExecutionResult{}, err
	}

	snapshot, err := types.EncodeRuntimeSnapshot(
		input.Runtime.RuntimeID,
		m.ModeID(),
		updated.StateVersion,
		updated.SchemaVersion,
		updated,
		m.now(),
	)
	if err != nil {
		return types.ModeExecutionResult{}, err
	}

	return types.ModeExecutionResult{
		NextStage: types.ExecutionSettled,
		Snapshot:  &snapshot,
		Events: []types.RuntimeEvent{
			{
				EventID: input.Execution.ExecutionID + ":scene.applied",
				Kind:    "scene.applied",
				Text:    input.Command.Text,
			},
		},
	}, nil
}

func (m SoloSceneModule) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m SoloSceneModule) sceneHost() corehost.SceneHost {
	if m.Host != nil {
		return m.Host
	}
	return corehost.LocalSceneHost{Settlement: m.Settlement}
}

var _ Module = (*SoloSceneModule)(nil)
