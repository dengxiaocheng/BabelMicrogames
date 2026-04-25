package store

import (
	"context"

	"babel-runtime/internal/core/types"
)

type TxStore interface {
	LoadSoloSession(ctx context.Context, sessionID string) (types.SoloSession, error)
	LoadRoom(ctx context.Context, roomID string) (types.MultiplayerRoom, error)
	SaveSoloSession(ctx context.Context, session types.SoloSession) error
	SaveRoom(ctx context.Context, room types.MultiplayerRoom) error
	SaveCheckpoint(ctx context.Context, checkpoint types.RuntimeCheckpoint) error
	AppendEvent(ctx context.Context, event types.InputEvent) error
	SaveRenderFrame(ctx context.Context, frame types.RenderFrame) error
}

type Store interface {
	RunTx(ctx context.Context, fn func(ctx context.Context, tx TxStore) error) error
}
