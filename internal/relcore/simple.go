package relcore

import "babel-runtime/internal/core/types"

type SimpleCore struct{}

func (SimpleCore) Apply(graph types.RelationshipGraph, deltas []types.RelationshipDelta) types.RelationshipGraph {
	edges := append([]types.RelationshipEdge(nil), graph.Edges...)
	for _, delta := range deltas {
		found := false
		for i := range edges {
			if edges[i].SourceActorID == delta.SourceActorID && edges[i].TargetActorID == delta.TargetActorID {
				edges[i].Affinity += delta.AffinityDelta
				edges[i].Trust += delta.TrustDelta
				edges[i].Fear += delta.FearDelta
				edges[i].Debt += delta.DebtDelta
				found = true
				break
			}
		}
		if !found {
			edges = append(edges, types.RelationshipEdge{
				SourceActorID: delta.SourceActorID,
				TargetActorID: delta.TargetActorID,
				Affinity:      delta.AffinityDelta,
				Trust:         delta.TrustDelta,
				Fear:          delta.FearDelta,
				Debt:          delta.DebtDelta,
			})
		}
	}
	return types.RelationshipGraph{Edges: edges}
}

func (SimpleCore) VisibleSlice(graph types.RelationshipGraph, actorID string) types.RelationshipGraph {
	var edges []types.RelationshipEdge
	for _, edge := range graph.Edges {
		if edge.SourceActorID == actorID || edge.TargetActorID == actorID {
			edges = append(edges, edge)
		}
	}
	return types.RelationshipGraph{Edges: edges}
}

