package quality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	requiredFamilyKeys     = []string{"family_id", "kind", "directory", "schema"}
	requiredAssetEntryKeys = []string{"asset_id", "family_id", "kind", "revision", "path"}
	requiredAssetFileKeys  = []string{
		"schema_version",
		"asset_id",
		"family_id",
		"kind",
		"revision",
		"status",
		"summary",
		"runtime_modes",
		"test_refs",
		"source_trace",
	}
	requiredListFields = []string{"runtime_modes", "test_refs", "source_trace"}
)

type requirementRegistryIndex struct {
	SchemaVersion int                       `json:"schema_version"`
	Families      []requirementRegistryItem `json:"families"`
	Assets        []requirementAssetEntry   `json:"assets"`
}

type requirementRegistryItem struct {
	FamilyID  string `json:"family_id"`
	Kind      string `json:"kind"`
	Directory string `json:"directory"`
	Schema    string `json:"schema"`
}

type requirementAssetEntry struct {
	AssetID  string `json:"asset_id"`
	FamilyID string `json:"family_id"`
	Kind     string `json:"kind"`
	Revision string `json:"revision"`
	Path     string `json:"path"`
}

type resolvedFamily struct {
	Spec          requirementRegistryItem
	DirectoryPath string
	SchemaPath    string
}

