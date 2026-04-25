package mapcore

import "testing"

func TestAdjacencyAndCost(t *testing.T) {
	core := NewSimpleCore()
	core.Link("yard", "tower", 3)
	if !core.IsAdjacent("yard", "tower") {
		t.Fatal("expected adjacency")
	}
	if cost := core.TravelCost("yard", "tower"); cost != 3 {
		t.Fatalf("expected cost 3, got %d", cost)
	}
}

