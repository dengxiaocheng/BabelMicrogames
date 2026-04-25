package eventlog

import (
	"context"
	"fmt"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/multiplayer"
)

type DeterministicRoomRunner struct {
	Runtime multiplayer.Runtime
}

func (r DeterministicRoomRunner) Replay(ctx context.Context, seed types.MultiplayerRoom, events []types.InputEvent) (types.MultiplayerRoom, error) {
	if r.Runtime == nil {
		return types.MultiplayerRoom{}, fmt.Errorf("missing multiplayer runtime")
	}

	current := seed
	for _, event := range events {
		if event.RuntimeType != types.RuntimeMultiplayer {
			continue
		}
		action := pendingRoomActionFromEvent(event)
		updated, err := r.Runtime.Submit(ctx, current, action)
		if err != nil {
			return types.MultiplayerRoom{}, err
		}
		current = updated
		if multiplayer.TurnReady(current) {
			closed, _, err := r.Runtime.CloseTurn(ctx, current)
			if err != nil {
				return types.MultiplayerRoom{}, err
			}
			current = closed
		}
	}
	return current, nil
}

func pendingRoomActionFromEvent(event types.InputEvent) types.PendingAction {
	return types.PendingAction{
		ActionID:       event.EventID,
		UserInput:      string(event.Payload),
		ActionType:     "free_text",
		Parameters:     map[string]string{"player_id": event.ActorID},
		IdempotencyKey: event.IdempotencyKey,
	}
}
