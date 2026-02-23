package software

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPathResolver implements registry.PathResolver for testing
type mockPathResolver struct {
	softwareConfigPath string
}

func (m *mockPathResolver) BaseSettingsPath() string                       { return "" }
func (m *mockPathResolver) BaseSettingsPathForMarketplace(_ string) string { return "" }
func (m *mockPathResolver) ClaudeDir() string                              { return "" }
func (m *mockPathResolver) RulesPath() string                              { return "" }
func (m *mockPathResolver) SoftwareConfigPath() string                     { return m.softwareConfigPath }

func TestLoader_Load(t *testing.T) {
	tests := []struct {
		fileContent string
		name        string
		expectCount int
		fileExists  bool
		wantErr     bool
	}{
		{
			name:        "file not found uses embedded fallback",
			fileExists:  false,
			expectCount: 2, // claude and gh from embedded config
		},
		{
			name:       "valid file loads successfully",
			fileExists: true,
			fileContent: `{
				"version": "1.0",
				"software": [
					{
						"id": "test-tool",
						"displayName": "Test Tool",
						"binaryName": "test",
						"category": "required",
						"priority": 1,
						"verifyCommand": "test --version",
						"installMethods": []
					}
				]
			}`,
			expectCount: 1,
		},
		{
			name:        "invalid JSON returns error",
			fileExists:  true,
			fileContent: `{invalid json`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "software_config.json")

			if tt.fileExists {
				err := os.WriteFile(configPath, []byte(tt.fileContent), 0644)
				require.NoError(t, err)
			}

			resolver := &mockPathResolver{softwareConfigPath: configPath}
			loader := NewLoader(resolver)

			cfg, err := loader.Load()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Len(t, cfg.Software, tt.expectCount)
		})
	}
}

func TestLoader_LoadEmbedded(t *testing.T) {
	resolver := &mockPathResolver{softwareConfigPath: "/nonexistent/path"}
	loader := NewLoader(resolver)

	cfg := loader.LoadEmbedded()

	require.NotNil(t, cfg)
	assert.Len(t, cfg.Software, 2)
	assert.Equal(t, "claude", cfg.Software[0].ID)
	assert.Equal(t, "gh", cfg.Software[1].ID)
}
