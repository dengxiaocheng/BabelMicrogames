package multiplayer

import (
	"context"
	"fmt"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/settlement"
)

type SimpleRuntime struct {
	Settlement settlement.Engine
}

func (r SimpleRuntime) Submit(ctx context.Context, room types.MultiplayerRoom, action types.PendingAction) (types.MultiplayerRoom, error) {
	_ = ctx
	if room.PendingTurn == nil {
		return types.MultiplayerRoom{}, fmt.Errorf("missing pending turn")
	}
	if action.ActionID == "" {
		return types.MultiplayerRoom{}, fmt.Errorf("missing action id")
	}

	updated := room
	turn := *room.PendingTurn
	if turn.SubmittedActions == nil {
		turn.SubmittedActions = map[string]types.PendingAction{}
	}
	playerID := action.ActionID
	if explicitPlayerID := action.Parameters["player_id"]; explicitPlayerID != "" {
		playerID = explicitPlayerID
	}
	turn.SubmittedActions[playerID] = action
	updated.PendingTurn = &turn
	return updated, nil
}

func (r SimpleRuntime) CloseTurn(ctx context.Context, room types.MultiplayerRoom) (types.MultiplayerRoom, types.TurnResult, error) {
	_ = ctx
	if r.Settlement == nil {
		return types.MultiplayerRoom{}, types.TurnResult{}, fmt.Errorf("missing settlement engine")
	}
	if room.PendingTurn == nil {
		return types.MultiplayerRoom{}, types.TurnResult{}, fmt.Errorf("missing pending turn")
	}

	actions := make([]types.PendingAction, 0, len(room.PendingTurn.SubmittedActions))
	for _, action := range room.PendingTurn.SubmittedActions {
		actions = append(actions, action)
	}

	delta, err := r.Settlement.StepRoom(types.RoomStepInput{
		Room:    room,
		Actions: actions,
	})
	if err != nil {
		return types.MultiplayerRoom{}, types.TurnResult{}, err
	}
	result := types.TurnResult{
		RoomID:       room.RoomID,
		StateVersion: room.StateVersion + 1,
		Delta:        delta,
		Checkpoint:   room.PendingTurn.TurnID,
	}

	return ApplyTurnResult(room, result), result, nil
}
