package mode

import (
	"context"
	"fmt"
	"time"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/corehost"
	"babel-runtime/internal/settlement"
)

type RoomSceneModule struct {
	Settlement settlement.Engine
	Host       corehost.SceneHost
	Now        func() time.Time
}

func (m RoomSceneModule) ModeID() types.ModeID {
	return types.ModeRoomScene
}

func (m RoomSceneModule) BuildCommand(ctx context.Context, runtime types.RuntimeRecord, env types.InboundEnvelope) (types.ModeCommand, error) {
	_ = ctx
	_ = runtime
	if env.Text == "" {
		return types.ModeCommand{}, fmt.Errorf("missing user text")
	}
	return types.ModeCommand{
		CommandType: "room.user_text",
		Text:        env.Text,
	}, nil
}

func (m RoomSceneModule) Execute(ctx context.Context, input types.ModeExecutionInput) (types.ModeExecutionResult, error) {
	if m.Host == nil && m.Settlement == nil {
		return types.ModeExecutionResult{}, fmt.Errorf("missing settlement engine")
	}
	if input.Snapshot == nil {
		return types.ModeExecutionResult{}, fmt.Errorf("missing runtime snapshot")
	}
	if input.Execution.ActorID == "" {
		return types.ModeExecutionResult{}, fmt.Errorf("missing actor id")
	}

	room, err := types.DecodeRuntimeSnapshot[types.MultiplayerRoom](*input.Snapshot)
	if err != nil {
		return types.ModeExecutionResult{}, err
	}
	if room.RoomID == "" {
		return types.ModeExecutionResult{}, fmt.Errorf("invalid room snapshot")
	}

	action := types.PendingAction{
		ActionID:       input.Execution.ExecutionID,
		UserInput:      input.Command.Text,
		ActionType:     "free_text",
		Parameters:     map[string]string{"player_id": input.Execution.ActorID},
		IdempotencyKey: input.Execution.IdempotencyKey,
	}
	updatedRoom, err := m.sceneHost().StepRoom(ctx, corehost.RoomStepInput{
		Runtime:      input.Runtime,
		Requirements: input.Requirements,
		Room:         room,
		Action:       action,
	})
	if err != nil {
		return types.ModeExecutionResult{}, err
	}

	events := []types.RuntimeEvent{
		{
			EventID: input.Execution.ExecutionID + ":room.action_submitted",
			Kind:    "room.action_submitted",
			Text:    input.Command.Text,
		},
	}
	snapshotRoom := updatedRoom
	if updatedRoom.LastCommittedTurnID != room.LastCommittedTurnID && updatedRoom.LastCommittedTurnID != "" {
		events = append(events, types.RuntimeEvent{
			EventID: input.Execution.ExecutionID + ":room.turn_closed",
			Kind:    "room.turn_closed",
			Text:    updatedRoom.LastCommittedTurnID,
		})
	}

	snapshot, err := types.EncodeRuntimeSnapshot(
		input.Runtime.RuntimeID,
		m.ModeID(),
		snapshotRoom.StateVersion,
		snapshotRoom.SchemaVersion,
		snapshotRoom,
		m.now(),
	)
	if err != nil {
		return types.ModeExecutionResult{}, err
	}

	return types.ModeExecutionResult{
		NextStage: types.ExecutionSettled,
		Snapshot:  &snapshot,
		Events:    events,
	}, nil
}

func (m RoomSceneModule) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m RoomSceneModule) sceneHost() corehost.SceneHost {
	if m.Host != nil {
		return m.Host
	}
	return corehost.LocalSceneHost{Settlement: m.Settlement}
}

var _ Module = (*RoomSceneModule)(nil)
