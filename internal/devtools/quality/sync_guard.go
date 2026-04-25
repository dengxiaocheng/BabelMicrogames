package quality

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type GitDiffOptions struct {
	Staged        bool
	Against       string
	Base          string
	Head          string
	MergeBaseWith string
}

func NormalizeRepoPaths(repoRoot string, paths []string) []string {
	normalized := map[string]struct{}{}
	for _, rawPath := range paths {
		candidate := strings.TrimSpace(rawPath)
		if candidate == "" {
			continue
		}
		if filepath.IsAbs(candidate) {
			rel, err := filepath.Rel(repoRoot, filepath.Clean(candidate))
			if err != nil {
				continue
			}
			candidate = rel
		}
		candidate = filepath.ToSlash(filepath.Clean(candidate))
		candidate = strings.TrimPrefix(candidate, "./")
		if candidate == "." || candidate == "" {
			continue
		}
		normalized[candidate] = struct{}{}
	}
	result := make([]string, 0, len(normalized))
	for candidate := range normalized {
		result = append(result, candidate)
	}
	sort.Strings(result)
	return result
}

func ChangedPathsFromGit(repoRoot string, options GitDiffOptions) ([]string, error) {
	if options.Head == "" {
		options.Head = "HEAD"
	}
	if options.Against == "" {
		options.Against = "HEAD"
	}
	if options.Staged {
		tracked, err := gitLines(repoRoot, "diff", "--name-only", "--cached", "--diff-filter=ACMRD")
		if err != nil {
			return nil, err
		}
		return NormalizeRepoPaths(repoRoot, tracked), nil
	}
	base := options.Base
	if options.MergeBaseWith != "" {
		mergeBase, err := gitLine(repoRoot, "merge-base", options.Head, options.MergeBaseWith)
		if err != nil {
			return nil, err
		}
		base = mergeBase
	}
	if base != "" {
		tracked, err := gitLines(repoRoot, "diff", "--name-only", "--diff-filter=ACMRD", base, options.Head, "--")
		if err != nil {
			return nil, err
		}
		return NormalizeRepoPaths(repoRoot, tracked), nil
	}
	tracked, err := gitLines(repoRoot, "diff", "--name-only", "--diff-filter=ACMRD", options.Against, "--")
	if err != nil {
		return nil, err
	}
	untracked, err := gitLines(repoRoot, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return NormalizeRepoPaths(repoRoot, append(tracked, untracked...)), nil
}

func MatchesRule(path string, rule PathRule) bool {
	include := false
	for _, prefix := range rule.Prefixes {
		if strings.HasPrefix(path, prefix) {
			include = true
			break
		}
	}
	if !include {
		for _, suffix := range rule.Suffixes {
			if strings.HasSuffix(path, suffix) {
				include = true
				break
			}
		}
	}
	if !include {
		for _, exact := range rule.Files {
			if path == exact {
				include = true
				break
			}
		}
	}
	if !include {
		return false
	}
	for _, prefix := range rule.ExcludePrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	for _, suffix := range rule.ExcludeSuffixes {
		if strings.HasSuffix(path, suffix) {
			return false
		}
	}
	for _, exact := range rule.ExcludeFiles {
		if path == exact {
			return false
		}
	}
	return true
}

func IgnoredPaths(changedPaths []string, manifest Manifest) []string {
	ignored := []string{}
	for _, path := range changedPaths {
		if MatchesRule(path, manifest.ImplementationGuard.Ignore) {
			ignored = append(ignored, path)
		}
	}
	return ignored
}

func RelevantPaths(changedPaths []string, manifest Manifest) []string {
	ignored := map[string]struct{}{}
	for _, path := range IgnoredPaths(changedPaths, manifest) {
		ignored[path] = struct{}{}
	}
	relevant := []string{}
	for _, path := range changedPaths {
		if _, ok := ignored[path]; !ok {
			relevant = append(relevant, path)
		}
	}
	return relevant
}

func TriggeredCategories(changedPaths []string, manifest Manifest) map[string][]string {
	relevant := RelevantPaths(changedPaths, manifest)
	triggered := map[string][]string{}
	for category, rule := range manifest.ImplementationGuard.Triggers {
		matched := []string{}
		for _, path := range relevant {
			if MatchesRule(path, rule) {
				matched = append(matched, path)
			}
		}
		if len(matched) > 0 {
			triggered[category] = matched
		}
	}
	return triggered
}

func ValidateSyncGuard(repoRoot string, changedPaths []string, manifest Manifest) []string {
	errors := []string{}
	changed := map[string]struct{}{}
	for _, path := range NormalizeRepoPaths(repoRoot, changedPaths) {
		changed[path] = struct{}{}
	}
	triggered := TriggeredCategories(keys(changed), manifest)
	categories := keysOfMap(triggered)
	sort.Strings(categories)
	for _, category := range categories {
		requiredDocs := manifest.SyncMatrix[category]
		satisfied := false
		for _, doc := range requiredDocs {
			if _, ok := changed[doc]; ok {
				satisfied = true
				break
			}
		}
		if satisfied {
			continue
		}
		errors = append(errors,
			"同步守卫失败：检测到 `"+category+"`，触发路径："+strings.Join(triggered[category], "、")+"；"+
				"但当前变更中没有同步这些文档之一："+strings.Join(requiredDocs, "、"),
		)
	}
	return errors
}

func gitLines(repoRoot string, args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	lines := []string{}
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func gitLine(repoRoot string, args ...string) (string, error) {
	lines, err := gitLines(repoRoot, args...)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("git command returned no output: git %s", strings.Join(args, " "))
	}
	return lines[0], nil
}

func keys(set map[string]struct{}) []string {
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func keysOfMap[V any](m map[string]V) []string {
	result := make([]string, 0, len(m))
	for key := range m {
		result = append(result, key)
	}
	return result
}
