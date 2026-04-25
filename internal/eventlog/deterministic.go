package eventlog

import (
	"context"
	"fmt"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/solo"
)

type DeterministicSoloRunner struct {
	Runtime solo.Runtime
}

func (r DeterministicSoloRunner) Replay(ctx context.Context, seed types.SoloSession, events []types.InputEvent) (types.SoloSession, error) {
	if r.Runtime == nil {
		return types.SoloSession{}, fmt.Errorf("missing solo runtime")
	}

	current := seed
	for _, event := range events {
		if event.RuntimeType != types.RuntimeSolo {
			continue
		}
		action := pendingActionFromEvent(event)
		result, err := r.Runtime.Step(ctx, current, action)
		if err != nil {
			return types.SoloSession{}, err
		}
		current = solo.ApplyStepResult(current, action, result, "")
	}
	return current, nil
}

func pendingActionFromEvent(event types.InputEvent) types.PendingAction {
	return types.PendingAction{
		ActionID:       event.EventID,
		UserInput:      string(event.Payload),
		ActionType:     "free_text",
		IdempotencyKey: event.IdempotencyKey,
	}
}
