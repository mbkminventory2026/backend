package usecase

import (
	"reflect"
	"sort"

	"permatatex-inventory/internal/model"
)

func buildChangedFieldsFromSnapshots(before, after map[string]any) []model.AuditLogChangedField {
	if len(before) == 0 && len(after) == 0 {
		return []model.AuditLogChangedField{}
	}

	keySet := make(map[string]struct{}, len(before)+len(after))
	for key := range before {
		keySet[key] = struct{}{}
	}
	for key := range after {
		keySet[key] = struct{}{}
	}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	changed := make([]model.AuditLogChangedField, 0, len(keys))
	for _, key := range keys {
		beforeValue, beforeExists := before[key]
		afterValue, afterExists := after[key]

		if !beforeExists {
			beforeValue = nil
		}
		if !afterExists {
			afterValue = nil
		}

		if reflect.DeepEqual(beforeValue, afterValue) {
			continue
		}

		changed = append(changed, model.AuditLogChangedField{
			Field:  key,
			Before: beforeValue,
			After:  afterValue,
		})
	}

	if changed == nil {
		return []model.AuditLogChangedField{}
	}

	return changed
}
