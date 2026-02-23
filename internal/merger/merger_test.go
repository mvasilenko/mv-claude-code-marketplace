package merger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/managed"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		base     any
		expected any
		name     string
		user     any
	}{
		{
			name:     "nil base returns user",
			base:     nil,
			user:     map[string]any{"key": "val"},
			expected: map[string]any{"key": "val"},
		},
		{
			name:     "nil user returns base",
			base:     map[string]any{"key": "val"},
			user:     nil,
			expected: map[string]any{"key": "val"},
		},
		{
			name:     "both nil returns nil",
			base:     nil,
			user:     nil,
			expected: nil,
		},
		{
			name:     "primitive user wins",
			base:     "base",
			user:     "user",
			expected: "user",
		},
		{
			name: "deep merge objects",
			base: map[string]any{"a": "1", "b": "2"},
			user: map[string]any{"b": "3", "c": "4"},
			expected: map[string]any{
				"a": "1",
				"b": "3",
				"c": "4",
			},
		},
		{
			name:     "arrays concat and dedup",
			base:     []any{"a", "b"},
			user:     []any{"b", "c"},
			expected: []any{"a", "b", "c"},
		},
		{
			name:     "type mismatch user wins",
			base:     map[string]any{"key": "val"},
			user:     "override",
			expected: "override",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Merge(tt.base, tt.user)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeObjects_AtomicKeys(t *testing.T) {
	base := map[string]any{
		"source": map[string]any{
			"source": "github",
			"repo":   "org/repo",
		},
	}
	user := map[string]any{
		"source": map[string]any{
			"source": "directory",
			"path":   "/local/path",
		},
	}

	result := Merge(base, user)
	resultMap := result.(map[string]any)
	source := resultMap["source"].(map[string]any)

	// User's atomic key replaces entirely
	assert.Equal(t, "directory", source["source"])
	assert.Equal(t, "/local/path", source["path"])
	assert.Nil(t, source["repo"])
}

func TestMergeEnabledPlugins(t *testing.T) {
	tests := []struct {
		base     map[string]any
		expected map[string]any
		name     string
		user     map[string]any
	}{
		{
			name: "user disabled stays disabled",
			base: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp": true,
					"plugin-b@mp": true,
				},
			},
			user: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp": true,
					"plugin-b@mp": false,
				},
			},
			expected: map[string]any{
				"plugin-a@mp": true,
				"plugin-b@mp": false,
			},
		},
		{
			name: "new plugin from base added",
			base: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp":   true,
					"plugin-new@mp": true,
				},
			},
			user: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp": true,
				},
			},
			expected: map[string]any{
				"plugin-a@mp":   true,
				"plugin-new@mp": true,
			},
		},
		{
			name: "custom marketplace plugin preserved",
			base: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp": true,
				},
			},
			user: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp":     true,
					"custom@other-mp": true,
				},
			},
			expected: map[string]any{
				"plugin-a@mp":     true,
				"custom@other-mp": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Merge(tt.base, tt.user)
			resultMap := result.(map[string]any)
			assert.Equal(t, tt.expected, resultMap["enabledPlugins"])
		})
	}
}

func TestMergeExtraKnownMarketplaces(t *testing.T) {
	defaultID := registry.DefaultMarketplaceID

	base := map[string]any{
		"extraKnownMarketplaces": map[string]any{
			defaultID: map[string]any{
				"source": map[string]any{
					"source": "github",
					"repo":   "org/default-mp",
				},
			},
		},
	}
	user := map[string]any{
		"extraKnownMarketplaces": map[string]any{
			defaultID: map[string]any{
				"source": map[string]any{
					"source": "github",
					"repo":   "old/default-mp",
				},
			},
			"custom-mp": map[string]any{
				"source": map[string]any{
					"source": "github",
					"repo":   "user/custom-mp",
				},
			},
		},
	}

	result := Merge(base, user)
	resultMap := result.(map[string]any)
	marketplaces := resultMap["extraKnownMarketplaces"].(map[string]any)

	// Default comes from base
	defaultMP := marketplaces[defaultID].(map[string]any)
	defaultSource := defaultMP["source"].(map[string]any)
	assert.Equal(t, "org/default-mp", defaultSource["repo"])

	// Custom is preserved from user
	customMP := marketplaces["custom-mp"].(map[string]any)
	customSource := customMP["source"].(map[string]any)
	assert.Equal(t, "user/custom-mp", customSource["repo"])
}

func TestMergeWithRemoval(t *testing.T) {
	base := map[string]any{
		"env": map[string]any{
			"VAR1": "val1",
			"VAR2": "val2",
		},
	}
	user := map[string]any{
		"env": map[string]any{
			"VAR1":   "val1",
			"VAR3":   "stale",
			"CUSTOM": "user-added",
		},
	}
	previous := managed.Entries{
		"env": {"VAR1", "VAR3"},
	}

	merged, entries := MergeWithRemoval(base, user, previous)

	envMap := merged["env"].(map[string]any)
	// VAR1 and VAR2 from base
	assert.Contains(t, envMap, "VAR1")
	assert.Contains(t, envMap, "VAR2")
	// VAR3 was managed but removed from base
	assert.NotContains(t, envMap, "VAR3")
	// CUSTOM was user-added, not managed — but first-run logic isn't applied here
	// because previous is not nil. CUSTOM is preserved because it wasn't in previous.
	assert.Contains(t, envMap, "CUSTOM")

	// Entries should track current base
	require.NotNil(t, entries)
	assert.Equal(t, []string{"VAR1", "VAR2"}, entries.Get("env"))
}

func TestEqual(t *testing.T) {
	tests := []struct {
		a        any
		b        any
		expected bool
		name     string
	}{
		{name: "equal strings", a: "abc", b: "abc", expected: true},
		{name: "different strings", a: "abc", b: "def", expected: false},
		{name: "equal maps", a: map[string]any{"k": "v"}, b: map[string]any{"k": "v"}, expected: true},
		{name: "different maps", a: map[string]any{"k": "v"}, b: map[string]any{"k": "x"}, expected: false},
		{name: "equal arrays", a: []any{"a", "b"}, b: []any{"a", "b"}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Equal(tt.a, tt.b))
		})
	}
}