func ValidateRequirementAssets(repoRoot string) []string {
	registryPath := filepath.Join(repoRoot, "requirements", "registry", "index.json")
	if _, err := os.Stat(registryPath); err != nil {
		return []string{"缺少 requirement registry：" + relativeToRepo(repoRoot, registryPath)}
	}

	registry, err := loadRequirementRegistryIndex(registryPath)
	if err != nil {
		return []string{"读取 requirement registry 失败：" + err.Error()}
	}

	errors := []string{}
	families := familyDirectories(repoRoot, registry)

	if registry.SchemaVersion != 1 {
		errors = append(errors, "requirement registry schema_version 必须为 1")
	}

	familyIDs := map[string]struct{}{}
	familyKinds := map[string]struct{}{}
	for _, family := range registry.Families {
		familyMap := map[string]string{
			"family_id": family.FamilyID,
			"kind":      family.Kind,
			"directory": family.Directory,
			"schema":    family.Schema,
		}
		for _, key := range requiredFamilyKeys {
			if stringsTrim(familyMap[key]) == "" {
				errors = append(errors, "family 缺少字段 `"+key+"`："+marshalLooseJSON(familyMap))
			}
		}
		if family.FamilyID != "" {
			if _, ok := familyIDs[family.FamilyID]; ok {
				errors = append(errors, "family_id 重复："+family.FamilyID)
			}
			familyIDs[family.FamilyID] = struct{}{}
		}
		if family.Kind != "" {
			if _, ok := familyKinds[family.Kind]; ok {
				errors = append(errors, "family kind 重复："+family.Kind)
			}
			familyKinds[family.Kind] = struct{}{}
		}
		resolved := families[family.FamilyID]
		if _, err := os.Stat(resolved.DirectoryPath); err != nil {
			errors = append(errors, "family 目录不存在："+relativeToRepo(repoRoot, resolved.DirectoryPath))
		}
		if _, err := os.Stat(resolved.SchemaPath); err != nil {
			errors = append(errors, "family schema 不存在："+relativeToRepo(repoRoot, resolved.SchemaPath))
		}
	}

	registeredAssetIDs := map[string]struct{}{}
	registeredAssetPaths := map[string]struct{}{}
	for _, asset := range registry.Assets {
		assetMap := map[string]string{
			"asset_id":  asset.AssetID,
			"family_id": asset.FamilyID,
			"kind":      asset.Kind,
			"revision":  asset.Revision,
			"path":      asset.Path,
		}
		for _, key := range requiredAssetEntryKeys {
			if stringsTrim(assetMap[key]) == "" {
				errors = append(errors, "asset entry 缺少字段 `"+key+"`："+marshalLooseJSON(assetMap))
			}
		}

		if asset.AssetID != "" {
			if _, ok := registeredAssetIDs[asset.AssetID]; ok {
				errors = append(errors, "asset_id 重复："+asset.AssetID)
			}
			registeredAssetIDs[asset.AssetID] = struct{}{}
		}

		family, ok := families[asset.FamilyID]
		if !ok {
			errors = append(errors, "asset entry 引用了未知 family_id："+asset.FamilyID)
			continue
		}
		if asset.Kind != family.Spec.Kind {
			errors = append(errors, "asset entry kind 与 family 不一致："+asset.AssetID+" -> "+asset.Kind+" != "+family.Spec.Kind)
		}

		assetPath := resolveRepoPath(repoRoot, asset.Path)
		if _, ok := registeredAssetPaths[assetPath]; ok {
			errors = append(errors, "asset path 重复："+relativeToRepo(repoRoot, assetPath))
		} else {
			registeredAssetPaths[assetPath] = struct{}{}
		}
		if _, err := os.Stat(assetPath); err != nil {
			errors = append(errors, "asset 文件不存在："+relativeToRepo(repoRoot, assetPath))
			continue
		}
		if filepath.Clean(filepath.Dir(assetPath)) != filepath.Clean(family.DirectoryPath) {
			errors = append(errors, "asset 文件不在对应 family 目录下："+relativeToRepo(repoRoot, assetPath))
		}
		assetFile, err := loadLooseJSON(assetPath)
		if err != nil {
			errors = append(errors, "读取 asset 文件失败："+relativeToRepo(repoRoot, assetPath))
			continue
		}
		for _, key := range requiredAssetFileKeys {
			if _, ok := assetFile[key]; !ok {
				errors = append(errors, "asset 文件缺少字段 `"+key+"`："+relativeToRepo(repoRoot, assetPath))
			}
		}
		for _, key := range []string{"asset_id", "family_id", "kind", "revision"} {
			if value, ok := assetFile[key]; !ok || stringsTrim(stringValue(value)) != assetMap[key] {
				errors = append(errors, "asset 文件字段与 registry 不一致："+relativeToRepo(repoRoot, assetPath)+" -> "+key)
			}
		}
		for _, key := range requiredListFields {
			if value, ok := assetFile[key]; ok {
				if _, ok := value.([]interface{}); !ok {
					errors = append(errors, "asset 文件字段必须为 list："+relativeToRepo(repoRoot, assetPath)+" -> "+key)
				}
			}
		}
	}

	for _, family := range families {
		matches, _ := filepath.Glob(filepath.Join(family.DirectoryPath, "*.json"))
		sort.Strings(matches)
		for _, assetPath := range matches {
			if _, ok := registeredAssetPaths[assetPath]; !ok {
				errors = append(errors, "family 目录中存在未登记 asset 文件："+relativeToRepo(repoRoot, assetPath))
			}
		}
	}

	for _, asset := range registry.Assets {
		assetPath := resolveRepoPath(repoRoot, asset.Path)
		if _, err := os.Stat(assetPath); err != nil {
			continue
		}
		assetFile, err := loadLooseJSON(assetPath)
		if err != nil {
			continue
		}
		for _, key := range []string{"linked_assets", "derived_assets"} {
			for _, referencedAsset := range stringListValue(assetFile[key]) {
				if _, ok := registeredAssetIDs[referencedAsset]; !ok {
					errors = append(errors, "asset 引用了未登记的关联资产："+relativeToRepo(repoRoot, assetPath)+" -> "+referencedAsset)
				}
			}
		}
	}

	return errors
}

func loadRequirementRegistryIndex(path string) (requirementRegistryIndex, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return requirementRegistryIndex{}, err
	}
	var registry requirementRegistryIndex
	if err := json.Unmarshal(payload, &registry); err != nil {
		return requirementRegistryIndex{}, err
	}
	return registry, nil
}

func familyDirectories(repoRoot string, registry requirementRegistryIndex) map[string]resolvedFamily {
	families := map[string]resolvedFamily{}
	for _, family := range registry.Families {
		families[family.FamilyID] = resolvedFamily{
			Spec:          family,
			DirectoryPath: resolveRepoPath(repoRoot, family.Directory),
			SchemaPath:    resolveRepoPath(repoRoot, family.Schema),
		}
	}
	return families
}

func loadLooseJSON(path string) (map[string]interface{}, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func marshalLooseJSON(v interface{}) string {
	payload, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return text
}

func stringListValue(value interface{}) []string {
	list, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(list))
	for _, item := range list {
		text, ok := item.(string)
		if ok && stringsTrim(text) != "" {
			result = append(result, text)
		}
	}
	return result
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}
