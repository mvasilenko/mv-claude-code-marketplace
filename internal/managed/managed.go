// Package managed provides dynamic tracking of settings entries from base_settings.json.
// It auto-discovers trackable fields (map keys and string arrays) without requiring
// hardcoded field definitions.
package managed

import (
	"sort"
	"strings"
)

// Entries stores tracked field values keyed by JSON path.
// For maps, it stores the keys. For string arrays, it stores the items.
// Example: {"env": ["VAR1", "VAR2"], "permissions.allow": ["Bash(git:*)"]}
type Entries map[string][]string

// Get returns the tracked entries for a field path.
func (e Entries) Get(path string) []string {
	if e == nil {
		return nil
	}
	return e[path]
}

// Set sets the tracked entries for a field path.
// Note: empty slices are stored (not deleted) because we need to track
// empty arrays (e.g., permissions.deny = []).
func (e Entries) Set(path string, values []string) {
	if values != nil {
		e[path] = values
	} else {
		delete(e, path)
	}
}

// ExtractFromSettings auto-discovers all trackable entries from settings.
// It traverses the settings map and extracts:
// - Keys from maps that contain primitive values (strings, bools, numbers, or objects)
// - Items from arrays that contain strings
func ExtractFromSettings(settings map[string]any) Entries {
	entries := make(Entries)
	extractFromMap(settings, "", entries)
	return entries
}

// extractFromMap recursively extracts trackable entries from a map.
func extractFromMap(m map[string]any, prefix string, entries Entries) {
	for key, value := range m {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			if isTrackableMap(v) {
				keys := extractMapKeys(v)
				entries.Set(path, keys)
			} else {
				extractFromMap(v, path, entries)
			}

		case []any:
			if items := extractStringArrayItems(v); items != nil {
				entries.Set(path, items)
			}
		}
	}
}

// isTrackableMap returns true if the map's values are primitives or objects
// (i.e., it's a map we should track keys for, not traverse into).
func isTrackableMap(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}

	for _, v := range m {
		switch v.(type) {
		case string, bool, float64, int, nil:
			return true
		case map[string]any:
			return true
		case []any:
			return false
		}
	}
	return true
}

// extractMapKeys extracts all keys from a map.
func extractMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// extractStringArrayItems extracts string items from an array.
// Returns nil if the array doesn't contain strings.
func extractStringArrayItems(arr []any) []string {
	if len(arr) == 0 {
		return []string{}
	}

	items := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			items = append(items, s)
		} else {
			return nil
		}
	}
	return items
}

// ApplyRemovalLogic removes stale entries from merged settings.
// - Normal run: Remove entries that were in previous metadata but no longer in current base
// - First run (previous=nil): Retroactive cleanup - remove entries not in current base
func ApplyRemovalLogic(merged map[string]any, previous, current Entries) {
	isFirstRun := previous == nil

	for path := range current {
		if isFirstRun {
			applyFirstRunRemoval(merged, path, current.Get(path))
		} else {
			applyRemovalForPath(merged, path, previous.Get(path), current.Get(path))
		}
	}

	if !isFirstRun {
		for path := range previous {
			if _, exists := current[path]; !exists {
				applyRemovalForPath(merged, path, previous.Get(path), nil)
			}
		}
	}
}

// applyFirstRunRemoval removes entries not in current base (retroactive cleanup).
func applyFirstRunRemoval(merged map[string]any, path string, currentEntries []string) {
	value := getNestedValue(merged, path)
	if value == nil {
		return
	}

	currentSet := sliceToSet(currentEntries)

	switch v := value.(type) {
	case map[string]any:
		for key := range v {
			if !currentSet[key] {
				delete(v, key)
			}
		}

	case []any:
		filtered := make([]any, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && currentSet[s] {
				filtered = append(filtered, item)
			} else if !ok {
				filtered = append(filtered, item)
			}
		}
		setNestedValue(merged, path, filtered)
	}
}

// applyRemovalForPath applies removal logic for a single field path.
func applyRemovalForPath(merged map[string]any, path string, previousEntries, currentEntries []string) {
	value := getNestedValue(merged, path)
	if value == nil {
		return
	}

	previousSet := sliceToSet(previousEntries)
	currentSet := sliceToSet(currentEntries)

	toRemove := make(map[string]bool)
	for entry := range previousSet {
		if !currentSet[entry] {
			toRemove[entry] = true
		}
	}

	if len(toRemove) == 0 {
		return
	}

	switch v := value.(type) {
	case map[string]any:
		for key := range toRemove {
			delete(v, key)
		}

	case []any:
		filtered := make([]any, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && toRemove[s] {
				continue
			}
			filtered = append(filtered, item)
		}
		setNestedValue(merged, path, filtered)
	}
}

// getNestedValue navigates a dot-notation path and returns the value.
func getNestedValue(m map[string]any, path string) any {
	parts := strings.Split(path, ".")
	current := any(m)

	for _, part := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = cm[part]
		if current == nil {
			return nil
		}
	}

	return current
}

// setNestedValue sets a value at a dot-notation path.
func setNestedValue(m map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := m

	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]].(map[string]any)
		if !ok {
			return
		}
		current = next
	}

	current[parts[len(parts)-1]] = value
}

// sliceToSet converts a string slice to a set (map[string]bool).
func sliceToSet(slice []string) map[string]bool {
	set := make(map[string]bool, len(slice))
	for _, s := range slice {
		set[s] = true
	}
	return set
}
