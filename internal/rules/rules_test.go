package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry/mocks"
)

// setupMockResolver creates a mock resolver with common expectations
func setupMockResolver(t *testing.T, claudeDir string) *mocks.MockPathResolver {
	resolver := mocks.NewMockPathResolver(t)
	resolver.EXPECT().RulesPath().Return(filepath.Join(claudeDir, "rules")).Maybe()
	resolver.EXPECT().ClaudeDir().Return(claudeDir).Maybe()
	return resolver
}

// createTestRegistry creates a test registry file with the given data
func createTestRegistry(t *testing.T, claudeDir string, data map[string]registry.MarketplaceEntry) {
	t.Helper()

	registryPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(registryPath), 0755))

	jsonData, err := json.Marshal(data)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(registryPath, jsonData, 0644))
}

// createRuleFile creates a test rule file with the given content
func createRuleFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644))
}

func TestNewManager(t *testing.T) {
	claudeDir := "/test/claude"
	resolver := setupMockResolver(t, claudeDir)
	mgr := NewManager(resolver)

	assert.NotNil(t, mgr)
	assert.Equal(t, resolver, mgr.resolver)
	assert.Equal(t, filepath.Join(claudeDir, "rules"), mgr.rulesDir)
}

func TestGetRulesDir(t *testing.T) {
	claudeDir := "/test/claude"
	resolver := setupMockResolver(t, claudeDir)
	mgr := NewManager(resolver)

	assert.Equal(t, filepath.Join(claudeDir, "rules"), mgr.GetRulesDir())
}

func TestGetWatermark(t *testing.T) {
	tests := []struct {
		expected      string
		marketplaceID string
	}{
		{expected: "Marketplace: my-marketplace", marketplaceID: "my-marketplace"},
		{expected: "Marketplace: test", marketplaceID: "test"},
		{expected: "Marketplace: mv-claude-code-marketplace", marketplaceID: "mv-claude-code-marketplace"},
	}

	for _, tt := range tests {
		t.Run(tt.marketplaceID, func(t *testing.T) {
			assert.Equal(t, tt.expected, getWatermark(tt.marketplaceID))
		})
	}
}

func TestHasWatermark(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
		name     string
	}{
		{
			content:  "# Test Rule\n\nSome content here.\n\n<!-- Marketplace: my-marketplace -->\n",
			expected: true,
			name:     "with watermark at end",
		},
		{
			content:  "# Test Rule\n\nSome content here.\n\n<!-- Marketplace: other-marketplace -->\n",
			expected: true,
			name:     "with different marketplace watermark",
		},
		{
			content:  "# Test Rule\n\nSome content here without watermark.\n",
			expected: false,
			name:     "without watermark",
		},
		{
			content:  "<!-- Marketplace: my-marketplace -->\n\n# Test Rule\n\nLine 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\n",
			expected: false,
			name:     "watermark not in last lines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.md")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			resolver := setupMockResolver(t, tempDir)
			mgr := NewManager(resolver)
			hasWM, err := mgr.hasWatermark(testFile)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, hasWM)
		})
	}
}

func TestGenerateBackupPath(t *testing.T) {
	resolver := setupMockResolver(t, "/test")
	mgr := NewManager(resolver)

	backupPath := mgr.generateBackupPath("/test/rules/my-rule.md")

	assert.Contains(t, filepath.ToSlash(backupPath), "/test/rules/my-rule.")
	assert.Contains(t, filepath.ToSlash(backupPath), ".backup.md")
}

func TestCopyFile_AddsWatermark(t *testing.T) {
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "source")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	sourceFile := filepath.Join(sourceDir, "rule.md")
	require.NoError(t, os.WriteFile(sourceFile, []byte("# My Rule\n\nSome content."), 0644))

	resolver := setupMockResolver(t, tempDir)
	mgr := NewManager(resolver)
	targetFile := filepath.Join(tempDir, "target.md")
	err := mgr.copyFile(sourceFile, targetFile, "test-marketplace")

	require.NoError(t, err)

	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# My Rule")
	assert.Contains(t, string(content), "<!-- Marketplace: test-marketplace -->")
}

