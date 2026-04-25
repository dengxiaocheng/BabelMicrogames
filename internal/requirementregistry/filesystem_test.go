package requirementregistry_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"babel-runtime/internal/requirementregistry"
)

func TestFilesystemRegistryResolvesBootstrapAssets(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	registry := requirementregistry.FilesystemRegistry{RepoRoot: repoRoot}

	ruleset, err := registry.ResolveRuleset(context.Background(), "bootstrap.ruleset")
	if err != nil {
		t.Fatalf("ResolveRuleset returned error: %v", err)
	}
	if ruleset.Version != "0.1.0" {
		t.Fatalf("expected ruleset version 0.1.0, got %q", ruleset.Version)
	}

	promptPack, err := registry.ResolvePromptPack(context.Background(), "bootstrap.prompt_pack")
	if err != nil {
		t.Fatalf("ResolvePromptPack returned error: %v", err)
	}
	if promptPack.PromptPackID != "bootstrap.prompt_pack" {
		t.Fatalf("unexpected prompt pack id: %q", promptPack.PromptPackID)
	}

	gameplayAsset, err := registry.ResolveGameplayAsset(context.Background(), "bootstrap.gameplay_asset")
	if err != nil {
		t.Fatalf("ResolveGameplayAsset returned error: %v", err)
	}
	if len(gameplayAsset.LinkedAssets) != 3 {
		t.Fatalf("expected linked assets, got %d", len(gameplayAsset.LinkedAssets))
	}

	constraint, err := registry.ResolveContentConstraint(context.Background(), "bootstrap.content_constraint")
	if err != nil {
		t.Fatalf("ResolveContentConstraint returned error: %v", err)
	}
	if constraint.ContentConstraintID != "bootstrap.content_constraint" {
		t.Fatalf("unexpected constraint id: %q", constraint.ContentConstraintID)
	}
}

func TestFilesystemRegistryReturnsNotFoundForUnknownAsset(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	registry := requirementregistry.FilesystemRegistry{RepoRoot: repoRoot}

	_, err := registry.ResolveRuleset(context.Background(), "missing.ruleset")
	if !errors.Is(err, requirementregistry.ErrAssetNotFound) {
		t.Fatalf("expected ErrAssetNotFound, got %v", err)
	}
}
