package quality

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	requiredDocFiles    = []string{
		"README.md",
		"docs/INDEX.md",
		"docs/ARCHITECTURE.md",
		"docs/FEATURE_INHERITANCE.md",
		"docs/SYSTEM_TARGET_ARCHITECTURE.md",
		"docs/ROADMAP.md",
		"docs/SUBSYSTEM_BOUNDARIES.md",
		"docs/INTERFACES.md",
		"docs/STORAGE_SCHEMA.md",
		"docs/TESTING.md",
		"docs/DESIGN_SYNC_WORKFLOW.md",
		"docs/OPERATIONS.md",
		"docs/REQUIREMENT_SYNC_CHECKLIST.md",
		"docs/REQUIREMENT_CHANGELOG.md",
		"docs/architecture/README.md",
		"docs/runtime/README.md",
		"docs/governance/README.md",
		"docs/governance/DOC_MANIFEST.json",
		"docs/governance/DOC_SYSTEM.md",
		"docs/governance/INCOMING_WORKFLOW.md",
		"docs/operations/README.md",
		"docs/operations/NODE_RUNTIME.md",
		"docs/operations/ISSUE_BRIDGE.md",
	}
	moduleDirectories = []string{"architecture", "runtime", "governance", "operations"}
)

func ValidateDocsConsistency(repoRoot string) ([]string, error) {
	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		return nil, err
	}
	errors := []string{}
	errors = append(errors, validateRequiredFiles(repoRoot)...)
	errors = append(errors, ValidateManifestSemantics(manifest)...)
	errors = append(errors, ValidateManifestPaths(repoRoot, manifest)...)
	errors = append(errors, ValidateTopLevelDocs(repoRoot, manifest)...)
	errors = append(errors, ValidateModuleDocs(repoRoot, manifest)...)
	errors = append(errors, validateLinks(repoRoot)...)
	return errors, nil
}

func ValidateManifestSemantics(manifest Manifest) []string {
	errors := []string{}
	for moduleName, module := range manifest.Modules {
		if strings.TrimSpace(module.Readme) == "" {
			errors = append(errors, "模块缺少 readme："+moduleName)
		}
		if len(module.Docs) == 0 {
			errors = append(errors, "模块缺少 docs 列表："+moduleName)
		}
	}
	if len(manifest.TopicRoutes) == 0 {
		errors = append(errors, "manifest 缺少 topic_routes")
	}
	if len(manifest.SyncMatrix) == 0 {
		errors = append(errors, "manifest 缺少 sync_matrix")
	}
	if len(manifest.ImplementationGuard.Triggers) == 0 {
		errors = append(errors, "manifest 缺少 implementation_guard")
		return errors
	}
	if len(manifest.ImplementationGuard.Ignore.Prefixes) == 0 &&
		len(manifest.ImplementationGuard.Ignore.Suffixes) == 0 &&
		len(manifest.ImplementationGuard.Ignore.Files) == 0 {
		errors = append(errors, "implementation_guard 缺少 ignore 规则")
	}
	for category, rule := range manifest.ImplementationGuard.Triggers {
		if _, ok := manifest.SyncMatrix[category]; !ok {
			errors = append(errors, "implementation_guard 引用了未定义的 sync_matrix 项："+category)
		}
		if len(rule.Prefixes) == 0 && len(rule.Suffixes) == 0 && len(rule.Files) == 0 {
			errors = append(errors, "implementation_guard 触发规则为空："+category)
		}
	}
	return errors
}

func ValidateManifestPaths(repoRoot string, manifest Manifest) []string {
	errors := []string{}
	for _, path := range manifestPaths(repoRoot, manifest) {
		if _, err := os.Stat(path); err != nil {
			errors = append(errors, "manifest 引用了不存在的路径："+relativeToRepo(repoRoot, path))
		}
	}
	return errors
}

func ValidateTopLevelDocs(repoRoot string, manifest Manifest) []string {
	errors := []string{}
	actual := map[string]struct{}{}
	matches, _ := filepath.Glob(filepath.Join(repoRoot, "docs", "*.md"))
	for _, path := range matches {
		actual[filepath.Clean(path)] = struct{}{}
	}

	declared := map[string]struct{}{}
	for _, rawPath := range manifest.CanonicalTopLevelDocs {
		declared[resolveRepoPath(repoRoot, rawPath)] = struct{}{}
	}

	for _, path := range sortedDifference(actual, declared) {
		errors = append(errors, "顶层 canonical 文档未登记到 manifest："+relativeToRepo(repoRoot, path))
	}
	for _, path := range sortedDifference(declared, actual) {
		errors = append(errors, "manifest 顶层文档多余或路径错误："+relativeToRepo(repoRoot, path))
	}
	return errors
}

