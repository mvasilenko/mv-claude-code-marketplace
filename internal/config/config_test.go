package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.True(t, cfg.AutoUpdate.Enabled)
	assert.Equal(t, "3h", cfg.AutoUpdate.CheckInterval)
	assert.Equal(t, time.Time{}, cfg.LastUpdateCheck)
	assert.Equal(t, []string{"mv-claude-code-marketplace"}, cfg.Marketplaces)
	assert.Empty(t, cfg.Software.ExcludedTools)
	assert.Empty(t, cfg.Software.LastInstalledTools)
}

func TestManager_Load(t *testing.T) {
	tests := []struct {
		fileContent   string
		name          string
		expectDefault bool
		expectError   bool
		fileExists    bool
	}{
		{
			name:          "file not exists returns defaults",
			fileExists:    false,
			expectDefault: true,
		},
		{
			name:        "valid file loads successfully",
			fileExists:  true,
			fileContent: `{"autoUpdate":{"enabled":false,"checkInterval":"12h"},"marketplaces":["test-mp"]}`,
		},
		{
			name:        "invalid JSON returns error",
			fileExists:  true,
			fileContent: `{invalid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			mgr := NewManager(tmpDir)

			if tt.fileExists {
				err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(tt.fileContent), 0644)
				require.NoError(t, err)
			}

			cfg, err := mgr.Load(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectDefault {
				assert.True(t, cfg.AutoUpdate.Enabled)
				assert.Equal(t, "3h", cfg.AutoUpdate.CheckInterval)
			} else {
				assert.False(t, cfg.AutoUpdate.Enabled)
				assert.Equal(t, "12h", cfg.AutoUpdate.CheckInterval)
				assert.Equal(t, []string{"test-mp"}, cfg.Marketplaces)
			}
		})
	}
}

func TestManager_Save(t *testing.T) {
	t.Run("creates directory and saves atomically", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, "nested", "config")
		mgr := NewManager(configDir)

		cfg := &Config{
			AutoUpdate: AutoUpdateConfig{
				CheckInterval: "48h",
				Enabled:       true,
			},
			Marketplaces: []string{"test"},
		}

		err := mgr.Save(context.Background(), cfg)
		require.NoError(t, err)

		configPath := filepath.Join(configDir, "config.json")
		assert.FileExists(t, configPath)

		matches, _ := filepath.Glob(filepath.Join(configDir, "*.tmp"))
		assert.Empty(t, matches)

		data, err := os.ReadFile(configPath)
		require.NoError(t, err)
		var loaded Config
		err = json.Unmarshal(data, &loaded)
		require.NoError(t, err)
		assert.Equal(t, cfg.AutoUpdate.Enabled, loaded.AutoUpdate.Enabled)
	})

	t.Run("roundtrip preserves data", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewManager(tmpDir)

		original := &Config{
			AutoUpdate: AutoUpdateConfig{
				CheckInterval: "6h",
				Enabled:       false,
			},
			ClaudeConfigDir:      "/custom/path",
			LastUpdateCheck:      time.Now().Truncate(time.Second),
			MarketplaceCachePath: "~/cache",
			Marketplaces:         []string{"mp1", "mp2"},
			Software: SoftwareConfig{
				ExcludedTools:      []string{"tool1"},
				LastInstalledTools: []string{"tool2", "tool3"},
			},
		}

		err := mgr.Save(context.Background(), original)
		require.NoError(t, err)

		loaded, err := mgr.Load(context.Background())
		require.NoError(t, err)

		assert.Equal(t, original.AutoUpdate.Enabled, loaded.AutoUpdate.Enabled)
		assert.Equal(t, original.AutoUpdate.CheckInterval, loaded.AutoUpdate.CheckInterval)
		assert.Equal(t, original.ClaudeConfigDir, loaded.ClaudeConfigDir)
		assert.Equal(t, original.MarketplaceCachePath, loaded.MarketplaceCachePath)
		assert.Equal(t, original.Marketplaces, loaded.Marketplaces)
		assert.Equal(t, original.Software.ExcludedTools, loaded.Software.ExcludedTools)
		assert.Equal(t, original.Software.LastInstalledTools, loaded.Software.LastInstalledTools)
	})
}

func TestGetNestedValue(t *testing.T) {
	cfg := &Config{
		AutoUpdate: AutoUpdateConfig{
			CheckInterval: "12h",
			Enabled:       true,
		},
		Backend:              "bedrock",
		ClaudeConfigDir:      "/custom/dir",
		ExtensionNamespaces:  []string{"ice", "tools"},
		MarketplaceCachePath: "~/cache/path",
		Marketplaces:         []string{"mp1", "mp2"},
		Software: SoftwareConfig{
			ExcludedTools: []string{"tool1", "tool2"},
		},
	}

	tests := []struct {
		expected    any
		key         string
		name        string
		expectError bool
	}{
		{
			name:     "autoUpdate.enabled",
			key:      "autoUpdate.enabled",
			expected: true,
		},
		{
			name:     "autoUpdate.checkInterval",
			key:      "autoUpdate.checkInterval",
			expected: "12h",
		},
		{
			name:     "backend",
			key:      "backend",
			expected: "bedrock",
		},
		{
			name:     "autoUpdate object",
			key:      "autoUpdate",
			expected: cfg.AutoUpdate,
		},
		{
			name:     "claudeConfigDir",
			key:      "claudeConfigDir",
			expected: "/custom/dir",
		},
		{
			name:     "extensionNamespaces",
			key:      "extensionNamespaces",
			expected: []string{"ice", "tools"},
		},
		{
			name:     "marketplaceCachePath",
			key:      "marketplaceCachePath",
			expected: "~/cache/path",
		},
		{
			name:     "marketplaces",
			key:      "marketplaces",
			expected: []string{"mp1", "mp2"},
		},
		{
			name:     "software object",
			key:      "software",
			expected: cfg.Software,
		},
		{
			name:     "software.excludedTools",
			key:      "software.excludedTools",
			expected: []string{"tool1", "tool2"},
		},
		{
			name:        "unknown key",
			key:         "unknown",
			expectError: true,
		},
		{
			name:        "unknown nested key",
			key:         "autoUpdate.unknown",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := getNestedValue(cfg, tt.key)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, val)
			}
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := []struct {
		key         string
		name        string
		validate    func(*testing.T, *Config)
		value       string
		expectError bool
	}{
		{
			name:  "set autoUpdate.enabled to true",
			key:   "autoUpdate.enabled",
			value: "true",
			validate: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.AutoUpdate.Enabled)
			},
		},
		{
			name:  "set autoUpdate.enabled to false",
			key:   "autoUpdate.enabled",
			value: "false",
			validate: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.AutoUpdate.Enabled)
			},
		},
		{
			name:        "invalid boolean value",
			key:         "autoUpdate.enabled",
			value:       "notabool",
			expectError: true,
		},
		{
			name:  "set backend",
			key:   "backend",
			value: "bedrock",
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "bedrock", cfg.Backend)
			},
		},
		{
			name:  "set valid duration",
			key:   "autoUpdate.checkInterval",
			value: "48h",
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "48h", cfg.AutoUpdate.CheckInterval)
			},
		},
		{
			name:        "invalid duration",
			key:         "autoUpdate.checkInterval",
			value:       "invalid",
			expectError: true,
		},
		{
			name:  "set claudeConfigDir",
			key:   "claudeConfigDir",
			value: "/new/path",
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "/new/path", cfg.ClaudeConfigDir)
			},
		},
		{
			name:  "set marketplaceCachePath",
			key:   "marketplaceCachePath",
			value: "~/new/cache",
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "~/new/cache", cfg.MarketplaceCachePath)
			},
		},
		{
			name:  "set gitProtocol ssh",
			key:   "gitProtocol",
			value: "ssh",
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, GitProtocolSSH, cfg.GitProtocol)
			},
		},
		{
			name:        "invalid gitProtocol",
			key:         "gitProtocol",
			value:       "ftp",
			expectError: true,
		},
		{
			name:        "cannot set marketplaces directly",
			key:         "marketplaces",
			value:       "test",
			expectError: true,
		},
		{
			name:        "cannot set lastUpdateCheck",
			key:         "lastUpdateCheck",
			value:       "2024-01-01",
			expectError: true,
		},
		{
			name:        "cannot set entire autoUpdate object",
			key:         "autoUpdate",
			value:       "{}",
			expectError: true,
		},
		{
			name:        "unknown key",
			key:         "unknown",
			value:       "value",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			err := setNestedValue(cfg, tt.key, tt.value)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestListAddValue(t *testing.T) {
	tests := []struct {
		key         string
		name        string
		setup       func(*Config)
		validate    func(*testing.T, *Config)
		value       string
		expectError bool
	}{
		{
			name:  "add to extensionNamespaces",
			key:   "extensionNamespaces",
			value: "tools",
			setup: func(cfg *Config) {
				cfg.ExtensionNamespaces = []string{"ice"}
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Contains(t, cfg.ExtensionNamespaces, "tools")
				assert.Contains(t, cfg.ExtensionNamespaces, "ice")
			},
		},
		{
			name:  "duplicate extensionNamespace",
			key:   "extensionNamespaces",
			value: "ice",
			setup: func(cfg *Config) {
				cfg.ExtensionNamespaces = []string{"ice"}
			},
			expectError: true,
		},
		{
			name:  "add to marketplaces",
			key:   "marketplaces",
			value: "new-mp",
			setup: func(cfg *Config) {
				cfg.Marketplaces = []string{"existing"}
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Contains(t, cfg.Marketplaces, "new-mp")
				assert.Contains(t, cfg.Marketplaces, "existing")
			},
		},
		{
			name:  "add to excludedTools",
			key:   "software.excludedTools",
			value: "new-tool",
			setup: func(cfg *Config) {
				cfg.Software.ExcludedTools = []string{"old-tool"}
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Contains(t, cfg.Software.ExcludedTools, "new-tool")
				assert.Contains(t, cfg.Software.ExcludedTools, "old-tool")
			},
		},
		{
			name:  "duplicate marketplace",
			key:   "marketplaces",
			value: "existing",
			setup: func(cfg *Config) {
				cfg.Marketplaces = []string{"existing"}
			},
			expectError: true,
		},
		{
			name:  "duplicate excludedTool",
			key:   "software.excludedTools",
			value: "tool1",
			setup: func(cfg *Config) {
				cfg.Software.ExcludedTools = []string{"tool1"}
			},
			expectError: true,
		},
		{
			name:        "unsupported key",
			key:         "unsupported",
			value:       "value",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if tt.setup != nil {
				tt.setup(cfg)
			}

			err := listAddValue(cfg, tt.key, tt.value)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestListRemoveValue(t *testing.T) {
	tests := []struct {
		key         string
		name        string
		setup       func(*Config)
		validate    func(*testing.T, *Config)
		value       string
		expectError bool
	}{
		{
			name:  "remove from extensionNamespaces",
			key:   "extensionNamespaces",
			value: "ice",
			setup: func(cfg *Config) {
				cfg.ExtensionNamespaces = []string{"ice", "tools"}
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.NotContains(t, cfg.ExtensionNamespaces, "ice")
				assert.Contains(t, cfg.ExtensionNamespaces, "tools")
			},
		},
		{
			name:  "extensionNamespace not found",
			key:   "extensionNamespaces",
			value: "nonexistent",
			setup: func(cfg *Config) {
				cfg.ExtensionNamespaces = []string{"ice"}
			},
			expectError: true,
		},
		{
			name:  "remove from marketplaces",
			key:   "marketplaces",
			value: "mp-to-remove",
			setup: func(cfg *Config) {
				cfg.Marketplaces = []string{"keep", "mp-to-remove"}
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.NotContains(t, cfg.Marketplaces, "mp-to-remove")
				assert.Contains(t, cfg.Marketplaces, "keep")
			},
		},
		{
			name:  "remove from excludedTools",
			key:   "software.excludedTools",
			value: "tool-to-remove",
			setup: func(cfg *Config) {
				cfg.Software.ExcludedTools = []string{"keep", "tool-to-remove"}
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.NotContains(t, cfg.Software.ExcludedTools, "tool-to-remove")
				assert.Contains(t, cfg.Software.ExcludedTools, "keep")
			},
		},
		{
			name:  "marketplace not found",
			key:   "marketplaces",
			value: "nonexistent",
			setup: func(cfg *Config) {
				cfg.Marketplaces = []string{"existing"}
			},
			expectError: true,
		},
		{
			name:  "tool not found",
			key:   "software.excludedTools",
			value: "nonexistent",
			setup: func(cfg *Config) {
				cfg.Software.ExcludedTools = []string{"existing"}
			},
			expectError: true,
		},
		{
			name:        "unsupported key",
			key:         "unsupported",
			value:       "value",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if tt.setup != nil {
				tt.setup(cfg)
			}

			err := listRemoveValue(cfg, tt.key, tt.value)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestConfig_IsToolExcluded(t *testing.T) {
	cfg := &Config{
		Software: SoftwareConfig{
			ExcludedTools: []string{"excluded1", "excluded2"},
		},
	}

	assert.True(t, cfg.IsToolExcluded("excluded1"))
	assert.True(t, cfg.IsToolExcluded("excluded2"))
	assert.False(t, cfg.IsToolExcluded("notexcluded"))
	assert.False(t, cfg.IsToolExcluded(""))
}

func TestConfig_GetNewTools(t *testing.T) {
	tests := []struct {
		allTools      []string
		excludedTools []string
		expected      []string
		lastInstalled []string
		name          string
	}{
		{
			name:          "detects new tools",
			allTools:      []string{"tool1", "tool2", "tool3"},
			lastInstalled: []string{"tool1"},
			excludedTools: []string{},
			expected:      []string{"tool2", "tool3"},
		},
		{
			name:          "excludes excluded tools",
			allTools:      []string{"tool1", "tool2", "tool3"},
			lastInstalled: []string{"tool1"},
			excludedTools: []string{"tool2"},
			expected:      []string{"tool3"},
		},
		{
			name:          "no new tools",
			allTools:      []string{"tool1", "tool2"},
			lastInstalled: []string{"tool1", "tool2"},
			excludedTools: []string{},
			expected:      nil,
		},
		{
			name:          "all tools new",
			allTools:      []string{"tool1", "tool2"},
			lastInstalled: []string{},
			excludedTools: []string{},
			expected:      []string{"tool1", "tool2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Software: SoftwareConfig{
					ExcludedTools:      tt.excludedTools,
					LastInstalledTools: tt.lastInstalled,
				},
			}

			result := cfg.GetNewTools(tt.allTools)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_GetMarketplaceCachePath(t *testing.T) {
	tests := []struct {
		cachePath string
		claudeDir string
		name      string
	}{
		{
			name:      "custom path with tilde expansion",
			cachePath: "~/custom/cache",
			claudeDir: "/some/claude/dir",
		},
		{
			name:      "custom path without tilde",
			cachePath: "/absolute/path",
			claudeDir: "/some/claude/dir",
		},
		{
			name:      "default path uses claudeDir",
			cachePath: "",
			claudeDir: "/custom/claude/dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				MarketplaceCachePath: tt.cachePath,
			}

			result := cfg.GetMarketplaceCachePath(tt.claudeDir)

			switch tt.cachePath {
			case "":
				assert.Equal(t, filepath.Join(tt.claudeDir, DefaultMarketplaceCacheDir), result)
			case "/absolute/path":
				assert.Equal(t, "/absolute/path", result)
			default:
				assert.NotContains(t, result, "~")
				assert.Contains(t, filepath.ToSlash(result), "custom/cache")
			}
		})
	}
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
			input:    "~/.claudectl_test",
			expected: filepath.Join(home, ".claudectl_test"),
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

func TestConfig_AddExcludedTool(t *testing.T) {
	cfg := &Config{
		Software: SoftwareConfig{
			ExcludedTools: []string{"existing"},
		},
	}

	cfg.AddExcludedTool("newtool")
	assert.Contains(t, cfg.Software.ExcludedTools, "newtool")

	// Adding duplicate should not add again
	cfg.AddExcludedTool("newtool")
	count := 0
	for _, tool := range cfg.Software.ExcludedTools {
		if tool == "newtool" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestConfig_RemoveExcludedTool(t *testing.T) {
	cfg := &Config{
		Software: SoftwareConfig{
			ExcludedTools: []string{"keep", "remove"},
		},
	}

	cfg.RemoveExcludedTool("remove")
	assert.NotContains(t, cfg.Software.ExcludedTools, "remove")
	assert.Contains(t, cfg.Software.ExcludedTools, "keep")

	// Removing non-existent should not panic
	cfg.RemoveExcludedTool("nonexistent")
	assert.Len(t, cfg.Software.ExcludedTools, 1)
}

func TestConfig_UpdateLastInstalledTools(t *testing.T) {
	cfg := &Config{
		Software: SoftwareConfig{
			LastInstalledTools: []string{"old1", "old2"},
		},
	}

	cfg.UpdateLastInstalledTools([]string{"new1", "new2", "new3"})
	assert.Equal(t, []string{"new1", "new2", "new3"}, cfg.Software.LastInstalledTools)
}

func TestManager_GetConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	expected := filepath.Join(tmpDir, "config.json")
	assert.Equal(t, expected, mgr.GetConfigPath())
}

func TestManager_SaveExtensionNamespaces(t *testing.T) {
	tests := []struct {
		existing      []string
		name          string
		newNamespaces []string
		expectedMsg   string
	}{
		{
			name:          "saves new namespaces",
			existing:      nil,
			newNamespaces: []string{"ice", "tools"},
			expectedMsg:   "Saved extension namespaces: [ice tools]",
		},
		{
			name:          "updates existing namespaces",
			existing:      []string{"old"},
			newNamespaces: []string{"new1", "new2"},
			expectedMsg:   "Saved extension namespaces: [new1 new2]",
		},
		{
			name:          "clears namespaces with nil slice",
			existing:      []string{"old"},
			newNamespaces: nil,
			expectedMsg:   "Saved extension namespaces: []",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			mgr := NewManager(tmpDir)

			cfg := DefaultConfig()
			if tt.existing != nil {
				cfg.ExtensionNamespaces = tt.existing
				err := mgr.Save(context.Background(), cfg)
				require.NoError(t, err)
			}

			msg, err := mgr.SaveExtensionNamespaces(context.Background(), cfg, tt.newNamespaces)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMsg, msg)
			assert.Equal(t, tt.newNamespaces, cfg.ExtensionNamespaces)

			loaded, err := mgr.Load(context.Background())
			require.NoError(t, err)
			if len(tt.newNamespaces) == 0 {
				assert.Empty(t, loaded.ExtensionNamespaces)
			} else {
				assert.Equal(t, tt.newNamespaces, loaded.ExtensionNamespaces)
			}
		})
	}
}

func TestResolveConfigDir(t *testing.T) {
	t.Run("env var takes priority", func(t *testing.T) {
		t.Setenv("CLAUDECTL_CONFIG_DIR", "/env/config")

		result := ResolveConfigDir(nil)
		assert.Equal(t, "/env/config", result)
	})

	t.Run("env var with tilde expansion", func(t *testing.T) {
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		t.Setenv("CLAUDECTL_CONFIG_DIR", "~/.claudectl")

		result := ResolveConfigDir(nil)
		assert.Equal(t, filepath.Join(home, ".claudectl"), result)
	})
}

func TestGitProtocol_Validate(t *testing.T) {
	tests := []struct {
		name        string
		protocol    GitProtocol
		expectError bool
	}{
		{name: "ssh valid", protocol: GitProtocolSSH},
		{name: "https valid", protocol: GitProtocolHTTPS},
		{name: "empty valid", protocol: ""},
		{name: "invalid", protocol: "ftp", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.protocol.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
