package corehost

import (
	"context"

	"babel-runtime/internal/core/types"
)

// SceneHost defines the Go-side adapter boundary for deterministic scene cores.
// A local Go fallback and a future Babel/C++ host adapter must both satisfy this contract.
type SceneHost interface {
	StepSolo(ctx context.Context, input SoloStepInput) (types.SoloSession, error)
	StepRoom(ctx context.Context, input RoomStepInput) (types.MultiplayerRoom, error)
}

type SoloStepInput struct {
	Runtime      types.RuntimeRecord
	Requirements types.RuntimeRequirements
	Session      types.SoloSession
	Action       types.PendingAction
}

type RoomStepInput struct {
	Runtime      types.RuntimeRecord
	Requirements types.RuntimeRequirements
	Room         types.MultiplayerRoom
	Action       types.PendingAction
}
