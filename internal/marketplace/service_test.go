package marketplace

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	svc := NewService("/cache/path")
	assert.Equal(t, "/cache/path", svc.cachePath)
}

func TestGetMarketplacePath(t *testing.T) {
	svc := NewService("/cache")
	assert.Equal(t, filepath.Join("/cache", "my-marketplace"), svc.GetMarketplacePath("my-marketplace"))
}

func TestIsCached(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewService(tempDir)

	// Not cached
	assert.False(t, svc.IsCached("nonexistent"))

	// Cached
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "exists"), 0755))
	assert.True(t, svc.IsCached("exists"))
}

func TestIsRegisteredWithClaude(t *testing.T) {
	tests := []struct {
		expected bool
		name     string
		registry map[string]any
		target   string
	}{
		{
			name:     "not registered - no file",
			registry: nil,
			target:   "my-marketplace",
			expected: false,
		},
		{
			name:     "not registered - empty registry",
			registry: map[string]any{},
			target:   "my-marketplace",
			expected: false,
		},
		{
			name: "registered",
			registry: map[string]any{
				"my-marketplace": map[string]any{"installLocation": "/some/path"},
			},
			target:   "my-marketplace",
			expected: true,
		},
		{
			name: "different marketplace registered",
			registry: map[string]any{
				"other-marketplace": map[string]any{"installLocation": "/some/path"},
			},
			target:   "my-marketplace",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			svc := NewService(filepath.Join(tempDir, "cache"))

			if tt.registry != nil {
				registryDir := filepath.Join(tempDir, "plugins")
				require.NoError(t, os.MkdirAll(registryDir, 0755))

				data, err := json.Marshal(tt.registry)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(registryDir, "known_marketplaces.json"), data, 0644))
			}

			result := svc.IsRegisteredWithClaude(tt.target, tempDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeletePluginCache(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, claudeDir string)
		verifyGon bool
	}{
		{
			name:      "cache does not exist",
			setup:     func(t *testing.T, claudeDir string) {},
			verifyGon: true,
		},
		{
			name: "cache exists and is deleted",
			setup: func(t *testing.T, claudeDir string) {
				cachePath := filepath.Join(claudeDir, "plugins", "cache", "test-mp")
				require.NoError(t, os.MkdirAll(cachePath, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(cachePath, "file.txt"), []byte("data"), 0644))
			},
			verifyGon: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			svc := NewService(filepath.Join(tempDir, "svc-cache"))

			tt.setup(t, tempDir)

			// Should not panic
			svc.DeletePluginCache(context.TODO(), tempDir, "test-mp")

			if tt.verifyGon {
				cachePath := filepath.Join(tempDir, "plugins", "cache", "test-mp")
				_, err := os.Stat(cachePath)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestGetDisabledPluginsFromSource(t *testing.T) {
	tests := []struct {
		expected map[string]bool
		name     string
		settings map[string]any
	}{
		{
			name:     "no base_settings.json",
			settings: nil,
			expected: nil,
		},
		{
			name: "no enabledPlugins key",
			settings: map[string]any{
				"someKey": "value",
			},
			expected: nil,
		},
		{
			name: "with disabled plugins",
			settings: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp": true,
					"plugin-b@mp": false,
					"plugin-c@mp": true,
					"plugin-d@mp": false,
				},
			},
			expected: map[string]bool{
				"plugin-b@mp": true,
				"plugin-d@mp": true,
			},
		},
		{
			name: "all enabled",
			settings: map[string]any{
				"enabledPlugins": map[string]any{
					"plugin-a@mp": true,
				},
			},
			expected: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			if tt.settings != nil {
				dir := filepath.Join(tempDir, "internal", "cmd")
				require.NoError(t, os.MkdirAll(dir, 0755))

				data, err := json.Marshal(tt.settings)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "base_settings.json"), data, 0644))
			}

			result := getDisabledPluginsFromSource(tempDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiscoverExtPlugins(t *testing.T) {
	tests := []struct {
		expected   []string
		name       string
		namespaces []string
		setup      func(t *testing.T, dir string)
	}{
		{
			name:       "no ext directory",
			namespaces: []string{"team1"},
			setup:      func(t *testing.T, dir string) {},
			expected:   nil,
		},
		{
			name:       "with ext plugins",
			namespaces: []string{"team1"},
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins-ext", "team1", "pluginA"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins-ext", "team1", "pluginB"), 0755))
			},
			expected: []string{"team1.pluginA", "team1.pluginB"},
		},
		{
			name:       "multiple namespaces",
			namespaces: []string{"ns1", "ns2"},
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins-ext", "ns1", "foo"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "plugins-ext", "ns2", "bar"), 0755))
			},
			expected: []string{"ns1.foo", "ns2.bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.setup(t, tempDir)

			result := discoverExtPlugins(tempDir, tt.namespaces)
			assert.Equal(t, tt.expected, result)
		})
	}
}
