package mapcore

import "babel-runtime/internal/core/types"

type Core interface {
	IsAdjacent(a, b string) bool
	TravelCost(a, b string) int
	VisiblePeers(zoneID string, actors []types.PlayerState) []types.PlayerState
}

