package quality

import "testing"

func TestManifestSemanticsAreComplete(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if errors := ValidateManifestSemantics(manifest); len(errors) > 0 {
		t.Fatalf("ValidateManifestSemantics returned errors: %v", errors)
	}
}

func TestManifestPathsExist(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if errors := ValidateManifestPaths(repoRoot, manifest); len(errors) > 0 {
		t.Fatalf("ValidateManifestPaths returned errors: %v", errors)
	}
}

func TestTopLevelDocsMatchManifest(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if errors := ValidateTopLevelDocs(repoRoot, manifest); len(errors) > 0 {
		t.Fatalf("ValidateTopLevelDocs returned errors: %v", errors)
	}
}

func TestModuleDocsMatchManifest(t *testing.T) {
	repoRoot := repoRootForTest(t)
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if errors := ValidateModuleDocs(repoRoot, manifest); len(errors) > 0 {
		t.Fatalf("ValidateModuleDocs returned errors: %v", errors)
	}
}
