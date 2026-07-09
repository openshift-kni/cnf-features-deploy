package fileutils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type schemaProperty struct {
	Type                       string                    `json:"type"`
	PatchStrategy              string                    `json:"x-kubernetes-patch-strategy"`
	PatchMergeKey              string                    `json:"x-kubernetes-patch-merge-key"`
	Properties                 map[string]schemaProperty `json:"properties"`
	KubernetesGroupVersionKind []struct {
		Group   string `json:"group"`
		Kind    string `json:"kind"`
		Version string `json:"version"`
	} `json:"x-kubernetes-group-version-kind"`
}

type openAPISchema struct {
	Definitions map[string]schemaProperty `json:"definitions"`
}

// mergeKeyInfo maps a dotted field path (e.g., "spec.filters") to the merge key name (e.g., "name")
type mergeKeyInfo struct {
	fieldPath string
	mergeKey  string
}

// ParseOpenAPISchema reads a schema.openapi file and returns a map of Kind → []mergeKeyInfo
func ParseOpenAPISchema(schemaPath string) (map[string][]mergeKeyInfo, error) {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", schemaPath, err)
	}

	var schema openAPISchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema file %s: %w", schemaPath, err)
	}

	result := make(map[string][]mergeKeyInfo)
	for _, def := range schema.Definitions {
		if len(def.KubernetesGroupVersionKind) == 0 {
			continue
		}
		kind := def.KubernetesGroupVersionKind[0].Kind
		var mergeKeys []mergeKeyInfo
		collectMergeKeys(def.Properties, "", &mergeKeys)
		if len(mergeKeys) > 0 {
			result[kind] = mergeKeys
		}
	}
	return result, nil
}

// collectMergeKeys recursively walks schema properties to find list fields with merge keys
func collectMergeKeys(props map[string]schemaProperty, prefix string, result *[]mergeKeyInfo) {
	for fieldName, prop := range props {
		path := fieldName
		if prefix != "" {
			path = prefix + "." + fieldName
		}
		if prop.Type == "array" && prop.PatchStrategy == "merge" && prop.PatchMergeKey != "" {
			*result = append(*result, mergeKeyInfo{fieldPath: path, mergeKey: prop.PatchMergeKey})
		}
		if prop.Properties != nil {
			collectMergeKeys(prop.Properties, path, result)
		}
	}
}

// InjectMergeKeys adds missing merge keys to patch list items by looking them up in the source CR.
// It uses the OpenAPI schema to know which list fields have merge keys.
func InjectMergeKeys(patches []map[string]interface{}, sourceCRPath string, mergeKeys []mergeKeyInfo) error {
	if len(patches) == 0 || len(mergeKeys) == 0 {
		return nil
	}

	sourceCRData, err := os.ReadFile(sourceCRPath)
	if err != nil {
		return fmt.Errorf("failed to read source CR %s: %w", sourceCRPath, err)
	}

	var sourceCR map[string]interface{}
	if err := yaml.Unmarshal(sourceCRData, &sourceCR); err != nil {
		return fmt.Errorf("failed to unmarshal source CR %s: %w", sourceCRPath, err)
	}

	for _, patch := range patches {
		for _, mk := range mergeKeys {
			injectMergeKeyAtPath(patch, sourceCR, mk.fieldPath, mk.mergeKey)
		}
	}
	return nil
}

// injectMergeKeyAtPath walks a dotted path in both patch and sourceCR, and injects merge keys
func injectMergeKeyAtPath(patch, sourceCR map[string]interface{}, fieldPath, mergeKey string) {
	parts := strings.Split(fieldPath, ".")
	patchNode := navigateToParent(patch, parts)
	sourceCRNode := navigateToParent(sourceCR, parts)
	if patchNode == nil || sourceCRNode == nil {
		return
	}

	lastField := parts[len(parts)-1]
	patchList, ok := toSliceOfMaps(patchNode[lastField])
	if !ok || len(patchList) == 0 {
		return
	}
	sourceCRList, ok := toSliceOfMaps(sourceCRNode[lastField])
	if !ok || len(sourceCRList) == 0 {
		return
	}

	// PGT merges list items by position, so patch item [i] corresponds to source CR item [i]
	for i, patchItem := range patchList {
		if _, hasMergeKey := patchItem[mergeKey]; hasMergeKey {
			continue
		}
		if i < len(sourceCRList) {
			if val, ok := sourceCRList[i][mergeKey]; ok {
				patchItem[mergeKey] = val
			}
		}
	}
}

func navigateToParent(node map[string]interface{}, parts []string) map[string]interface{} {
	current := node
	// Navigate to the parent of the last part
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part]
		if !ok {
			return nil
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil
		}
		current = nextMap
	}
	return current
}

func toSliceOfMaps(val interface{}) ([]map[string]interface{}, bool) {
	if val == nil {
		return nil, false
	}
	switch v := val.(type) {
	case []interface{}:
		var result []map[string]interface{}
		for _, item := range v {
			m, ok := item.(map[string]interface{})
			if !ok {
				return nil, false
			}
			result = append(result, m)
		}
		return result, true
	case []map[string]interface{}:
		return v, true
	}
	return nil, false
}
