package quality

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Manifest struct {
	CanonicalTopLevelDocs []string              `json:"canonical_top_level_docs"`
	Modules               map[string]ModuleSpec `json:"modules"`
	TopicRoutes           map[string][]string   `json:"topic_routes"`
	SyncMatrix            map[string][]string   `json:"sync_matrix"`
	ImplementationGuard   ImplementationGuard   `json:"implementation_guard"`
}

type ModuleSpec struct {
	Readme string   `json:"readme"`
	Docs   []string `json:"docs"`
}

type ImplementationGuard struct {
	Ignore   PathRule            `json:"ignore"`
	Triggers map[string]PathRule `json:"triggers"`
}

type PathRule struct {
	Prefixes        []string `json:"prefixes"`
	Suffixes        []string `json:"suffixes"`
	Files           []string `json:"files"`
	ExcludePrefixes []string `json:"exclude_prefixes"`
	ExcludeSuffixes []string `json:"exclude_suffixes"`
	ExcludeFiles    []string `json:"exclude_files"`
}

func LoadManifest(repoRoot string) (Manifest, error) {
	path := filepath.Join(repoRoot, "docs", "governance", "DOC_MANIFEST.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func relativeToRepo(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func resolveRepoPath(repoRoot, rawPath string) string {
	return filepath.Clean(filepath.Join(repoRoot, filepath.FromSlash(rawPath)))
}
