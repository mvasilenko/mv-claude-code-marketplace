package fetcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		expected string
		input    string
		name     string
	}{
		{
			expected: "https://github.com/owner/repo.git",
			input:    "owner/repo",
			name:     "owner/repo format",
		},
		{
			expected: "git@github.com:owner/repo.git",
			input:    "git@github.com:owner/repo.git",
			name:     "SSH URL passthrough",
		},
		{
			expected: "https://github.com/owner/repo.git",
			input:    "https://github.com/owner/repo.git",
			name:     "HTTPS URL passthrough",
		},
		{
			expected: "https://github.com/owner/repo",
			input:    "https://github.com/owner/repo",
			name:     "HTTPS without .git passthrough",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeRepoURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeRepoURLWithProtocol(t *testing.T) {
	tests := []struct {
		expected string
		input    string
		name     string
		protocol string
	}{
		{
			expected: "git@github.com:owner/repo.git",
			input:    "owner/repo",
			name:     "owner/repo with ssh protocol",
			protocol: "ssh",
		},
		{
			expected: "https://github.com/owner/repo.git",
			input:    "owner/repo",
			name:     "owner/repo with https protocol",
			protocol: "https",
		},
		{
			expected: "https://github.com/owner/repo.git",
			input:    "owner/repo",
			name:     "owner/repo with empty protocol defaults to https",
			protocol: "",
		},
		{
			expected: "git@github.com:owner/repo.git",
			input:    "git@github.com:owner/repo.git",
			name:     "SSH URL passthrough ignores protocol",
			protocol: "https",
		},
		{
			expected: "https://github.com/owner/repo.git",
			input:    "https://github.com/owner/repo.git",
			name:     "HTTPS URL passthrough ignores protocol",
			protocol: "ssh",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeRepoURLWithProtocol(tt.input, tt.protocol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseRepository(t *testing.T) {
	tests := []struct {
		expectedOwner string
		expectedRepo  string
		input         string
		name          string
		expectError   bool
	}{
		{
			expectedOwner: "owner",
			expectedRepo:  "repo",
			input:         "owner/repo",
			name:          "valid owner/repo",
		},
		{
			expectedOwner: "owner",
			expectedRepo:  "repo",
			input:         " owner / repo ",
			name:          "valid with spaces",
		},
		{
			input:       "ownerrepo",
			name:        "missing slash",
			expectError: true,
		},
		{
			input:       "owner/repo/extra",
			name:        "too many slashes",
			expectError: true,
		},
		{
			input:       "/repo",
			name:        "empty owner",
			expectError: true,
		},
		{
			input:       "owner/",
			name:        "empty repo",
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRepository(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOwner, owner)
				assert.Equal(t, tt.expectedRepo, repo)
			}
		})
	}
}

func TestIsLocalPath(t *testing.T) {
	tests := []struct {
		expected bool
		input    string
		name     string
	}{
		{expected: true, input: "/path/to/marketplace", name: "absolute path"},
		{expected: true, input: "./marketplace", name: "relative dot"},
		{expected: true, input: "../marketplace", name: "relative parent"},
		{expected: true, input: "~/marketplace", name: "tilde expansion"},
		{expected: false, input: "owner/repo", name: "simple owner/repo"},
		{expected: false, input: "mvasilenko/mv-claude-code-marketplace", name: "owner/repo with org"},
		{expected: true, input: "org.name/repo", name: "dots in owner treated as path"},
		{expected: true, input: "path/to/somewhere/deep", name: "multi-level path"},
		{expected: true, input: "/", name: "root path"},
		{expected: true, input: ".", name: "current directory"},
		{expected: true, input: "..", name: "parent directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLocalPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFetchMarketplaceFromPath(t *testing.T) {
	t.Run("reads from .claude-plugin directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, ".claude-plugin")
		require.NoError(t, os.MkdirAll(pluginDir, 0755))

		marketplaceJSON := `{
			"name": "test-marketplace",
			"version": "1.0.0",
			"plugins": [
				{"name": "plugin-a", "version": "0.1.0"}
			]
		}`
		require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "marketplace.json"), []byte(marketplaceJSON), 0644))

		mp, err := FetchMarketplaceFromPath(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "test-marketplace", mp.Name)
		assert.Equal(t, "1.0.0", mp.Version)
		assert.Len(t, mp.Plugins, 1)
		assert.Equal(t, "plugin-a", mp.Plugins[0].Name)
	})

	t.Run("reads from root directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		marketplaceJSON := `{
			"name": "root-marketplace",
			"version": "2.0.0",
			"plugins": []
		}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "marketplace.json"), []byte(marketplaceJSON), 0644))

		mp, err := FetchMarketplaceFromPath(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "root-marketplace", mp.Name)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := FetchMarketplaceFromPath(tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "marketplace.json not found")
	})

	t.Run("validates required fields", func(t *testing.T) {
		tmpDir := t.TempDir()

		marketplaceJSON := `{"plugins": []}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "marketplace.json"), []byte(marketplaceJSON), 0644))

		_, err := FetchMarketplaceFromPath(tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required field: name")
	})
}
