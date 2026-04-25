package mapcore

import "babel-runtime/internal/core/types"

type SimpleCore struct {
	adjacency map[string]map[string]int
}

func NewSimpleCore() *SimpleCore {
	return &SimpleCore{adjacency: map[string]map[string]int{}}
}

func (s *SimpleCore) Link(a, b string, cost int) {
	if s.adjacency[a] == nil {
		s.adjacency[a] = map[string]int{}
	}
	if s.adjacency[b] == nil {
		s.adjacency[b] = map[string]int{}
	}
	s.adjacency[a][b] = cost
	s.adjacency[b][a] = cost
}

func (s *SimpleCore) IsAdjacent(a, b string) bool {
	_, ok := s.adjacency[a][b]
	return ok
}

func (s *SimpleCore) TravelCost(a, b string) int {
	if cost, ok := s.adjacency[a][b]; ok {
		return cost
	}
	return -1
}

func (s *SimpleCore) VisiblePeers(zoneID string, actors []types.PlayerState) []types.PlayerState {
	var peers []types.PlayerState
	for _, actor := range actors {
		if actor.Location.ZoneID == zoneID {
			peers = append(peers, actor)
		}
	}
	return peers
}

