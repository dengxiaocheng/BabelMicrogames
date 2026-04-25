package relcore

import (
	"testing"

	"babel-runtime/internal/core/types"
)

func TestApplyRelationshipDelta(t *testing.T) {
	core := SimpleCore{}
	graph := types.RelationshipGraph{}
	out := core.Apply(graph, []types.RelationshipDelta{{
		SourceActorID: "a",
		TargetActorID: "b",
		TrustDelta:    2,
	}})
	if len(out.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(out.Edges))
	}
	if out.Edges[0].Trust != 2 {
		t.Fatalf("expected trust 2, got %d", out.Edges[0].Trust)
	}
}

