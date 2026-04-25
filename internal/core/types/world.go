package types

type Modifier struct {
	ModifierID string `json:"modifier_id"`
	Scope      string `json:"scope"`
	TargetID   string `json:"target_id,omitempty"`
	EffectType string `json:"effect_type"`
	Magnitude  int    `json:"magnitude"`
}

type EnvironmentState struct {
	SiteID           string     `json:"site_id"`
	TowerHeightLevel int        `json:"tower_height_level"`
	WeatherID        string     `json:"weather_id"`
	HazardLevel      int        `json:"hazard_level"`
	FoodSupplyLevel  int        `json:"food_supply_level"`
	SupervisionLevel int        `json:"supervision_level"`
	MoralePressure   int        `json:"morale_pressure"`
	ActiveModifiers  []Modifier `json:"active_modifiers,omitempty"`
}

type SharedSceneState struct {
	ZoneID           string            `json:"zone_id"`
	StageID          string            `json:"stage_id"`
	WorkerGroupIDs   []string          `json:"worker_group_ids,omitempty"`
	PublicHazards    []string          `json:"public_hazards,omitempty"`
	PublicEvents     []string          `json:"public_events,omitempty"`
	AggregateMetrics map[string]int    `json:"aggregate_metrics,omitempty"`
}

