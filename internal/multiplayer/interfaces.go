package multiplayer

import (
	"context"

	"babel-runtime/internal/core/types"
)

type Runtime interface {
	Submit(ctx context.Context, room types.MultiplayerRoom, action types.PendingAction) (types.MultiplayerRoom, error)
	CloseTurn(ctx context.Context, room types.MultiplayerRoom) (types.MultiplayerRoom, types.TurnResult, error)
}
