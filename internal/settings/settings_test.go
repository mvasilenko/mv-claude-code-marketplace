package settings

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeGitHubRepo(t *testing.T) {
	tests := []struct {
		expected string
		input    string
		name     string
	}{
		{
			name:     "plain owner/repo",
			input:    "owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "HTTPS URL",
			input:    "https://github.com/owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "HTTPS URL with .git",
			input:    "https://github.com/owner/repo.git",
			expected: "owner/repo",
		},
		{
			name:     "SSH URL with .git",
			input:    "git@github.com:owner/repo.git",
			expected: "owner/repo",
		},
		{
			name:     "SSH URL without .git",
			input:    "git@github.com:owner/repo",
			expected: "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeGitHubRepo(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManager_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Load on missing file returns empty map
	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)
	assert.Empty(t, settings)

	// Save settings
	settings["key"] = "value"
	err = mgr.Save(context.Background(), settings)
	require.NoError(t, err)

	// Load returns saved settings
	loaded, err := mgr.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "value", loaded["key"])
}

func TestManager_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "dir")
	mgr := NewManager(nestedDir)

	err := mgr.Save(context.Background(), map[string]any{"key": "val"})
	require.NoError(t, err)

	loaded, err := mgr.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "val", loaded["key"])
}

func TestManager_Save_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.Save(context.Background(), map[string]any{"key": "val"})
	require.NoError(t, err)

	// No temp files should remain
	matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.tmp"))
	assert.Empty(t, matches)
}

func TestManager_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "settings.json"), []byte("{invalid"), 0644)
	require.NoError(t, err)

	mgr := NewManager(tmpDir)
	_, err = mgr.Load(context.Background())
	assert.Error(t, err)
}

func TestManager_GetSettingsPath(t *testing.T) {
	mgr := NewManager("/tmp/test-claude")
	assert.Equal(t, filepath.Join("/tmp/test-claude", "settings.json"), mgr.GetSettingsPath())
}

func TestGetMarketplaceRepo(t *testing.T) {
	tmpDir := t.TempDir()

	settings := map[string]any{
		"extraKnownMarketplaces": map[string]any{
			"test-marketplace": map[string]any{
				"source": map[string]any{
					"source": "github",
					"repo":   "owner/repo",
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "settings.json"), data, 0644)
	require.NoError(t, err)

	mgr := NewManager(tmpDir)
	repo, err := mgr.GetMarketplaceRepo(context.Background(), "test-marketplace")
	assert.NoError(t, err)
	assert.Equal(t, "owner/repo", repo)
}

func TestGetMarketplaceRepo_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	data, err := json.MarshalIndent(map[string]any{}, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "settings.json"), data, 0644)
	require.NoError(t, err)

	mgr := NewManager(tmpDir)
	_, err = mgr.GetMarketplaceRepo(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMarketplaceNotFound)
}

func TestGetMarketplaceRepo_NoRepo(t *testing.T) {
	tmpDir := t.TempDir()

	settings := map[string]any{
		"extraKnownMarketplaces": map[string]any{
			"test-marketplace": map[string]any{
				"source": map[string]any{
					"source": "local",
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "settings.json"), data, 0644)
	require.NoError(t, err)

	mgr := NewManager(tmpDir)
	_, err = mgr.GetMarketplaceRepo(context.Background(), "test-marketplace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no repo URL")
}

func TestAddMarketplace(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.AddMarketplace(context.Background(), "test-mp", "owner", "repo")
	require.NoError(t, err)

	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)

	extra := settings["extraKnownMarketplaces"].(map[string]any)
	mp := extra["test-mp"].(map[string]any)
	source := mp["source"].(map[string]any)

	assert.Equal(t, "github", source["source"])
	assert.Equal(t, "owner/repo", source["repo"])
}

func TestAddMarketplace_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.AddMarketplace(context.Background(), "test-mp", "owner", "repo")
	require.NoError(t, err)

	// Same call should be idempotent
	err = mgr.AddMarketplace(context.Background(), "test-mp", "owner", "repo")
	assert.NoError(t, err)
}