func TestCopyRules_MultipleMarketplaces(t *testing.T) {
	tempDir := t.TempDir()

	mp1Dir := filepath.Join(tempDir, "mp1")
	mp1RulesDir := filepath.Join(mp1Dir, "rules")
	createRuleFile(t, mp1RulesDir, "rule1.md", "# Rule 1 from MP1")

	mp2Dir := filepath.Join(tempDir, "mp2")
	mp2RulesDir := filepath.Join(mp2Dir, "rules")
	createRuleFile(t, mp2RulesDir, "rule2.md", "# Rule 2 from MP2")

	testData := map[string]registry.MarketplaceEntry{
		"mp1": {
			InstallLocation: mp1Dir,
			Source: registry.MarketplaceSource{
				Repo:   "owner/mp1",
				Source: "github",
			},
		},
		"mp2": {
			InstallLocation: mp2Dir,
			Source: registry.MarketplaceSource{
				Repo:   "owner/mp2",
				Source: "github",
			},
		},
	}
	createTestRegistry(t, tempDir, testData)

	resolver := registry.NewResolver(tempDir)
	mgr := NewManager(resolver)
	result, err := mgr.CopyRules()

	require.NoError(t, err)
	assert.Equal(t, 2, len(result.CopiedFiles))

	_, err = os.Stat(filepath.Join(mgr.rulesDir, "mp1_rule1.md"))
	assert.NoError(t, err, "mp1_rule1.md should exist")

	_, err = os.Stat(filepath.Join(mgr.rulesDir, "mp2_rule2.md"))
	assert.NoError(t, err, "mp2_rule2.md should exist")
}

func TestCopyRules_SameFilenameAcrossMarketplaces(t *testing.T) {
	tempDir := t.TempDir()

	mp1Dir := filepath.Join(tempDir, "mp1")
	mp1RulesDir := filepath.Join(mp1Dir, "rules")
	createRuleFile(t, mp1RulesDir, "common.md", "# Common rule from MP1")

	mp2Dir := filepath.Join(tempDir, "mp2")
	mp2RulesDir := filepath.Join(mp2Dir, "rules")
	createRuleFile(t, mp2RulesDir, "common.md", "# Common rule from MP2")

	testData := map[string]registry.MarketplaceEntry{
		"mp1": {
			InstallLocation: mp1Dir,
			Source: registry.MarketplaceSource{
				Repo:   "owner/mp1",
				Source: "github",
			},
		},
		"mp2": {
			InstallLocation: mp2Dir,
			Source: registry.MarketplaceSource{
				Repo:   "owner/mp2",
				Source: "github",
			},
		},
	}
	createTestRegistry(t, tempDir, testData)

	resolver := registry.NewResolver(tempDir)
	mgr := NewManager(resolver)
	result, err := mgr.CopyRules()

	require.NoError(t, err)
	assert.Equal(t, 2, len(result.CopiedFiles))

	mp1Content, err := os.ReadFile(filepath.Join(mgr.rulesDir, "mp1_common.md"))
	require.NoError(t, err)
	assert.Contains(t, string(mp1Content), "MP1")

	mp2Content, err := os.ReadFile(filepath.Join(mgr.rulesDir, "mp2_common.md"))
	require.NoError(t, err)
	assert.Contains(t, string(mp2Content), "MP2")
}

func TestCopyRules_EdgeCases(t *testing.T) {
	tests := []struct {
		assertions   func(t *testing.T, mgr *Manager, result *CopyRulesResult, err error)
		name         string
		setupMarkets func(t *testing.T, tempDir string) map[string]registry.MarketplaceEntry
	}{
		{
			assertions: func(t *testing.T, mgr *Manager, result *CopyRulesResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, 0, len(result.CopiedFiles))
			},
			name: "no marketplaces",
			setupMarkets: func(t *testing.T, tempDir string) map[string]registry.MarketplaceEntry {
				return map[string]registry.MarketplaceEntry{}
			},
		},
		{
			assertions: func(t *testing.T, mgr *Manager, result *CopyRulesResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, 0, len(result.CopiedFiles))
			},
			name: "no rule files",
			setupMarkets: func(t *testing.T, tempDir string) map[string]registry.MarketplaceEntry {
				mpDir := filepath.Join(tempDir, "mp1")
				mpRulesDir := filepath.Join(mpDir, "rules")
				require.NoError(t, os.MkdirAll(mpRulesDir, 0755))

				return map[string]registry.MarketplaceEntry{
					"mp1": {
						InstallLocation: mpDir,
						Source: registry.MarketplaceSource{
							Repo:   "owner/mp1",
							Source: "github",
						},
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testData := tt.setupMarkets(t, tempDir)
			createTestRegistry(t, tempDir, testData)

			resolver := registry.NewResolver(tempDir)
			mgr := NewManager(resolver)
			result, err := mgr.CopyRules()

			tt.assertions(t, mgr, result, err)
		})
	}
}

