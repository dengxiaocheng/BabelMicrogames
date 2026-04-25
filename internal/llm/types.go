package llm

import "babel-runtime/internal/core/types"

type SoloRenderRequest struct {
	RequestID         string                     `json:"request_id"`
	PromptPackID      string                     `json:"prompt_pack_id"`
	TonePackID        string                     `json:"tone_pack_id"`
	ChapterID         string                     `json:"chapter_id"`
	WorldClock        types.WorldClock           `json:"world_clock"`
	PlayerSummary     string                     `json:"player_summary"`
	EnvironmentSummary string                    `json:"environment_summary"`
	RecentSummary     string                     `json:"recent_summary"`
	Consequences      []types.VisibleConsequence `json:"consequences,omitempty"`
	PlayerActionLabel string                     `json:"player_action_label"`
	LegalOptionTags   []string                   `json:"legal_option_tags,omitempty"`
	MaxSceneChars     int                        `json:"max_scene_chars"`
	MaxOptionChars    int                        `json:"max_option_chars"`
	OptionCount       int                        `json:"option_count"`
}

type SoloRenderResponse struct {
	SceneText string               `json:"scene_text"`
	Options   []types.RenderOption `json:"options,omitempty"`
	ToneTags  []string             `json:"tone_tags,omitempty"`
}