func TestAddMarketplace_ConflictingRepo(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.AddMarketplace(context.Background(), "test-mp", "owner", "repo")
	require.NoError(t, err)

	err = mgr.AddMarketplace(context.Background(), "test-mp", "other", "repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddMarketplaceWithSource_GitHub(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.AddMarketplaceWithSource(context.Background(), "test-mp", "github", "owner/repo", false)
	require.NoError(t, err)

	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)

	extra := settings["extraKnownMarketplaces"].(map[string]any)
	mp := extra["test-mp"].(map[string]any)
	source := mp["source"].(map[string]any)

	assert.Equal(t, "github", source["source"])
	assert.Equal(t, "owner/repo", source["repo"])
}

func TestAddMarketplaceWithSource_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.AddMarketplaceWithSource(context.Background(), "local-mp", "directory", "/path/to/local", false)
	require.NoError(t, err)

	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)

	extra := settings["extraKnownMarketplaces"].(map[string]any)
	mp := extra["local-mp"].(map[string]any)
	source := mp["source"].(map[string]any)

	assert.Equal(t, "directory", source["source"])
	assert.Equal(t, "/path/to/local", source["path"])
}

func TestAddMarketplaceWithSource_ForceReplaces(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.AddMarketplaceWithSource(context.Background(), "test-mp", "github", "owner/repo", false)
	require.NoError(t, err)

	err = mgr.AddMarketplaceWithSource(context.Background(), "test-mp", "directory", "/new/path", true)
	require.NoError(t, err)

	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)

	extra := settings["extraKnownMarketplaces"].(map[string]any)
	mp := extra["test-mp"].(map[string]any)
	source := mp["source"].(map[string]any)

	assert.Equal(t, "directory", source["source"])
	assert.Equal(t, "/new/path", source["path"])
}

func TestEnablePlugins(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.EnablePlugins(context.Background(), []string{"plugin-a", "plugin-b"}, "test-mp")
	require.NoError(t, err)

	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)

	plugins := settings["enabledPlugins"].(map[string]any)
	assert.Equal(t, true, plugins["plugin-a@test-mp"])
	assert.Equal(t, true, plugins["plugin-b@test-mp"])
}

func TestEnablePlugins_PreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// First enable with default true
	err := mgr.EnablePlugins(context.Background(), []string{"plugin-a"}, "test-mp")
	require.NoError(t, err)

	// Manually set to false (user disabled)
	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)
	settings["enabledPlugins"].(map[string]any)["plugin-a@test-mp"] = false
	err = mgr.Save(context.Background(), settings)
	require.NoError(t, err)

	// Re-enable should preserve user's false
	err = mgr.EnablePlugins(context.Background(), []string{"plugin-a"}, "test-mp")
	require.NoError(t, err)

	settings, err = mgr.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, false, settings["enabledPlugins"].(map[string]any)["plugin-a@test-mp"])
}

func TestEnablePluginsWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	defaults := map[string]bool{
		"plugin-a@test-mp": false,
		"plugin-b@test-mp": true,
	}

	err := mgr.EnablePluginsWithDefaults(context.Background(), []string{"plugin-a", "plugin-b"}, "test-mp", defaults)
	require.NoError(t, err)

	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)

	plugins := settings["enabledPlugins"].(map[string]any)
	assert.Equal(t, false, plugins["plugin-a@test-mp"])
	assert.Equal(t, true, plugins["plugin-b@test-mp"])
}

func TestDisablePluginsForMarketplace(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Enable plugins for two marketplaces
	err := mgr.EnablePlugins(context.Background(), []string{"plugin-a", "plugin-b"}, "mp1")
	require.NoError(t, err)
	err = mgr.EnablePlugins(context.Background(), []string{"plugin-c"}, "mp2")
	require.NoError(t, err)

	// Disable all for mp1
	disabled, err := mgr.DisablePluginsForMarketplace(context.Background(), "mp1")
	require.NoError(t, err)
	assert.Len(t, disabled, 2)

	// Verify mp2 plugins are preserved
	settings, err := mgr.Load(context.Background())
	require.NoError(t, err)
	plugins := settings["enabledPlugins"].(map[string]any)
	assert.NotContains(t, plugins, "plugin-a@mp1")
	assert.NotContains(t, plugins, "plugin-b@mp1")
	assert.Contains(t, plugins, "plugin-c@mp2")
}

