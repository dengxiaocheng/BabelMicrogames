package wechat

import "sync"

type SessionModeStore interface {
	GetSelectedRoute(userID string) (RouteKind, bool)
	SetSelectedRoute(userID string, route RouteKind)
}

type MemorySessionModeStore struct {
	mu     sync.RWMutex
	routes map[string]RouteKind
}

func NewMemorySessionModeStore() *MemorySessionModeStore {
	return &MemorySessionModeStore{
		routes: map[string]RouteKind{},
	}
}

func (s *MemorySessionModeStore) GetSelectedRoute(userID string) (RouteKind, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	route, ok := s.routes[userID]
	return route, ok
}

func (s *MemorySessionModeStore) SetSelectedRoute(userID string, route RouteKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[userID] = route
}

var _ SessionModeStore = (*MemorySessionModeStore)(nil)
