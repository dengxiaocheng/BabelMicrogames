package mapcore

type Edge struct {
	TargetZoneID  string   `json:"target_zone_id"`
	TraversalCost int      `json:"traversal_cost"`
	TraversalTags []string `json:"traversal_tags,omitempty"`
}

type HazardProfile struct {
	HazardLevel int      `json:"hazard_level"`
	Tags        []string `json:"tags,omitempty"`
}

type ResourceProfile struct {
	ResourceIDs []string `json:"resource_ids,omitempty"`
}

type Zone struct {
	ZoneID          string          `json:"zone_id"`
	ZoneType        string          `json:"zone_type"`
	Tags            []string        `json:"tags,omitempty"`
	Neighbors       []Edge          `json:"neighbors,omitempty"`
	HazardProfile   HazardProfile   `json:"hazard_profile"`
	ResourceProfile ResourceProfile `json:"resource_profile"`
}

