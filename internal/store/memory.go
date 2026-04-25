package store

import (
	"context"
	"sync"

	"babel-runtime/internal/core/types"
)

type MemoryStore struct {
	mu sync.Mutex

	soloSessions map[string]types.SoloSession
	rooms        map[string]types.MultiplayerRoom
	checkpoints  []types.RuntimeCheckpoint
	events       []types.InputEvent
	frames       []types.RenderFrame
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		soloSessions: map[string]types.SoloSession{},
		rooms:        map[string]types.MultiplayerRoom{},
	}
}

func NewMemoryStoreFromSnapshot(snapshot MemorySnapshot) *MemoryStore {
	return &MemoryStore{
		soloSessions: copySoloSessions(snapshot.SoloSessions),
		rooms:        copyRooms(snapshot.Rooms),
		checkpoints:  append([]types.RuntimeCheckpoint(nil), snapshot.Checkpoints...),
		events:       append([]types.InputEvent(nil), snapshot.Events...),
		frames:       append([]types.RenderFrame(nil), snapshot.Frames...),
	}
}

func (s *MemoryStore) SeedSoloSession(session types.SoloSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.soloSessions[session.SessionID] = session
}

func (s *MemoryStore) SeedRoom(room types.MultiplayerRoom) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.RoomID] = room
}

func (s *MemoryStore) Snapshot() MemorySnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return MemorySnapshot{
		SoloSessions: copySoloSessions(s.soloSessions),
		Rooms:        copyRooms(s.rooms),
		Checkpoints:  append([]types.RuntimeCheckpoint(nil), s.checkpoints...),
		Events:       append([]types.InputEvent(nil), s.events...),
		Frames:       append([]types.RenderFrame(nil), s.frames...),
	}
}

func (s *MemoryStore) RunTx(ctx context.Context, fn func(ctx context.Context, tx TxStore) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(ctx, &memoryTx{store: s})
}

type MemorySnapshot struct {
	SoloSessions map[string]types.SoloSession
	Rooms        map[string]types.MultiplayerRoom
	Checkpoints  []types.RuntimeCheckpoint
	Events       []types.InputEvent
	Frames       []types.RenderFrame
}

type memoryTx struct {
	store *MemoryStore
}

func (t *memoryTx) LoadSoloSession(ctx context.Context, sessionID string) (types.SoloSession, error) {
	_ = ctx
	return t.store.soloSessions[sessionID], nil
}

func (t *memoryTx) LoadRoom(ctx context.Context, roomID string) (types.MultiplayerRoom, error) {
	_ = ctx
	return t.store.rooms[roomID], nil
}

func (t *memoryTx) SaveSoloSession(ctx context.Context, session types.SoloSession) error {
	_ = ctx
	t.store.soloSessions[session.SessionID] = session
	return nil
}

func (t *memoryTx) SaveRoom(ctx context.Context, room types.MultiplayerRoom) error {
	_ = ctx
	t.store.rooms[room.RoomID] = room
	return nil
}

func (t *memoryTx) SaveCheckpoint(ctx context.Context, checkpoint types.RuntimeCheckpoint) error {
	_ = ctx
	t.store.checkpoints = append(t.store.checkpoints, checkpoint)
	return nil
}

func (t *memoryTx) AppendEvent(ctx context.Context, event types.InputEvent) error {
	_ = ctx
	t.store.events = append(t.store.events, event)
	return nil
}

func (t *memoryTx) SaveRenderFrame(ctx context.Context, frame types.RenderFrame) error {
	_ = ctx
	t.store.frames = append(t.store.frames, frame)
	return nil
}

func copySoloSessions(src map[string]types.SoloSession) map[string]types.SoloSession {
	out := make(map[string]types.SoloSession, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func copyRooms(src map[string]types.MultiplayerRoom) map[string]types.MultiplayerRoom {
	out := make(map[string]types.MultiplayerRoom, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

var _ Store = (*MemoryStore)(nil)
