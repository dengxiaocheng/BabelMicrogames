package fakestore

import (
	"context"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/store"
)

type Tx struct {
	SoloSessions map[string]types.SoloSession
	Rooms        map[string]types.MultiplayerRoom
	Checkpoints  []types.RuntimeCheckpoint
	Events       []types.InputEvent
	Frames       []types.RenderFrame
}

func NewTx() *Tx {
	return &Tx{
		SoloSessions: map[string]types.SoloSession{},
		Rooms:        map[string]types.MultiplayerRoom{},
	}
}

func (t *Tx) LoadSoloSession(ctx context.Context, sessionID string) (types.SoloSession, error) {
	_ = ctx
	return t.SoloSessions[sessionID], nil
}

func (t *Tx) LoadRoom(ctx context.Context, roomID string) (types.MultiplayerRoom, error) {
	_ = ctx
	return t.Rooms[roomID], nil
}

func (t *Tx) SaveSoloSession(ctx context.Context, session types.SoloSession) error {
	_ = ctx
	t.SoloSessions[session.SessionID] = session
	return nil
}

func (t *Tx) SaveRoom(ctx context.Context, room types.MultiplayerRoom) error {
	_ = ctx
	t.Rooms[room.RoomID] = room
	return nil
}

func (t *Tx) SaveCheckpoint(ctx context.Context, checkpoint types.RuntimeCheckpoint) error {
	_ = ctx
	t.Checkpoints = append(t.Checkpoints, checkpoint)
	return nil
}

func (t *Tx) AppendEvent(ctx context.Context, event types.InputEvent) error {
	_ = ctx
	t.Events = append(t.Events, event)
	return nil
}

func (t *Tx) SaveRenderFrame(ctx context.Context, frame types.RenderFrame) error {
	_ = ctx
	t.Frames = append(t.Frames, frame)
	return nil
}

var _ store.TxStore = (*Tx)(nil)

type Store struct {
	Tx *Tx
}

func NewStore() *Store {
	return &Store{Tx: NewTx()}
}

func (s *Store) RunTx(ctx context.Context, fn func(ctx context.Context, tx store.TxStore) error) error {
	return fn(ctx, s.Tx)
}

var _ store.Store = (*Store)(nil)
