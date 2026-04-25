package requirementregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"babel-runtime/internal/core/types"
)

type FilesystemRegistry struct {
	RepoRoot string
}

type registryIndex struct {
	SchemaVersion int                  `json:"schema_version"`
	Assets        []registryAssetEntry `json:"assets"`
}

type registryAssetEntry struct {
	AssetID  string `json:"asset_id"`
	FamilyID string `json:"family_id"`
	Kind     string `json:"kind"`
	Revision string `json:"revision"`
	Path     string `json:"path"`
}

type assetDocument struct {
	SchemaVersion int      `json:"schema_version"`
	AssetID       string   `json:"asset_id"`
	FamilyID      string   `json:"family_id"`
	Kind          string   `json:"kind"`
	Revision      string   `json:"revision"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	RuntimeModes  []string `json:"runtime_modes"`
	TestRefs      []string `json:"test_refs"`
	SourceTrace   []string `json:"source_trace"`
	LinkedAssets  []string `json:"linked_assets,omitempty"`
	DerivedAssets []string `json:"derived_assets,omitempty"`
}

func (r FilesystemRegistry) ResolveRuleset(ctx context.Context, rulesetID string) (types.RulesetBundle, error) {
	entry, doc, raw, err := r.loadAsset(ctx, rulesetID, "ruleset")
	if err != nil {
		return types.RulesetBundle{}, err
	}
	return types.RulesetBundle{
		RulesetID:    entry.AssetID,
		Version:      doc.Revision,
		Status:       doc.Status,
		Summary:      doc.Summary,
		RuntimeModes: append([]string(nil), doc.RuntimeModes...),
		TestRefs:     append([]string(nil), doc.TestRefs...),
		SourceTrace:  append([]string(nil), doc.SourceTrace...),
		Document:     raw,
	}, nil
}

func (r FilesystemRegistry) ResolvePromptPack(ctx context.Context, promptPackID string) (types.PromptPackBundle, error) {
	entry, doc, raw, err := r.loadAsset(ctx, promptPackID, "prompt_pack")
	if err != nil {
		return types.PromptPackBundle{}, err
	}
	return types.PromptPackBundle{
		PromptPackID: entry.AssetID,
		Version:      doc.Revision,
		Status:       doc.Status,
		Summary:      doc.Summary,
		RuntimeModes: append([]string(nil), doc.RuntimeModes...),
		TestRefs:     append([]string(nil), doc.TestRefs...),
		SourceTrace:  append([]string(nil), doc.SourceTrace...),
		Document:     raw,
	}, nil
}

func (r FilesystemRegistry) ResolveGameplayAsset(ctx context.Context, assetID string) (types.GameplayAssetBundle, error) {
	entry, doc, raw, err := r.loadAsset(ctx, assetID, "gameplay_asset")
	if err != nil {
		return types.GameplayAssetBundle{}, err
	}
	return types.GameplayAssetBundle{
		GameplayAssetID: entry.AssetID,
		Version:         doc.Revision,
		Status:          doc.Status,
		Summary:         doc.Summary,
		RuntimeModes:    append([]string(nil), doc.RuntimeModes...),
		TestRefs:        append([]string(nil), doc.TestRefs...),
		SourceTrace:     append([]string(nil), doc.SourceTrace...),
		LinkedAssets:    append([]string(nil), doc.LinkedAssets...),
		Document:        raw,
	}, nil
}

func (r FilesystemRegistry) ResolveContentConstraint(ctx context.Context, constraintID string) (types.ContentConstraintBundle, error) {
	entry, doc, raw, err := r.loadAsset(ctx, constraintID, "content_constraint")
	if err != nil {
		return types.ContentConstraintBundle{}, err
	}
	return types.ContentConstraintBundle{
		ContentConstraintID: entry.AssetID,
		Version:             doc.Revision,
		Status:              doc.Status,
		Summary:             doc.Summary,
		RuntimeModes:        append([]string(nil), doc.RuntimeModes...),
		TestRefs:            append([]string(nil), doc.TestRefs...),
		SourceTrace:         append([]string(nil), doc.SourceTrace...),
		Document:            raw,
	}, nil
}

func (r FilesystemRegistry) ResolveExperimentTrace(ctx context.Context, traceID string) (types.ExperimentTraceBundle, error) {
	entry, doc, raw, err := r.loadAsset(ctx, traceID, "experiment_trace")
	if err != nil {
		return types.ExperimentTraceBundle{}, err
	}
	return types.ExperimentTraceBundle{
		ExperimentTraceID: entry.AssetID,
		Version:           doc.Revision,
		Status:            doc.Status,
		Summary:           doc.Summary,
		RuntimeModes:      append([]string(nil), doc.RuntimeModes...),
		TestRefs:          append([]string(nil), doc.TestRefs...),
		SourceTrace:       append([]string(nil), doc.SourceTrace...),
		DerivedAssets:     append([]string(nil), doc.DerivedAssets...),
		Document:          raw,
	}, nil
}

func (r FilesystemRegistry) loadAsset(ctx context.Context, assetID, expectedKind string) (registryAssetEntry, assetDocument, json.RawMessage, error) {
	_ = ctx
	if assetID == "" {
		return registryAssetEntry{}, assetDocument{}, nil, fmt.Errorf("missing asset id")
	}
	index, err := r.loadIndex()
	if err != nil {
		return registryAssetEntry{}, assetDocument{}, nil, err
	}
	for _, entry := range index.Assets {
		if entry.AssetID != assetID {
			continue
		}
		if expectedKind != "" && entry.Kind != expectedKind {
			return registryAssetEntry{}, assetDocument{}, nil, fmt.Errorf("asset %s is kind %s, want %s", assetID, entry.Kind, expectedKind)
		}
		raw, doc, err := r.loadAssetDocument(entry)
		if err != nil {
			return registryAssetEntry{}, assetDocument{}, nil, err
		}
		return entry, doc, raw, nil
	}
	return registryAssetEntry{}, assetDocument{}, nil, fmt.Errorf("%w: %s", ErrAssetNotFound, assetID)
}

func (r FilesystemRegistry) loadIndex() (registryIndex, error) {
	registryPath := filepath.Join(r.repoRoot(), "requirements", "registry", "index.json")
	payload, err := os.ReadFile(registryPath)
	if err != nil {
		return registryIndex{}, err
	}
	var index registryIndex
	if err := json.Unmarshal(payload, &index); err != nil {
		return registryIndex{}, err
	}
	return index, nil
}

func (r FilesystemRegistry) loadAssetDocument(entry registryAssetEntry) (json.RawMessage, assetDocument, error) {
	assetPath := filepath.Join(r.repoRoot(), filepath.FromSlash(entry.Path))
	payload, err := os.ReadFile(assetPath)
	if err != nil {
		return nil, assetDocument{}, err
	}
	var doc assetDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil, assetDocument{}, err
	}
	return append(json.RawMessage(nil), payload...), doc, nil
}

func (r FilesystemRegistry) repoRoot() string {
	if r.RepoRoot != "" {
		return r.RepoRoot
	}
	return "."
}
