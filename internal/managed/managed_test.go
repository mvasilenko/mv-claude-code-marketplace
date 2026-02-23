package managed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntries_GetSet(t *testing.T) {
	e := make(Entries)

	// Get on empty returns nil
	assert.Nil(t, e.Get("missing"))

	// Set and Get
	e.Set("env", []string{"VAR1", "VAR2"})
	assert.Equal(t, []string{"VAR1", "VAR2"}, e.Get("env"))

	// Set nil deletes
	e.Set("env", nil)
	assert.Nil(t, e.Get("env"))

	// Get on nil Entries
	var nilEntries Entries
	assert.Nil(t, nilEntries.Get("anything"))
}

func TestExtractFromSettings(t *testing.T) {
	tests := []struct {
		expected Entries
		name     string
		settings map[string]any
	}{
		{
			name: "extracts map keys",
			settings: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp": true,
					"plugin-b@mp": false,
				},
			},
			expected: Entries{
				"enabledPlugins": {"plugin-a@mp", "plugin-b@mp"},
			},
		},
		{
			name: "extracts string array items",
			settings: map[string]any{
				"permissions": map[string]any{
					"allow": []any{"Bash(git:*)", "Read"},
				},
			},
			expected: Entries{
				"permissions.allow": {"Bash(git:*)", "Read"},
			},
		},
		{
			name: "extracts empty array",
			settings: map[string]any{
				"permissions": map[string]any{
					"deny": []any{},
				},
			},
			expected: Entries{
				"permissions.deny": {},
			},
		},
		{
			name:     "empty settings",
			settings: map[string]any{},
			expected: Entries{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFromSettings(tt.settings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyRemovalLogic_NormalRun(t *testing.T) {
	merged := map[string]any{
		"env": map[string]any{
			"VAR1": "val1",
			"VAR2": "val2",
			"VAR3": "val3",
		},
	}

	previous := Entries{
		"env": {"VAR1", "VAR2", "VAR3"},
	}
	current := Entries{
		"env": {"VAR1", "VAR2"},
	}

	ApplyRemovalLogic(merged, previous, current)

	envMap := merged["env"].(map[string]any)
	assert.Contains(t, envMap, "VAR1")
	assert.Contains(t, envMap, "VAR2")
	assert.NotContains(t, envMap, "VAR3")
}

func TestApplyRemovalLogic_FirstRun(t *testing.T) {
	merged := map[string]any{
		"env": map[string]any{
			"VAR1":   "val1",
			"CUSTOM": "user-added",
			"VAR2":   "val2",
		},
	}

	current := Entries{
		"env": {"VAR1", "VAR2"},
	}

	// nil previous = first run
	ApplyRemovalLogic(merged, nil, current)

	envMap := merged["env"].(map[string]any)
	assert.Contains(t, envMap, "VAR1")
	assert.Contains(t, envMap, "VAR2")
	// CUSTOM is removed on first run (retroactive cleanup)
	assert.NotContains(t, envMap, "CUSTOM")
}

func TestApplyRemovalLogic_ArrayRemoval(t *testing.T) {
	merged := map[string]any{
		"permissions": map[string]any{
			"allow": []any{"Bash(git:*)", "Read", "Write"},
		},
	}

	previous := Entries{
		"permissions.allow": {"Bash(git:*)", "Read", "Write"},
	}
	current := Entries{
		"permissions.allow": {"Bash(git:*)", "Read"},
	}

	ApplyRemovalLogic(merged, previous, current)

	allow := merged["permissions"].(map[string]any)["allow"].([]any)
	require.Len(t, allow, 2)
	assert.Equal(t, "Bash(git:*)", allow[0])
	assert.Equal(t, "Read", allow[1])
}

func TestApplyRemovalLogic_NoChanges(t *testing.T) {
	merged := map[string]any{
		"env": map[string]any{
			"VAR1": "val1",
		},
	}

	previous := Entries{"env": {"VAR1"}}
	current := Entries{"env": {"VAR1"}}

	ApplyRemovalLogic(merged, previous, current)

	envMap := merged["env"].(map[string]any)
	assert.Contains(t, envMap, "VAR1")
}
