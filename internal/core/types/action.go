package types

type PendingAction struct {
	ActionID        string            `json:"action_id"`
	UserInput       string            `json:"user_input"`
	ActionType      string            `json:"action_type"`
	TargetIDs       []string          `json:"target_ids,omitempty"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	IdempotencyKey  string            `json:"idempotency_key"`
}

type PendingTurn struct {
	TurnID            string                    `json:"turn_id"`
	Segment           string                    `json:"segment"`
	RequiredPlayers   []string                  `json:"required_players"`
	SubmittedActions  map[string]PendingAction  `json:"submitted_actions"`
}

type VisibleConsequence struct {
	Tag      string `json:"tag"`
	TextHint string `json:"text_hint"`
	Severity int    `json:"severity"`
}

