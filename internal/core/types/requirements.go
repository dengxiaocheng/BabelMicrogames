package types

import "encoding/json"

type RuntimeRequirements struct {
	Ruleset           *RulesetBundle           `json:"ruleset,omitempty"`
	PromptPack        *PromptPackBundle        `json:"prompt_pack,omitempty"`
	GameplayAsset     *GameplayAssetBundle     `json:"gameplay_asset,omitempty"`
	ContentConstraint *ContentConstraintBundle `json:"content_constraint,omitempty"`
}

type RulesetBundle struct {
	RulesetID    string          `json:"ruleset_id"`
	Version      string          `json:"version,omitempty"`
	Status       string          `json:"status,omitempty"`
	Summary      string          `json:"summary,omitempty"`
	RuntimeModes []string        `json:"runtime_modes,omitempty"`
	TestRefs     []string        `json:"test_refs,omitempty"`
	SourceTrace  []string        `json:"source_trace,omitempty"`
	Document     json.RawMessage `json:"document,omitempty"`
}

type PromptPackBundle struct {
	PromptPackID string          `json:"prompt_pack_id"`
	Version      string          `json:"version,omitempty"`
	Status       string          `json:"status,omitempty"`
	Summary      string          `json:"summary,omitempty"`
	RuntimeModes []string        `json:"runtime_modes,omitempty"`
	TestRefs     []string        `json:"test_refs,omitempty"`
	SourceTrace  []string        `json:"source_trace,omitempty"`
	Document     json.RawMessage `json:"document,omitempty"`
}

type GameplayAssetBundle struct {
	GameplayAssetID string          `json:"gameplay_asset_id"`
	Version         string          `json:"version,omitempty"`
	Status          string          `json:"status,omitempty"`
	Summary         string          `json:"summary,omitempty"`
	RuntimeModes    []string        `json:"runtime_modes,omitempty"`
	TestRefs        []string        `json:"test_refs,omitempty"`
	SourceTrace     []string        `json:"source_trace,omitempty"`
	LinkedAssets    []string        `json:"linked_assets,omitempty"`
	Document        json.RawMessage `json:"document,omitempty"`
}

type ContentConstraintBundle struct {
	ContentConstraintID string          `json:"content_constraint_id"`
	Version             string          `json:"version,omitempty"`
	Status              string          `json:"status,omitempty"`
	Summary             string          `json:"summary,omitempty"`
	RuntimeModes        []string        `json:"runtime_modes,omitempty"`
	TestRefs            []string        `json:"test_refs,omitempty"`
	SourceTrace         []string        `json:"source_trace,omitempty"`
	Document            json.RawMessage `json:"document,omitempty"`
}

type ExperimentTraceBundle struct {
	ExperimentTraceID string          `json:"experiment_trace_id"`
	Version           string          `json:"version,omitempty"`
	Status            string          `json:"status,omitempty"`
	Summary           string          `json:"summary,omitempty"`
	RuntimeModes      []string        `json:"runtime_modes,omitempty"`
	TestRefs          []string        `json:"test_refs,omitempty"`
	SourceTrace       []string        `json:"source_trace,omitempty"`
	DerivedAssets     []string        `json:"derived_assets,omitempty"`
	Document          json.RawMessage `json:"document,omitempty"`
}
