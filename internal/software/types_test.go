package software

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBinaryName(t *testing.T) {
	tests := []struct {
		expected string
		goos     string
		name     string
		sw       Software
	}{
		{
			name:     "unix returns binaryName",
			sw:       Software{BinaryName: "gh", BinaryNameWindows: "gh.exe"},
			goos:     "darwin",
			expected: "gh",
		},
		{
			name:     "linux returns binaryName",
			sw:       Software{BinaryName: "claude", BinaryNameWindows: "claude.exe"},
			goos:     "linux",
			expected: "claude",
		},
		{
			name:     "windows returns binaryNameWindows",
			sw:       Software{BinaryName: "gh", BinaryNameWindows: "gh.exe"},
			goos:     "windows",
			expected: "gh.exe",
		},
		{
			name:     "windows falls back to binaryName when windows name empty",
			sw:       Software{BinaryName: "tool"},
			goos:     "windows",
			expected: "tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sw.GetBinaryName(tt.goos)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportsCurrentPlatform(t *testing.T) {
	tests := []struct {
		expected bool
		goos     string
		name     string
		sw       Software
	}{
		{
			name: "supported platform returns true",
			sw: Software{
				InstallMethods: []InstallMethod{
					{Platforms: []string{"darwin", "linux"}},
				},
			},
			goos:     "darwin",
			expected: true,
		},
		{
			name: "unsupported platform returns false",
			sw: Software{
				InstallMethods: []InstallMethod{
					{Platforms: []string{"linux"}},
				},
			},
			goos:     "windows",
			expected: false,
		},
		{
			name: "multiple methods with one supporting returns true",
			sw: Software{
				InstallMethods: []InstallMethod{
					{Platforms: []string{"darwin"}},
					{Platforms: []string{"linux", "windows"}},
				},
			},
			goos:     "windows",
			expected: true,
		},
		{
			name:     "no install methods returns false",
			sw:       Software{},
			goos:     "darwin",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sw.SupportsCurrentPlatform(tt.goos)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSoftware(t *testing.T) {
	cfg := &Config{
		Software: []Software{
			{ID: "claude", DisplayName: "Claude Code CLI"},
			{ID: "gh", DisplayName: "GitHub CLI"},
		},
	}

	tests := []struct {
		expectedName string
		id           string
		name         string
		wantErr      bool
	}{
		{
			name:         "existing software returns correctly",
			id:           "claude",
			expectedName: "Claude Code CLI",
		},
		{
			name:         "second software returns correctly",
			id:           "gh",
			expectedName: "GitHub CLI",
		},
		{
			name:    "non-existent software returns error",
			id:      "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw, err := cfg.GetSoftware(tt.id)

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrSoftwareNotFound)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedName, sw.DisplayName)
		})
	}
}

func TestGetAllIDs(t *testing.T) {
	tests := []struct {
		cfg      *Config
		expected []string
		name     string
	}{
		{
			name: "returns all IDs",
			cfg: &Config{
				Software: []Software{
					{ID: "claude"},
					{ID: "gh"},
				},
			},
			expected: []string{"claude", "gh"},
		},
		{
			name:     "empty config returns empty slice",
			cfg:      &Config{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.GetAllIDs()
			assert.Equal(t, tt.expected, result)
		})
	}
}
