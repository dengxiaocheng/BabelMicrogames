package relcore

import "babel-runtime/internal/core/types"

type Core interface {
	Apply(graph types.RelationshipGraph, deltas []types.RelationshipDelta) types.RelationshipGraph
	VisibleSlice(graph types.RelationshipGraph, actorID string) types.RelationshipGraph
}

