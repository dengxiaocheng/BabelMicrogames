package types

type CheckpointStatus string

const (
	CheckpointPendingInput  CheckpointStatus = "pending_input"
	CheckpointValidating    CheckpointStatus = "validating"
	CheckpointSimulating    CheckpointStatus = "simulating"
	CheckpointPersisting    CheckpointStatus = "persisting_state"
	CheckpointRendering     CheckpointStatus = "rendering"
	CheckpointDelivering    CheckpointStatus = "delivering"
	CheckpointCommitted     CheckpointStatus = "committed"
	CheckpointFailed        CheckpointStatus = "failed"
)

type ResumePolicy string

const (
	ResumeRetryStep    ResumePolicy = "retry_step"
	ResumeRerenderOnly ResumePolicy = "rerender_only"
	ResumeRedeliver    ResumePolicy = "redeliver_only"
	ResumeManual       ResumePolicy = "manual_intervention"
)

type InputEvent struct {
	EventID         string      `json:"event_id"`
	RuntimeType     RuntimeType `json:"runtime_type"`
	RuntimeID       string      `json:"runtime_id"`
	ActorID         string      `json:"actor_id"`
	IdempotencyKey  string      `json:"idempotency_key"`
	Payload         []byte      `json:"payload"`
}

type RuntimeCheckpoint struct {
	CheckpointID       string           `json:"checkpoint_id"`
	RuntimeType        RuntimeType      `json:"runtime_type"`
	RuntimeID          string           `json:"runtime_id"`
	StateVersionBefore int64            `json:"state_version_before"`
	StateVersionAfter  int64            `json:"state_version_after,omitempty"`
	InputEventID       string           `json:"input_event_id"`
	StepName           string           `json:"step_name"`
	Status             CheckpointStatus `json:"status"`
	ResumePolicy       ResumePolicy     `json:"resume_policy"`
	LeaseOwner         string           `json:"lease_owner,omitempty"`
	LeaseExpiresAtUnix int64            `json:"lease_expires_at_unix,omitempty"`
}

type RenderOption struct {
	OptionID  string `json:"option_id"`
	Label     string `json:"label"`
	ActionTag string `json:"action_tag"`
}

type RenderFrame struct {
	FrameID            string         `json:"frame_id"`
	OwnerType          string         `json:"owner_type"`
	OwnerID            string         `json:"owner_id"`
	BasedOnActionID    string         `json:"based_on_action_id,omitempty"`
	BasedOnTurnID      string         `json:"based_on_turn_id,omitempty"`
	StateVersion       int64          `json:"state_version"`
	VisibleText        string         `json:"visible_text"`
	Options            []RenderOption `json:"options,omitempty"`
}

