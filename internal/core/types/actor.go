package types

type Stats struct {
	Stamina int `json:"stamina"`
	Spirit  int `json:"spirit"`
	Satiety int `json:"satiety"`
	Health  int `json:"health"`
	Stress  int `json:"stress"`
}

type Skills struct {
	Hauling  int `json:"hauling"`
	Masonry  int `json:"masonry"`
	Ropework int `json:"ropework"`
	Fishing  int `json:"fishing"`
	Crafting int `json:"crafting"`
}

type Traits struct {
	Obedience     int `json:"obedience"`
	Ambition      int `json:"ambition"`
	Homesickness  int `json:"homesickness"`
	Loyalty       int `json:"loyalty"`
	RiskTolerance int `json:"risk_tolerance"`
}

type RoleState struct {
	JobID    string `json:"job_id"`
	JobLevel int    `json:"job_level"`
}

type InventoryEntry struct {
	ItemID      string   `json:"item_id"`
	Quantity    int      `json:"quantity"`
	Durability  int      `json:"durability,omitempty"`
	Quality     int      `json:"quality,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type Condition struct {
	ConditionID string `json:"condition_id"`
	Stacks      int    `json:"stacks"`
}

type ActorLocation struct {
	ZoneID    string `json:"zone_id"`
	SubzoneID string `json:"subzone_id"`
}

type PlayerMeta struct {
	DaysWorked int `json:"days_worked"`
}

type PlayerState struct {
	ActorID     string           `json:"actor_id"`
	Stats       Stats            `json:"stats"`
	Skills      Skills           `json:"skills"`
	Traits      Traits           `json:"traits"`
	Role        RoleState        `json:"role"`
	Inventory   []InventoryEntry `json:"inventory"`
	Capacity    int              `json:"capacity"`
	Conditions  []Condition      `json:"conditions,omitempty"`
	Location    ActorLocation    `json:"location"`
	Meta        PlayerMeta       `json:"meta"`
}

type Objective struct {
	ObjectiveID string `json:"objective_id"`
	Tag         string `json:"tag"`
	Priority    int    `json:"priority"`
}

type PrivateState struct {
	ActorID             string               `json:"actor_id"`
	HiddenFlags         map[string]FlagValue `json:"hidden_flags,omitempty"`
	PrivateObservations []string             `json:"private_observations,omitempty"`
	PrivateInventory    []string             `json:"private_inventory,omitempty"`
	SecretObjectives    []Objective          `json:"secret_objectives,omitempty"`
}

