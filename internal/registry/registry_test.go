package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRegistry creates a test registry file with the given data
func createTestRegistry(t *testing.T, claudeDir string, data map[string]MarketplaceEntry) {
	t.Helper()

	registryPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(registryPath), 0755))

	jsonData, err := json.Marshal(data)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(registryPath, jsonData, 0644))
}

func TestNewResolver(t *testing.T) {
	claudeDir := "/test/claude"
	resolver := NewResolver(claudeDir)

	assert.NotNil(t, resolver)
	assert.Equal(t, claudeDir, resolver.claudeDir)
}

func TestClaudeDir(t *testing.T) {
	claudeDir := "/test/claude"
	resolver := NewResolver(claudeDir)

	assert.Equal(t, claudeDir, resolver.ClaudeDir())
}

func TestGetKnownMarketplacesPath(t *testing.T) {
	path := GetKnownMarketplacesPath("/home/user/.claude")
	assert.Equal(t, filepath.Join("/home/user/.claude", "plugins", "known_marketplaces.json"), path)
}

func TestGetPluginCachePath(t *testing.T) {
	path := GetPluginCachePath("/home/user/.claude", "my-marketplace")
	assert.Equal(t, filepath.Join("/home/user/.claude", "plugins", "cache", "my-marketplace"), path)
}

