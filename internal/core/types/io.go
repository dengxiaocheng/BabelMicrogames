package types

type SoloStepInput struct {
	Session SoloSession   `json:"session"`
	Action  PendingAction `json:"action"`
}

type RoomStepInput struct {
	Room    MultiplayerRoom `json:"room"`
	Actions []PendingAction `json:"actions"`
}

type StepDelta struct {
	NextClock         WorldClock            `json:"next_clock"`
	PlayerState       *PlayerState          `json:"player_state,omitempty"`
	SharedState       *SharedSceneState     `json:"shared_state,omitempty"`
	PrivateStates     map[string]PrivateState `json:"private_states,omitempty"`
	EnvironmentState  *EnvironmentState     `json:"environment_state,omitempty"`
	RelationshipState *RelationshipGraph    `json:"relationship_state,omitempty"`
	Consequences      []VisibleConsequence  `json:"consequences,omitempty"`
}

type StepResult struct {
	RuntimeID    string       `json:"runtime_id"`
	StateVersion int64        `json:"state_version"`
	Delta        StepDelta    `json:"delta"`
	Checkpoint   string       `json:"checkpoint"`
}

type TurnResult struct {
	RoomID       string       `json:"room_id"`
	StateVersion int64        `json:"state_version"`
	Delta        StepDelta    `json:"delta"`
	Checkpoint   string       `json:"checkpoint"`
}

type ActionSpec struct {
	ActionType string            `json:"action_type"`
	TargetIDs  []string          `json:"target_ids,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