func TestMarketplaceExists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Non-existent
	exists, _, err := mgr.MarketplaceExists(context.Background(), "test-mp")
	require.NoError(t, err)
	assert.False(t, exists)

	// Add marketplace
	err = mgr.AddMarketplace(context.Background(), "test-mp", "owner", "repo")
	require.NoError(t, err)

	// Exists
	exists, repo, err := mgr.MarketplaceExists(context.Background(), "test-mp")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "owner/repo", repo)
}

func TestMarketplaceExists_NormalizesRepoURL(t *testing.T) {
	tmpDir := t.TempDir()

	settings := map[string]any{
		"extraKnownMarketplaces": map[string]any{
			"test-marketplace": map[string]any{
				"source": map[string]any{
					"source": "github",
					"repo":   "https://github.com/owner/repo",
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "settings.json"), data, 0644)
	require.NoError(t, err)

	mgr := NewManager(tmpDir)
	exists, repo, err := mgr.MarketplaceExists(context.Background(), "test-marketplace")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "owner/repo", repo)
}

func TestMarketplaceExists_DirectorySource(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.AddMarketplaceWithSource(context.Background(), "local-mp", "directory", "/path/to/local", false)
	require.NoError(t, err)

	exists, repoOrPath, err := mgr.MarketplaceExists(context.Background(), "local-mp")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "/path/to/local", repoOrPath)
}

func TestRemoveMarketplace(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Setup: add marketplace and plugins
	err := mgr.AddMarketplace(context.Background(), "test-mp", "owner", "repo")
	require.NoError(t, err)
	err = mgr.EnablePlugins(context.Background(), []string{"plugin-a", "plugin-b"}, "test-mp")
	require.NoError(t, err)

	// Remove
	repo, disabled, err := mgr.RemoveMarketplace(context.Background(), "test-mp")
	require.NoError(t, err)
	assert.Equal(t, "owner/repo", repo)
	assert.Len(t, disabled, 2)

	// Verify removed
	exists, _, err := mgr.MarketplaceExists(context.Background(), "test-mp")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRemoveMarketplace_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, _, err := mgr.RemoveMarketplace(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMarketplaceNotFound)
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		expected string
		input    string
		name     string
	}{
		{
			name:     "expands tilde with forward slash",
			input:    "~/.claude_test",
			expected: filepath.Join(home, ".claude_test"),
		},
		{
			name:     "expands standalone tilde",
			input:    "~",
			expected: home,
		},
		{
			name:     "does not expand absolute path",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "does not expand relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "does not expand tilde in middle of path",
			input:    "/some/~/path",
			expected: "/some/~/path",
		},
		{
			name:     "does not expand tilde without separator",
			input:    "~username",
			expected: "~username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveClaudeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		configValue  string
		envValue     string
		expected     string
		flagOverride string
		name         string
	}{
		{
			name:     "returns default when all empty",
			expected: filepath.Join(home, ".claude"),
		},
		{
			name:         "flag takes priority",
			flagOverride: "/flag/path",
			configValue:  "/config/path",
			envValue:     "/env/path",
			expected:     "/flag/path",
		},
		{
			name:        "config takes priority over env",
			configValue: "/config/path",
			envValue:    "/env/path",
			expected:    "/config/path",
		},
		{
			name:     "env var used when no flag or config",
			envValue: "/env/path",
			expected: "/env/path",
		},
		{
			name:         "expands tilde in flag",
			flagOverride: "~/.claude_flag",
			expected:     filepath.Join(home, ".claude_flag"),
		},
		{
			name:        "expands tilde in config",
			configValue: "~/.claude_config",
			expected:    filepath.Join(home, ".claude_config"),
		},
		{
			name:     "expands tilde in env",
			envValue: "~/.claude_env",
			expected: filepath.Join(home, ".claude_env"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("CLAUDE_CONFIG_DIR", tt.envValue)
			} else {
				if os.Getenv("CLAUDE_CONFIG_DIR") != "" {
					t.Setenv("CLAUDE_CONFIG_DIR", "")
				}
			}

			result := ResolveClaudeDir(tt.flagOverride, tt.configValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
