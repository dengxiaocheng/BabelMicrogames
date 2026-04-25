package types

type RelationshipEdge struct {
	SourceActorID string   `json:"source_actor_id"`
	TargetActorID string   `json:"target_actor_id"`
	Affinity      int      `json:"affinity"`
	Trust         int      `json:"trust"`
	Fear          int      `json:"fear"`
	Debt          int      `json:"debt"`
	Tags          []string `json:"tags,omitempty"`
}

type RelationshipGraph struct {
	Edges []RelationshipEdge `json:"edges"`
}

type RelationshipDelta struct {
	SourceActorID string `json:"source_actor_id"`
	TargetActorID string `json:"target_actor_id"`
	AffinityDelta int    `json:"affinity_delta"`
	TrustDelta    int    `json:"trust_delta"`
	FearDelta     int    `json:"fear_delta"`
	DebtDelta     int    `json:"debt_delta"`
}

