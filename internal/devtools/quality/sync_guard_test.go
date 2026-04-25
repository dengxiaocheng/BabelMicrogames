package quality

import (
	"strings"
	"testing"
)

func TestKernelChangeRequiresKernelDocs(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	errors := ValidateSyncGuard(repoRoot, []string{"internal/kernel/simple.go"}, manifest)
	if len(errors) != 1 || !strings.Contains(errors[0], "kernel_execution_change") {
		t.Fatalf("unexpected errors: %v", errors)
	}
}

func TestModeChangePassesWhenModeDocsAreTouched(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	errors := ValidateSyncGuard(repoRoot, []string{
		"internal/mode/free_chat.go",
		"docs/FEATURE_INHERITANCE.md",
		"docs/ROADMAP.md",
	}, manifest)
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func TestRepositoryChangeCanBeSatisfiedBySharedDoc(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	errors := ValidateSyncGuard(repoRoot, []string{
		"internal/repository/memory.go",
		"docs/INTERFACES.md",
	}, manifest)
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func TestDocsOnlyChangeIsIgnored(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	errors := ValidateSyncGuard(repoRoot, []string{"docs/ROADMAP.md"}, manifest)
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func TestOperationsFlowChangeRequiresOpsDocs(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	errors := ValidateSyncGuard(repoRoot, []string{"internal/ops/issuebridge/command.go"}, manifest)
	if len(errors) != 1 || !strings.Contains(errors[0], "operations_flow_change") {
		t.Fatalf("unexpected errors: %v", errors)
	}
}

func TestCollaborationMCPChangeIsTreatedAsOperationsFlow(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	triggered := TriggeredCategories([]string{"internal/ops/collabmcp/store.go"}, manifest)
	if got := keysOfMap(triggered); len(got) != 1 || got[0] != "operations_flow_change" {
		t.Fatalf("unexpected categories: %v", got)
	}
}

func TestGithubWorkflowChangeIsTreatedAsGovernanceGuard(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	triggered := TriggeredCategories([]string{".github/workflows/docs-sync-guard.yml"}, manifest)
	if got := keysOfMap(triggered); len(got) != 1 || got[0] != "governance_guard_change" {
		t.Fatalf("unexpected categories: %v", got)
	}
}

func TestRuntimeStructureChangeExcludesSpecializedSubsystems(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	triggered := TriggeredCategories([]string{"internal/controlplane/admin.go"}, manifest)
	if got := keysOfMap(triggered); len(got) != 1 || got[0] != "runtime_structure_change" {
		t.Fatalf("unexpected categories: %v", got)
	}
}

func TestAgentChangeUsesAgentSpecificMatrix(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	triggered := TriggeredCategories([]string{"internal/agent/simple.go"}, manifest)
	if got := keysOfMap(triggered); len(got) != 1 || got[0] != "agent_projection_delivery_change" {
		t.Fatalf("unexpected categories: %v", got)
	}
}

func TestRequirementRegistryFoundationChangeHasOwnCategory(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	triggered := TriggeredCategories([]string{"requirements/registry/index.json"}, manifest)
	if got := keysOfMap(triggered); len(got) != 1 || got[0] != "requirement_registry_foundation_change" {
		t.Fatalf("unexpected categories: %v", got)
	}
}

func TestRequirementRegistryCodeChangeHasFoundationCategory(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	triggered := TriggeredCategories([]string{"internal/requirementregistry/filesystem.go"}, manifest)
	if got := keysOfMap(triggered); len(got) != 1 || got[0] != "requirement_registry_foundation_change" {
		t.Fatalf("unexpected categories: %v", got)
	}
}

func TestRequirementAssetChangeHasOwnCategory(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	triggered := TriggeredCategories([]string{"requirements/gameplay-assets/bootstrap.asset.json"}, manifest)
	if got := keysOfMap(triggered); len(got) != 1 || got[0] != "requirement_asset_change" {
		t.Fatalf("unexpected categories: %v", got)
	}
}

func TestGuardReportCommandIsTreatedAsGovernanceGuard(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	triggered := TriggeredCategories([]string{"cmd/babel-dev/main.go"}, manifest)
	if got := keysOfMap(triggered); len(got) != 1 || got[0] != "governance_guard_change" {
		t.Fatalf("unexpected categories: %v", got)
	}
}
