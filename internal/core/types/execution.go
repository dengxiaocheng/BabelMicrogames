package types

import "encoding/json"

type ModeID string

const (
	ModeFreeChat       ModeID = "free_chat"
	ModeProjectConsult ModeID = "project_consult"
	ModeSoloScene      ModeID = "solo_scene"
	ModeRoomScene      ModeID = "room_scene"
)

type ExecutionStage string

const (
	ExecutionAccepted          ExecutionStage = "accepted"
	ExecutionPlanned           ExecutionStage = "planned"
	ExecutionAwaitingArtifacts ExecutionStage = "awaiting_artifacts"
	ExecutionSettled           ExecutionStage = "settled"
	ExecutionProjected         ExecutionStage = "projected"
	ExecutionCommitted         ExecutionStage = "committed"
	ExecutionDelivered         ExecutionStage = "delivered"
	ExecutionFailed            ExecutionStage = "failed"
)

type InboundEnvelope struct {
	EnvelopeID     string `json:"envelope_id"`
	RuntimeID      string `json:"runtime_id"`
	UserID         string `json:"user_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Transport      string `json:"transport,omitempty"`
	RouteHint      string `json:"route_hint,omitempty"`
	Text           string `json:"text"`
	ReceivedAtUnix int64  `json:"received_at_unix,omitempty"`
}

type RuntimeRecord struct {
	RuntimeID     string        `json:"runtime_id"`
	ModeID        ModeID        `json:"mode_id"`
	SchemaVersion int           `json:"schema_version"`
	HeadVersion   int64         `json:"head_version"`
	RulesetID     string        `json:"ruleset_id,omitempty"`
	PromptPackID  string        `json:"prompt_pack_id,omitempty"`
	GameplayAssetID string      `json:"gameplay_asset_id,omitempty"`
	Status        RuntimeStatus `json:"status"`
}

type RuntimeSnapshot struct {
	RuntimeID     string          `json:"runtime_id"`
	ModeID        ModeID          `json:"mode_id"`
	SchemaVersion int             `json:"schema_version"`
	Version       int64           `json:"version"`
	State         json.RawMessage `json:"state"`
	UpdatedAtUnix int64           `json:"updated_at_unix,omitempty"`
}

type ModeCommand struct {
	CommandType string `json:"command_type"`
	Text        string `json:"text,omitempty"`
}

type ExecutionRecord struct {
	ExecutionID        string         `json:"execution_id"`
	RuntimeID          string         `json:"runtime_id"`
	EnvelopeID         string         `json:"envelope_id"`
	ActorID            string         `json:"actor_id,omitempty"`
	Transport          string         `json:"transport,omitempty"`
	IdempotencyKey     string         `json:"idempotency_key"`
	ModeID             ModeID         `json:"mode_id"`
	Stage              ExecutionStage `json:"stage"`
	CommandType        string         `json:"command_type,omitempty"`
	CommandText        string         `json:"command_text,omitempty"`
	LeaseOwner         string         `json:"lease_owner,omitempty"`
	LeaseExpiresAtUnix int64          `json:"lease_expires_at_unix,omitempty"`
	AcceptedAtUnix     int64          `json:"accepted_at_unix,omitempty"`
	UpdatedAtUnix      int64          `json:"updated_at_unix,omitempty"`
	LastError          string         `json:"last_error,omitempty"`
	PendingDelivery    []DeliveryPlan `json:"pending_delivery,omitempty"`
}

type ExecutionTicket struct {
	ExecutionID string         `json:"execution_id"`
	RuntimeID   string         `json:"runtime_id"`
	Stage       ExecutionStage `json:"stage"`
	Reused      bool           `json:"reused"`
}

type RuntimeEvent struct {
	EventID        string `json:"event_id"`
	RuntimeID      string `json:"runtime_id"`
	ExecutionID    string `json:"execution_id,omitempty"`
	Kind           string `json:"kind"`
	Text           string `json:"text,omitempty"`
	OccurredAtUnix int64  `json:"occurred_at_unix,omitempty"`
}

type ModeExecutionInput struct {
	Runtime   RuntimeRecord    `json:"runtime"`
	Execution ExecutionRecord  `json:"execution"`
	Snapshot  *RuntimeSnapshot `json:"snapshot,omitempty"`
	Requirements RuntimeRequirements `json:"requirements,omitempty"`
	Artifacts []AgentArtifact  `json:"artifacts,omitempty"`
	Command   ModeCommand      `json:"command"`
}

type ModeExecutionResult struct {
	NextStage           ExecutionStage   `json:"next_stage"`
	RuntimeVersionDelta int64            `json:"runtime_version_delta,omitempty"`
	Snapshot            *RuntimeSnapshot `json:"snapshot,omitempty"`
	AgentTasks          []AgentTaskSpec  `json:"agent_tasks,omitempty"`
	Events              []RuntimeEvent   `json:"events,omitempty"`
}

type DeterministicRequest struct {
	RuntimeID   string `json:"runtime_id"`
	ExecutionID string `json:"execution_id"`
	ModeID      ModeID `json:"mode_id"`
	Command     string `json:"command,omitempty"`
}

type DeterministicResult struct {
	RuntimeVersionDelta int64          `json:"runtime_version_delta,omitempty"`
	Events              []RuntimeEvent `json:"events,omitempty"`
}

type AgentTaskSpec struct {
	TaskID       string `json:"task_id"`
	TaskType     string `json:"task_type"`
	RuntimeID    string `json:"runtime_id"`
	Input        string `json:"input,omitempty"`
	ArtifactType string `json:"artifact_type,omitempty"`
}

type AgentTaskStatus string

const (
	AgentTaskQueued    AgentTaskStatus = "queued"
	AgentTaskCompleted AgentTaskStatus = "completed"
	AgentTaskFailed    AgentTaskStatus = "failed"
)

type AgentTaskRecord struct {
	TaskID          string          `json:"task_id"`
	ExecutionID     string          `json:"execution_id"`
	RuntimeID       string          `json:"runtime_id"`
	TaskType        string          `json:"task_type"`
	Input           string          `json:"input,omitempty"`
	ArtifactType    string          `json:"artifact_type,omitempty"`
	Status          AgentTaskStatus `json:"status"`
	CreatedAtUnix   int64           `json:"created_at_unix,omitempty"`
	CompletedAtUnix int64           `json:"completed_at_unix,omitempty"`
	LastError       string          `json:"last_error,omitempty"`
}

type AgentArtifact struct {
	ArtifactID   string `json:"artifact_id"`
	TaskID       string `json:"task_id,omitempty"`
	ExecutionID  string `json:"execution_id"`
	RuntimeID    string `json:"runtime_id,omitempty"`
	ArtifactType string `json:"artifact_type"`
	Body         string `json:"body"`
}

type ProjectionInput struct {
	Runtime   RuntimeRecord    `json:"runtime"`
	Execution ExecutionRecord  `json:"execution"`
	Snapshot  *RuntimeSnapshot `json:"snapshot,omitempty"`
}

type ProjectionResult struct {
	Frames        []ProjectionFrame `json:"frames,omitempty"`
	DeliveryPlans []DeliveryPlan    `json:"delivery_plans,omitempty"`
}

type DeliveryPlan struct {
	PlanID      string `json:"plan_id"`
	RuntimeID   string `json:"runtime_id"`
	ExecutionID string `json:"execution_id"`
	RecipientID string `json:"recipient_id,omitempty"`
	Transport   string `json:"transport"`
	FrameID     string `json:"frame_id,omitempty"`
	Payload     string `json:"payload,omitempty"`
}

type ProjectionFrame struct {
	FrameID       string `json:"frame_id"`
	RuntimeID     string `json:"runtime_id"`
	ExecutionID   string `json:"execution_id"`
	ModeID        ModeID `json:"mode_id"`
	Body          string `json:"body"`
	CreatedAtUnix int64  `json:"created_at_unix,omitempty"`
}

type DeliveryJobStatus string

const (
	DeliveryQueued    DeliveryJobStatus = "queued"
	DeliveryDelivered DeliveryJobStatus = "delivered"
	DeliveryFailed    DeliveryJobStatus = "failed"
)

type DeliveryJob struct {
	JobID          string            `json:"job_id"`
	PlanID         string            `json:"plan_id"`
	RuntimeID      string            `json:"runtime_id"`
	ExecutionID    string            `json:"execution_id"`
	RecipientID    string            `json:"recipient_id,omitempty"`
	Transport      string            `json:"transport"`
	FrameID        string            `json:"frame_id,omitempty"`
	Payload        string            `json:"payload,omitempty"`
	Status         DeliveryJobStatus `json:"status"`
	EnqueuedAtUnix int64             `json:"enqueued_at_unix,omitempty"`
}

type RecoveryReport struct {
	ResumedExecutions int `json:"resumed_executions"`
	FailedExecutions  int `json:"failed_executions"`
}
