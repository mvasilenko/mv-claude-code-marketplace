package syncer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
)

func newTestPrinter() *printer.Printer {
	return printer.New(false, true, false)
}

func TestSyncSettings_InitMode_CreatesNew(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")

	base := map[string]any{
		"apiKey": "test-key",
		"env":    map[string]any{"VAR1": "val1"},
	}

	result, err := SyncSettings(context.Background(), SettingsSyncInput{
		BaseSettings: base,
		ClaudeDir:    claudeDir,
		IsUpdate:     false,
		Printer:      newTestPrinter(),
	})

	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.True(t, result.SettingsChanged)
	assert.Empty(t, result.BackupPath)

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	require.NoError(t, err)

	var saved map[string]any
	require.NoError(t, json.Unmarshal(data, &saved))
	assert.Equal(t, "test-key", saved["apiKey"])
}

func TestSyncSettings_InitMode_MergesWithExisting(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// Pre-existing user settings
	userSettings := map[string]any{
		"customKey": "user-value",
		"apiKey":    "user-key",
	}
	data, err := json.MarshalIndent(userSettings, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644))

	base := map[string]any{
		"apiKey": "base-key",
		"newKey": "new-value",
	}

	result, err := SyncSettings(context.Background(), SettingsSyncInput{
		BaseSettings: base,
		ClaudeDir:    claudeDir,
		IsUpdate:     false,
		Printer:      newTestPrinter(),
	})

	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.True(t, result.SettingsChanged)
	assert.NotEmpty(t, result.BackupPath)

	// Verify merged settings
	savedData, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	require.NoError(t, err)

	var saved map[string]any
	require.NoError(t, json.Unmarshal(savedData, &saved))

	// User value wins for existing key
	assert.Equal(t, "user-key", saved["apiKey"])
	// User custom key preserved
	assert.Equal(t, "user-value", saved["customKey"])
	// New key from base added
	assert.Equal(t, "new-value", saved["newKey"])
}

func TestSyncSettings_NoChanges(t *testing.T) {
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	settings := map[string]any{
		"apiKey": "value",
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644))

	// Same base as user
	result, err := SyncSettings(context.Background(), SettingsSyncInput{
		BaseSettings: settings,
		ClaudeDir:    claudeDir,
		IsUpdate:     false,
		Printer:      newTestPrinter(),
	})

	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.False(t, result.SettingsChanged)
	assert.Empty(t, result.BackupPath)
}

func TestCalculateStats(t *testing.T) {
	tests := []struct {
		base     any
		expected MergeStats
		merged   any
		name     string
		user     any
	}{
		{
			name: "all new keys",
			base: map[string]any{
				"key1": "val1",
				"key2": "val2",
			},
			user:   map[string]any{},
			merged: map[string]any{"key1": "val1", "key2": "val2"},
			expected: MergeStats{
				KeysAdded: 2,
			},
		},
		{
			name: "mixed keys with arrays and objects",
			base: map[string]any{
				"key1": "val1",
				"arr":  []any{"a", "b"},
				"obj":  map[string]any{"nested": "value"},
				"new":  "added",
			},
			user: map[string]any{
				"key1": "user-val",
				"arr":  []any{"c"},
				"obj":  map[string]any{"other": "data"},
			},
			merged: map[string]any{
				"key1": "user-val",
				"arr":  []any{"a", "b", "c"},
				"obj":  map[string]any{"nested": "value", "other": "data"},
				"new":  "added",
			},
			expected: MergeStats{
				ArraysMerged:  1,
				KeysAdded:     1,
				ObjectsMerged: 1,
			},
		},
		{
			name:     "nil user settings",
			base:     map[string]any{"key1": "val1"},
			user:     nil,
			merged:   map[string]any{"key1": "val1"},
			expected: MergeStats{KeysAdded: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateStats(tt.base, tt.user, tt.merged)
			assert.Equal(t, tt.expected, stats)
		})
	}
}
