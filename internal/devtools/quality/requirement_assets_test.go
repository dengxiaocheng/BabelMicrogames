package quality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoRequirementAssetsAreValid(t *testing.T) {
	repoRoot := repoRootForTest(t)
	if errors := ValidateRequirementAssets(repoRoot); len(errors) > 0 {
		t.Fatalf("ValidateRequirementAssets returned errors: %v", errors)
	}
}

func TestUnregisteredAssetFileIsReported(t *testing.T) {
	repoRoot := repoRootForTest(t)
	tempRoot := copyRequirementsFixture(t, repoRoot)
	extraPath := filepath.Join(tempRoot, "requirements", "rulesets", "extra.ruleset.json")
	writeJSONFile(t, extraPath, map[string]any{
		"schema_version": 1,
		"asset_id":       "extra.ruleset",
		"family_id":      "rulesets",
		"kind":           "ruleset",
		"revision":       "0.1.0",
		"status":         "draft",
		"summary":        "extra",
		"runtime_modes":  []string{},
		"test_refs":      []string{},
		"source_trace":   []string{},
	})
	errors := ValidateRequirementAssets(tempRoot)
	if !containsError(errors, "未登记 asset 文件") {
		t.Fatalf("expected unregistered asset error, got %v", errors)
	}
}

func TestBrokenReferenceIsReported(t *testing.T) {
	repoRoot := repoRootForTest(t)
	tempRoot := copyRequirementsFixture(t, repoRoot)
	gameplayAssetPath := filepath.Join(tempRoot, "requirements", "gameplay-assets", "bootstrap.asset.json")
	payload, err := os.ReadFile(gameplayAssetPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	var gameplayAsset map[string]any
	if err := json.Unmarshal(payload, &gameplayAsset); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	gameplayAsset["linked_assets"] = append(stringListValue(gameplayAsset["linked_assets"]), "missing.asset")
	writeJSONFile(t, gameplayAssetPath, gameplayAsset)
	errors := ValidateRequirementAssets(tempRoot)
	if !containsError(errors, "未登记的关联资产") {
		t.Fatalf("expected broken reference error, got %v", errors)
	}
}

func copyRequirementsFixture(t *testing.T, repoRoot string) string {
	t.Helper()
	tempRoot := t.TempDir()
	sourceRoot := filepath.Join(repoRoot, "requirements")
	targetRoot := filepath.Join(tempRoot, "requirements")
	if err := filepath.Walk(sourceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, payload, 0o644)
	}); err != nil {
		t.Fatalf("copy requirements fixture: %v", err)
	}
	return tempRoot
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent returned error: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func containsError(errors []string, fragment string) bool {
	for _, message := range errors {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}
