package settlement

import "babel-runtime/internal/core/types"

type Engine interface {
	StepSolo(input types.SoloStepInput) (types.StepDelta, error)
	StepRoom(input types.RoomStepInput) (types.StepDelta, error)
}

