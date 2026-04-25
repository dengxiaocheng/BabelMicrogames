package corehost

import (
	"context"
	"fmt"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/multiplayer"
	"babel-runtime/internal/settlement"
	"babel-runtime/internal/solo"
)

// LocalSceneHost keeps the current Go validation path behind the same contract
// that a future Babel/C++ host adapter will satisfy.
type LocalSceneHost struct {
	Settlement settlement.Engine
}

func (h LocalSceneHost) StepSolo(ctx context.Context, input SoloStepInput) (types.SoloSession, error) {
	_ = ctx
	if h.Settlement == nil {
		return types.SoloSession{}, fmt.Errorf("missing settlement engine")
	}

	delta, err := h.Settlement.StepSolo(types.SoloStepInput{
		Session: input.Session,
		Action:  input.Action,
	})
	if err != nil {
		return types.SoloSession{}, err
	}

	step := types.StepResult{
		RuntimeID:    input.Session.SessionID,
		StateVersion: input.Session.StateVersion + 1,
		Delta:        delta,
		Checkpoint:   input.Action.ActionID,
	}
	updated := solo.ApplyStepResult(input.Session, input.Action, step, "")
	if input.Requirements.Ruleset != nil {
		updated.RulesetID = input.Requirements.Ruleset.RulesetID
	}
	if input.Requirements.PromptPack != nil {
		updated.PromptPackID = input.Requirements.PromptPack.PromptPackID
	}
	return updated, nil
}

func (h LocalSceneHost) StepRoom(ctx context.Context, input RoomStepInput) (types.MultiplayerRoom, error) {
	if h.Settlement == nil {
		return types.MultiplayerRoom{}, fmt.Errorf("missing settlement engine")
	}

	runtime := multiplayer.SimpleRuntime{Settlement: h.Settlement}
	updatedRoom, err := runtime.Submit(ctx, input.Room, input.Action)
	if err != nil {
		return types.MultiplayerRoom{}, err
	}
	if multiplayer.TurnReady(updatedRoom) {
		closedRoom, _, err := runtime.CloseTurn(ctx, updatedRoom)
		if err != nil {
			return types.MultiplayerRoom{}, err
		}
		updatedRoom = closedRoom
	}
	if input.Requirements.Ruleset != nil {
		updatedRoom.RulesetID = input.Requirements.Ruleset.RulesetID
	}
	if input.Requirements.PromptPack != nil {
		updatedRoom.PromptPackID = input.Requirements.PromptPack.PromptPackID
	}
	return updatedRoom, nil
}