func TestIsDefaultMarketplace(t *testing.T) {
	tests := []struct {
		expected bool
		name     string
		mpName   string
		owner    string
		repo     string
	}{
		{
			expected: true,
			name:     "GitHub default marketplace",
			mpName:   DefaultMarketplaceID,
			owner:    DefaultMarketplaceOrg,
			repo:     DefaultMarketplaceRepo,
		},
		{
			expected: true,
			name:     "local path default marketplace",
			mpName:   DefaultMarketplaceID,
			owner:    "",
			repo:     "",
		},
		{
			expected: false,
			name:     "same name different owner",
			mpName:   DefaultMarketplaceID,
			owner:    "other-org",
			repo:     DefaultMarketplaceRepo,
		},
		{
			expected: false,
			name:     "same name different repo",
			mpName:   DefaultMarketplaceID,
			owner:    DefaultMarketplaceOrg,
			repo:     "other-repo",
		},
		{
			expected: false,
			name:     "different name with default owner/repo",
			mpName:   "other-marketplace",
			owner:    DefaultMarketplaceOrg,
			repo:     DefaultMarketplaceRepo,
		},
		{
			expected: false,
			name:     "completely different marketplace",
			mpName:   "other-marketplace",
			owner:    "other-org",
			repo:     "other-repo",
		},
		{
			expected: false,
			name:     "only owner provided but mismatches",
			mpName:   DefaultMarketplaceID,
			owner:    "other-org",
			repo:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDefaultMarketplace(tt.mpName, tt.owner, tt.repo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadRegistry(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			"test-marketplace": {
				InstallLocation: "/path/to/marketplace",
				Source: MarketplaceSource{
					Repo:   "owner/repo",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		registry, err := resolver.loadRegistry()

		require.NoError(t, err)
		assert.Len(t, registry, 1)
		assert.Equal(t, "/path/to/marketplace", registry["test-marketplace"].InstallLocation)
		assert.Equal(t, "owner/repo", registry["test-marketplace"].Source.Repo)
		assert.Equal(t, "github", registry["test-marketplace"].Source.Source)
	})

	t.Run("file not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		resolver := NewResolver(tempDir)
		registry, err := resolver.loadRegistry()

		require.NoError(t, err)
		assert.Empty(t, registry)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "plugins", "known_marketplaces.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(registryPath), 0755))
		require.NoError(t, os.WriteFile(registryPath, []byte("invalid json{"), 0644))

		resolver := NewResolver(tempDir)
		registry, err := resolver.loadRegistry()

		assert.Error(t, err)
		assert.Nil(t, registry)
		assert.Contains(t, err.Error(), "failed to parse registry file")
	})

	t.Run("empty file", func(t *testing.T) {
		tempDir := t.TempDir()
		createTestRegistry(t, tempDir, map[string]MarketplaceEntry{})

		resolver := NewResolver(tempDir)
		registry, err := resolver.loadRegistry()

		require.NoError(t, err)
		assert.Empty(t, registry)
	})

	t.Run("multiple marketplaces", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			"marketplace-1": {
				InstallLocation: "/path/to/mp1",
				Source: MarketplaceSource{
					Repo:   "owner/repo1",
					Source: "github",
				},
			},
			"marketplace-2": {
				InstallLocation: "/path/to/mp2",
				Source: MarketplaceSource{
					Repo:   "owner/repo2",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		registry, err := resolver.loadRegistry()

		require.NoError(t, err)
		assert.Len(t, registry, 2)
		assert.Equal(t, "/path/to/mp1", registry["marketplace-1"].InstallLocation)
		assert.Equal(t, "/path/to/mp2", registry["marketplace-2"].InstallLocation)
	})
}

func TestGetInstallLocation(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			"test-marketplace": {
				InstallLocation: "/path/to/marketplace",
				Source: MarketplaceSource{
					Repo:   "owner/repo",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		path, found, err := resolver.GetInstallLocation("test-marketplace")

		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "/path/to/marketplace", path)
	})

	t.Run("not found", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			"test-marketplace": {
				InstallLocation: "/path/to/marketplace",
				Source: MarketplaceSource{
					Repo:   "owner/repo",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		path, found, err := resolver.GetInstallLocation("nonexistent")

		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, path)
	})

	t.Run("no registry", func(t *testing.T) {
		tempDir := t.TempDir()
		resolver := NewResolver(tempDir)
		path, found, err := resolver.GetInstallLocation("test-marketplace")

		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, path)
	})

	t.Run("registry load error", func(t *testing.T) {
		tempDir := t.TempDir()
		registryPath := filepath.Join(tempDir, "plugins", "known_marketplaces.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(registryPath), 0755))
		require.NoError(t, os.WriteFile(registryPath, []byte("invalid"), 0644))

		resolver := NewResolver(tempDir)
		path, found, err := resolver.GetInstallLocation("test-marketplace")

		assert.Error(t, err)
		assert.False(t, found)
		assert.Empty(t, path)
	})
}

func TestGetResourcePath(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			"test-marketplace": {
				InstallLocation: "/path/to/marketplace",
				Source: MarketplaceSource{
					Repo:   "owner/repo",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		path, found, err := resolver.getResourcePath("test-marketplace", "internal", "file.json")

		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, filepath.Join("/path/to/marketplace", "internal", "file.json"), path)
	})

	t.Run("not found", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			"test-marketplace": {
				InstallLocation: "/path/to/marketplace",
				Source: MarketplaceSource{
					Repo:   "owner/repo",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		path, found, err := resolver.getResourcePath("nonexistent", "file.json")

		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, path)
	})

	t.Run("single path part", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			"test-marketplace": {
				InstallLocation: "/path/to/marketplace",
				Source: MarketplaceSource{
					Repo:   "owner/repo",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		path, found, err := resolver.getResourcePath("test-marketplace", "rules")

		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, filepath.Join("/path/to/marketplace", "rules"), path)
	})
}

func TestBaseSettingsPath(t *testing.T) {
	t.Run("with registry", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			DefaultMarketplaceID: {
				InstallLocation: "/custom/marketplace/path",
				Source: MarketplaceSource{
					Repo:   "mvasilenko/mv-claude-code-marketplace",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		path := resolver.BaseSettingsPath()

		assert.Equal(t, filepath.Join("/custom/marketplace/path", "internal", "cmd", "base_settings.json"), path)
	})

	t.Run("fallback without registry", func(t *testing.T) {
		tempDir := t.TempDir()
		resolver := NewResolver(tempDir)
		path := resolver.BaseSettingsPath()

		expected := filepath.Join(tempDir, "plugins", "marketplaces", DefaultMarketplaceID, "internal", "cmd", "base_settings.json")
		assert.Equal(t, expected, path)
	})
}

func TestBaseSettingsPathForMarketplace(t *testing.T) {
	tempDir := t.TempDir()
	testData := map[string]MarketplaceEntry{
		"custom-mp": {
			InstallLocation: "/custom/mp/path",
			Source: MarketplaceSource{
				Repo:   "owner/custom-mp",
				Source: "github",
			},
		},
	}
	createTestRegistry(t, tempDir, testData)

	resolver := NewResolver(tempDir)
	path := resolver.BaseSettingsPathForMarketplace("custom-mp")

	assert.Equal(t, filepath.Join("/custom/mp/path", "internal", "cmd", "base_settings.json"), path)
}

func TestRulesPath(t *testing.T) {
	t.Run("with registry", func(t *testing.T) {
		tempDir := t.TempDir()
		testData := map[string]MarketplaceEntry{
			DefaultMarketplaceID: {
				InstallLocation: "/custom/marketplace/path",
				Source: MarketplaceSource{
					Repo:   "mvasilenko/mv-claude-code-marketplace",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		path := resolver.RulesPath()

		assert.Equal(t, filepath.Join("/custom/marketplace/path", "rules"), path)
	})

	t.Run("fallback without registry", func(t *testing.T) {
		tempDir := t.TempDir()
		resolver := NewResolver(tempDir)
		path := resolver.RulesPath()

		expected := filepath.Join(tempDir, "plugins", "marketplaces", DefaultMarketplaceID, "rules")
		assert.Equal(t, expected, path)
	})
}

func TestSoftwareConfigPath(t *testing.T) {
	t.Run("with registry and file exists", func(t *testing.T) {
		tempDir := t.TempDir()
		mpDir := filepath.Join(tempDir, "mp")
		configDir := filepath.Join(mpDir, "internal", "software")
		require.NoError(t, os.MkdirAll(configDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "software_config.json"), []byte("{}"), 0644))

		testData := map[string]MarketplaceEntry{
			DefaultMarketplaceID: {
				InstallLocation: mpDir,
				Source: MarketplaceSource{
					Repo:   "mvasilenko/mv-claude-code-marketplace",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		path := resolver.SoftwareConfigPath()

		assert.Equal(t, filepath.Join(mpDir, "internal", "software", "software_config.json"), path)
	})

	t.Run("fallback when file missing", func(t *testing.T) {
		tempDir := t.TempDir()
		resolver := NewResolver(tempDir)
		path := resolver.SoftwareConfigPath()

		expected := filepath.Join(tempDir, "plugins", "marketplaces", DefaultMarketplaceID, "internal", "software", "software_config.json")
		assert.Equal(t, expected, path)
	})
}

func TestGetAllRulesPaths(t *testing.T) {
	t.Run("multiple marketplaces", func(t *testing.T) {
		tempDir := t.TempDir()

		mp1RulesDir := filepath.Join(tempDir, "mp1", "rules")
		mp2RulesDir := filepath.Join(tempDir, "mp2", "rules")
		require.NoError(t, os.MkdirAll(mp1RulesDir, 0755))
		require.NoError(t, os.MkdirAll(mp2RulesDir, 0755))

		testData := map[string]MarketplaceEntry{
			"marketplace-1": {
				InstallLocation: filepath.Join(tempDir, "mp1"),
				Source: MarketplaceSource{
					Repo:   "owner/repo1",
					Source: "github",
				},
			},
			"marketplace-2": {
				InstallLocation: filepath.Join(tempDir, "mp2"),
				Source: MarketplaceSource{
					Repo:   "owner/repo2",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		paths, err := resolver.GetAllRulesPaths()

		require.NoError(t, err)
		assert.Len(t, paths, 2)
		assert.Equal(t, mp1RulesDir, paths["marketplace-1"])
		assert.Equal(t, mp2RulesDir, paths["marketplace-2"])
	})

	t.Run("only existing rules dirs", func(t *testing.T) {
		tempDir := t.TempDir()

		mp1RulesDir := filepath.Join(tempDir, "mp1", "rules")
		require.NoError(t, os.MkdirAll(mp1RulesDir, 0755))

		testData := map[string]MarketplaceEntry{
			"marketplace-1": {
				InstallLocation: filepath.Join(tempDir, "mp1"),
				Source: MarketplaceSource{
					Repo:   "owner/repo1",
					Source: "github",
				},
			},
			"marketplace-2": {
				InstallLocation: filepath.Join(tempDir, "mp2"),
				Source: MarketplaceSource{
					Repo:   "owner/repo2",
					Source: "github",
				},
			},
		}
		createTestRegistry(t, tempDir, testData)

		resolver := NewResolver(tempDir)
		paths, err := resolver.GetAllRulesPaths()

		require.NoError(t, err)
		assert.Len(t, paths, 1)
		assert.Equal(t, mp1RulesDir, paths["marketplace-1"])
		_, exists := paths["marketplace-2"]
		assert.False(t, exists)
	})

	t.Run("empty registry", func(t *testing.T) {
		tempDir := t.TempDir()
		createTestRegistry(t, tempDir, map[string]MarketplaceEntry{})

		resolver := NewResolver(tempDir)
		paths, err := resolver.GetAllRulesPaths()

		require.NoError(t, err)
		assert.Empty(t, paths)
	})

	t.Run("no registry", func(t *testing.T) {
		tempDir := t.TempDir()
		resolver := NewResolver(tempDir)
		paths, err := resolver.GetAllRulesPaths()

		require.NoError(t, err)
		assert.Empty(t, paths)
	})
}

func TestResourcePaths_NoRegistry(t *testing.T) {
	tempDir := t.TempDir()
	resolver := NewResolver(tempDir)

	tests := []struct {
		expectedPath string
		fn           func() string
		name         string
	}{
		{
			expectedPath: filepath.Join(tempDir, "plugins", "marketplaces", DefaultMarketplaceID, "internal", "software", "software_config.json"),
			fn:           resolver.SoftwareConfigPath,
			name:         "SoftwareConfigPath",
		},
		{
			expectedPath: filepath.Join(tempDir, "plugins", "marketplaces", DefaultMarketplaceID, "internal", "cmd", "base_settings.json"),
			fn:           resolver.BaseSettingsPath,
			name:         "BaseSettingsPath",
		},
		{
			expectedPath: filepath.Join(tempDir, "plugins", "marketplaces", DefaultMarketplaceID, "rules"),
			fn:           resolver.RulesPath,
			name:         "RulesPath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.fn()
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}
