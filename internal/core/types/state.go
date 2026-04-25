package types

type RuntimeType string

const (
	RuntimeSolo        RuntimeType = "solo"
	RuntimeMultiplayer RuntimeType = "multiplayer"
)

type RuntimeStatus string

const (
	RuntimeStatusActive     RuntimeStatus = "active"
	RuntimeStatusPaused     RuntimeStatus = "paused"
	RuntimeStatusArchived   RuntimeStatus = "archived"
	RuntimeStatusLobby      RuntimeStatus = "lobby"
	RuntimeStatusRecovering RuntimeStatus = "recovering"
)

type WorldClock struct {
	DayIndex     int    `json:"day_index"`
	Segment      string `json:"segment"`
	AbsoluteTick int64  `json:"absolute_tick"`
}

type SeedState struct {
	WorldSeed  uint64 `json:"world_seed"`
	RNGCounter uint64 `json:"rng_counter"`
}

type FlagKind string

const (
	FlagBool   FlagKind = "bool"
	FlagInt    FlagKind = "int"
	FlagString FlagKind = "string"
	FlagSet    FlagKind = "set"
)

type FlagValue struct {
	Kind        FlagKind `json:"kind"`
	BoolValue   bool     `json:"bool_value,omitempty"`
	IntValue    int      `json:"int_value,omitempty"`
	StringValue string   `json:"string_value,omitempty"`
	SetValue    []string `json:"set_value,omitempty"`
}

type SoloSession struct {
	SessionID           string               `json:"session_id"`
	UserID              string               `json:"user_id"`
	SchemaVersion       int                  `json:"schema_version"`
	StateVersion        int64                `json:"state_version"`
	RulesetID           string               `json:"ruleset_id"`
	PromptPackID        string               `json:"prompt_pack_id"`
	Status              RuntimeStatus        `json:"status"`
	ChapterID           string               `json:"chapter_id"`
	WorldClock          WorldClock           `json:"world_clock"`
	SeedState           SeedState            `json:"seed_state"`
	PlayerState         PlayerState          `json:"player_state"`
	EnvironmentState    EnvironmentState     `json:"environment_state"`
	RelationshipState   RelationshipGraph    `json:"relationship_state"`
	EventFlags          map[string]FlagValue `json:"event_flags"`
	PendingAction       *PendingAction       `json:"pending_action,omitempty"`
	LastCommittedAction string               `json:"last_committed_action,omitempty"`
	LastActionText      string               `json:"last_action_text,omitempty"`
	LastRenderFrameID   string               `json:"last_render_frame_id,omitempty"`
}

type MultiplayerRoom struct {
	RoomID              string                  `json:"room_id"`
	SchemaVersion       int                     `json:"schema_version"`
	StateVersion        int64                   `json:"state_version"`
	RulesetID           string                  `json:"ruleset_id"`
	PromptPackID        string                  `json:"prompt_pack_id"`
	Status              RuntimeStatus           `json:"status"`
	ChapterID           string                  `json:"chapter_id"`
	WorldClock          WorldClock              `json:"world_clock"`
	SeedState           SeedState               `json:"seed_state"`
	Roster              []RoomMember            `json:"roster"`
	SharedState         SharedSceneState        `json:"shared_state"`
	HiddenState         map[string]PrivateState `json:"hidden_state"`
	EventFlags          map[string]FlagValue    `json:"event_flags"`
	PendingTurn         *PendingTurn            `json:"pending_turn,omitempty"`
	LastCommittedTurnID string                  `json:"last_committed_turn_id,omitempty"`
	LastRenderFrameIDs  map[string]string       `json:"last_render_frame_ids,omitempty"`
}

type RoomMember struct {
	PlayerID       string `json:"player_id"`
	DisplayName    string `json:"display_name"`
	Ready          bool   `json:"ready"`
	RoleID         string `json:"role_id"`
	LastActiveUnix int64  `json:"last_active_unix,omitempty"`
}