func TestCopyRules_BacksUpExistingFileWithoutWatermark(t *testing.T) {
	tempDir := t.TempDir()

	mpDir := filepath.Join(tempDir, "mp1")
	mpRulesDir := filepath.Join(mpDir, "rules")
	createRuleFile(t, mpRulesDir, "rule.md", "# New Rule")

	testData := map[string]registry.MarketplaceEntry{
		"mp1": {
			InstallLocation: mpDir,
			Source: registry.MarketplaceSource{
				Repo:   "owner/mp1",
				Source: "github",
			},
		},
	}
	createTestRegistry(t, tempDir, testData)

	resolver := registry.NewResolver(tempDir)
	mgr := NewManager(resolver)
	require.NoError(t, os.MkdirAll(mgr.rulesDir, 0755))
	existingFile := filepath.Join(mgr.rulesDir, "mp1_rule.md")
	require.NoError(t, os.WriteFile(existingFile, []byte("# User's custom rule"), 0644))

	result, err := mgr.CopyRules()

	require.NoError(t, err)
	assert.Equal(t, 1, len(result.CopiedFiles))
	assert.Equal(t, 1, len(result.BackedUp), "should have backed up existing file")

	entries, err := os.ReadDir(mgr.rulesDir)
	require.NoError(t, err)

	backupFound := false
	for _, e := range entries {
		if e.Name() != "mp1_rule.md" && filepath.Ext(e.Name()) == ".md" {
			backupFound = true
			break
		}
	}
	assert.True(t, backupFound, "backup file should exist")
}

func TestCopyRules_OverwritesFileWithWatermark(t *testing.T) {
	tempDir := t.TempDir()

	mpDir := filepath.Join(tempDir, "mp1")
	mpRulesDir := filepath.Join(mpDir, "rules")
	createRuleFile(t, mpRulesDir, "rule.md", "# Updated Rule")

	testData := map[string]registry.MarketplaceEntry{
		"mp1": {
			InstallLocation: mpDir,
			Source: registry.MarketplaceSource{
				Repo:   "owner/mp1",
				Source: "github",
			},
		},
	}
	createTestRegistry(t, tempDir, testData)

	resolver := registry.NewResolver(tempDir)
	mgr := NewManager(resolver)
	require.NoError(t, os.MkdirAll(mgr.rulesDir, 0755))
	existingFile := filepath.Join(mgr.rulesDir, "mp1_rule.md")
	require.NoError(t, os.WriteFile(existingFile, []byte("# Old Rule\n\n<!-- Marketplace: mp1 -->"), 0644))

	result, err := mgr.CopyRules()

	require.NoError(t, err)
	assert.Equal(t, 1, len(result.CopiedFiles))

	entries, err := os.ReadDir(mgr.rulesDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "should only have the rule file, no backup")

	content, err := os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Updated Rule")
}

func TestCopyRules_NoBackupOnSecondUpdate(t *testing.T) {
	tempDir := t.TempDir()

	mpDir := filepath.Join(tempDir, "mp1")
	mpRulesDir := filepath.Join(mpDir, "rules")
	createRuleFile(t, mpRulesDir, "rule.md", "# Rule Content")

	testData := map[string]registry.MarketplaceEntry{
		"mp1": {
			InstallLocation: mpDir,
			Source: registry.MarketplaceSource{
				Repo:   "owner/mp1",
				Source: "github",
			},
		},
	}
	createTestRegistry(t, tempDir, testData)

	resolver := registry.NewResolver(tempDir)
	mgr := NewManager(resolver)

	// First update
	result, err := mgr.CopyRules()
	require.NoError(t, err)
	assert.Equal(t, 1, len(result.CopiedFiles))

	copiedFile := filepath.Join(mgr.rulesDir, "mp1_rule.md")
	content, err := os.ReadFile(copiedFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "<!-- Marketplace: mp1 -->")

	// Second update - should NOT create backup because watermark exists
	result, err = mgr.CopyRules()
	require.NoError(t, err)
	assert.Equal(t, 1, len(result.CopiedFiles))

	entries, err := os.ReadDir(mgr.rulesDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "should only have the rule file, no backup")
}
