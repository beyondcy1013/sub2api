package service

import (
	"sort"
	"strings"
)

// AccountTestPickerModelIDs narrows a test-picker catalog to the models the
// account actually serves, using the same whitelist semantics as routing
// (IsModelSupported). 配置了 model_mapping（模型白名单/模型映射）的账号只保留
// 映射可命中的目录项，并补齐目录中缺失的具体白名单模型；透传账号和未配置
// 白名单的账号保持原目录，但仍排除路由必然不会服务的模型族。
func AccountTestPickerModelIDs(account *Account, catalogIDs []string) []string {
	if account == nil || account.IsOpenAIPassthroughEnabled() {
		return catalogIDs
	}

	kept := make([]string, 0, len(catalogIDs))
	seen := make(map[string]struct{}, len(catalogIDs))
	for _, id := range catalogIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		if !account.IsModelSupported(id) {
			continue
		}
		seen[id] = struct{}{}
		kept = append(kept, id)
	}

	mapping := account.GetModelMapping()
	if len(mapping) == 0 {
		return kept
	}

	missing := make([]string, 0, len(mapping))
	for key := range mapping {
		key = strings.TrimSpace(key)
		if key == "" || strings.ContainsAny(key, "*?") {
			// Wildcard patterns are routing rules, not concrete testable models.
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		missing = append(missing, key)
	}
	sort.Strings(missing)
	return append(kept, missing...)
}
