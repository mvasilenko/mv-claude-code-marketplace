package merger

import (
	"encoding/json"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/managed"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
)

// Merge performs a deep merge of base into user settings.
// Returns merged result with deduplication of arrays.
//
// Strategy:
//   - Objects (maps): Deep recursive merge, all keys preserved
//   - Arrays: Concatenate then deduplicate, base items first
//   - Primitives or type conflicts: User value wins
func Merge(base, user any) any {
	if base == nil {
		return user
	}
	if user == nil {
		return base
	}

	switch baseVal := base.(type) {
	case map[string]any:
		if userMap, ok := user.(map[string]any); ok {
			return mergeObjects(baseVal, userMap)
		}
		return user

	case []any:
		if userArray, ok := user.([]any); ok {
			return mergeArrays(baseVal, userArray)
		}
		return user

	default:
		return user
	}
}

// mergeObjects recursively merges two maps.
// All keys from both maps are preserved.
// For shared keys, values are recursively merged.
func mergeObjects(base, user map[string]any) map[string]any {
	result := make(map[string]any)

	for key, baseValue := range base {
		if userValue, exists := user[key]; exists {
			if isAtomicKey(key) {
				result[key] = userValue
			} else if key == "enabledPlugins" {
				result[key] = mergeEnabledPlugins(baseValue, userValue)
			} else if key == "extraKnownMarketplaces" {
				result[key] = mergeExtraKnownMarketplaces(baseValue, userValue)
			} else {
				result[key] = Merge(baseValue, userValue)
			}
		} else {
			result[key] = baseValue
		}
	}

	for key, userValue := range user {
		if _, exists := base[key]; !exists {
			result[key] = userValue
		}
	}

	return result
}

// isAtomicKey checks if a key should be treated as atomic (not deep merged).
// Atomic keys have their entire value replaced by the user value, not merged.
func isAtomicKey(key string) bool {
	return key == "source"
}

// mergeEnabledPlugins handles special merging for the enabledPlugins map.
// User values always win (both true and false) to preserve user preferences.
// New plugins from base are added with their default values.
func mergeEnabledPlugins(base, user any) any {
	baseMap, baseOk := base.(map[string]any)
	userMap, userOk := user.(map[string]any)

	if !baseOk {
		return user
	}
	if !userOk {
		return base
	}

	result := make(map[string]any)

	// Step 1: User preferences win
	for key, userValue := range userMap {
		result[key] = userValue
	}

	// Step 2: Add NEW plugins from base that user hasn't set
	for key, baseValue := range baseMap {
		if _, exists := result[key]; !exists {
			result[key] = baseValue
		}
	}

	return result
}

// mergeExtraKnownMarketplaces handles special merging for the extraKnownMarketplaces map.
// Preserves custom marketplaces added by user.
// Default marketplace always comes from base settings.
func mergeExtraKnownMarketplaces(base, user any) any {
	baseMap, baseOk := base.(map[string]any)
	userMap, userOk := user.(map[string]any)

	if !baseOk {
		return user
	}
	if !userOk {
		return base
	}

	result := make(map[string]any)

	// Start with all marketplaces from base
	for key, baseValue := range baseMap {
		result[key] = baseValue
	}

	// Add custom marketplaces from user (skip default marketplace)
	for key, userValue := range userMap {
		if key == registry.DefaultMarketplaceID {
			continue // Default marketplace comes from base
		}
		if _, existsInBase := baseMap[key]; !existsInBase {
			result[key] = userValue
		}
	}

	return result
}

// mergeArrays concatenates base and user arrays, then deduplicates.
// Preserves order: base items first, then unique user items.
func mergeArrays(base, user []any) []any {
	combined := make([]any, 0, len(base)+len(user))
	combined = append(combined, base...)
	combined = append(combined, user...)
	return deduplicateArray(combined)
}

// deduplicateArray removes duplicate items from array.
// Keeps first occurrence of each unique value.
func deduplicateArray(arr []any) []any {
	seen := make(map[string]bool)
	result := make([]any, 0, len(arr))

	for _, item := range arr {
		key := itemToKey(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result
}

// itemToKey converts an any to a string key for deduplication.
func itemToKey(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	data, err := json.Marshal(v)
	if err != nil {
		return string(data)
	}
	return string(data)
}

// MergeWithRemoval performs a deep merge with support for removing stale entries.
// It tracks which entries came from base_settings.json and removes entries that
// were previously managed but are no longer in base.
//
// previousManaged: entries from metadata file (nil for first run)
// Returns (merged settings, updated managed.Entries for saving)
func MergeWithRemoval(base, user map[string]any, previousManaged managed.Entries) (map[string]any, managed.Entries) {
	currentBaseEntries := managed.ExtractFromSettings(base)

	mergedInterface := Merge(base, user)
	merged, ok := mergedInterface.(map[string]any)
	if !ok {
		merged = make(map[string]any)
	}

	managed.ApplyRemovalLogic(merged, previousManaged, currentBaseEntries)

	return merged, currentBaseEntries
}

// Equal compares two any values for deep equality.
// Uses JSON serialization for consistent comparison.
func Equal(a, b any) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}