func ValidateModuleDocs(repoRoot string, manifest Manifest) []string {
	errors := []string{}
	declared := map[string]struct{}{}
	docsRoot := filepath.Join(repoRoot, "docs")
	for _, module := range manifest.Modules {
		if module.Readme != "" {
			path := resolveRepoPath(repoRoot, module.Readme)
			if filepath.Dir(path) != docsRoot {
				declared[path] = struct{}{}
			}
		}
		for _, rawPath := range module.Docs {
			path := resolveRepoPath(repoRoot, rawPath)
			if filepath.Dir(path) != docsRoot {
				declared[path] = struct{}{}
			}
		}
	}

	actual := map[string]struct{}{}
	for _, moduleDir := range moduleDirectories {
		matches, _ := filepath.Glob(filepath.Join(repoRoot, "docs", moduleDir, "*.md"))
		for _, path := range matches {
			actual[filepath.Clean(path)] = struct{}{}
		}
	}

	for _, path := range sortedDifference(actual, declared) {
		errors = append(errors, "模块文档未登记到 manifest："+relativeToRepo(repoRoot, path))
	}
	for _, path := range sortedDifference(declared, actual) {
		errors = append(errors, "manifest 模块文档多余或路径错误："+relativeToRepo(repoRoot, path))
	}
	return errors
}

func validateRequiredFiles(repoRoot string) []string {
	errors := []string{}
	for _, rawPath := range requiredDocFiles {
		path := resolveRepoPath(repoRoot, rawPath)
		if _, err := os.Stat(path); err != nil {
			errors = append(errors, "缺少必需文档："+rawPath)
		}
	}
	return errors
}

func validateLinks(repoRoot string) []string {
	errors := []string{}
	for _, path := range markdownFiles(repoRoot) {
		payload, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, "读取文档失败："+relativeToRepo(repoRoot, path))
			continue
		}
		matches := markdownLinkPattern.FindAllStringSubmatch(string(payload), -1)
		for _, match := range matches {
			target := strings.TrimSpace(match[1])
			if target == "" || isExternalLink(target) {
				continue
			}
			resolved := normalizeLinkTarget(path, target)
			if _, err := os.Stat(resolved); err != nil {
				errors = append(errors, "无效链接："+relativeToRepo(repoRoot, path)+" -> "+match[1])
			}
		}
	}
	return errors
}

func markdownFiles(repoRoot string) []string {
	files := []string{filepath.Join(repoRoot, "README.md")}
	_ = filepath.WalkDir(filepath.Join(repoRoot, "docs"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files
}

func isExternalLink(target string) bool {
	for _, prefix := range []string{"http://", "https://", "mailto:", "#", "app://"} {
		if strings.HasPrefix(target, prefix) {
			return true
		}
	}
	return false
}

func normalizeLinkTarget(baseFile, rawTarget string) string {
	target := strings.SplitN(rawTarget, "#", 2)[0]
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "<") && strings.HasSuffix(target, ">") {
		target = strings.TrimSuffix(strings.TrimPrefix(target, "<"), ">")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(baseFile), filepath.FromSlash(target)))
}

func manifestPaths(repoRoot string, manifest Manifest) []string {
	paths := map[string]struct{}{}
	for _, rawPath := range manifest.CanonicalTopLevelDocs {
		paths[resolveRepoPath(repoRoot, rawPath)] = struct{}{}
	}
	for _, module := range manifest.Modules {
		if module.Readme != "" {
			paths[resolveRepoPath(repoRoot, module.Readme)] = struct{}{}
		}
		for _, rawPath := range module.Docs {
			paths[resolveRepoPath(repoRoot, rawPath)] = struct{}{}
		}
	}
	for _, docList := range manifest.TopicRoutes {
		for _, rawPath := range docList {
			paths[resolveRepoPath(repoRoot, rawPath)] = struct{}{}
		}
	}
	for _, docList := range manifest.SyncMatrix {
		for _, rawPath := range docList {
			paths[resolveRepoPath(repoRoot, rawPath)] = struct{}{}
		}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func sortedDifference(left, right map[string]struct{}) []string {
	missing := []string{}
	for path := range left {
		if _, ok := right[path]; !ok {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	return missing
}
