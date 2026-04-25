package coordinator

import (
	"context"

	"babel-runtime/internal/core/types"
)

type Coordinator interface {
	HandleSoloInput(ctx context.Context, in types.SoloInput) error
	HandleRoomInput(ctx context.Context, in types.RoomInput) error
}

