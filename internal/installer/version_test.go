package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		expected Version
		input    string
		name     string
		wantErr  bool
	}{
		{
			name:     "standard version",
			input:    "1.2.3",
			expected: Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "version with v prefix",
			input:    "v1.2.3",
			expected: Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "version with pre-release",
			input:    "1.2.3-dirty",
			expected: Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "version with build metadata",
			input:    "1.2.3+build.123",
			expected: Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:     "version with alpha pre-release",
			input:    "v2.0.0-alpha.1",
			expected: Version{Major: 2, Minor: 0, Patch: 0},
		},
		{
			name:    "invalid format too few parts",
			input:   "1.2",
			wantErr: true,
		},
		{
			name:    "invalid format too many parts",
			input:   "1.2.3.4",
			wantErr: true,
		},
		{
			name:    "non-numeric major",
			input:   "abc.2.3",
			wantErr: true,
		},
		{
			name:    "non-numeric minor",
			input:   "1.abc.3",
			wantErr: true,
		},
		{
			name:    "non-numeric patch",
			input:   "1.2.abc",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseVersion(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		a        Version
		b        Version
		expected int
		name     string
	}{
		{
			name:     "equal versions",
			a:        Version{Major: 1, Minor: 2, Patch: 3},
			b:        Version{Major: 1, Minor: 2, Patch: 3},
			expected: 0,
		},
		{
			name:     "major version greater",
			a:        Version{Major: 2, Minor: 0, Patch: 0},
			b:        Version{Major: 1, Minor: 9, Patch: 9},
			expected: 1,
		},
		{
			name:     "major version less",
			a:        Version{Major: 1, Minor: 0, Patch: 0},
			b:        Version{Major: 2, Minor: 0, Patch: 0},
			expected: -1,
		},
		{
			name:     "minor version greater",
			a:        Version{Major: 1, Minor: 3, Patch: 0},
			b:        Version{Major: 1, Minor: 2, Patch: 9},
			expected: 1,
		},
		{
			name:     "minor version less",
			a:        Version{Major: 1, Minor: 1, Patch: 0},
			b:        Version{Major: 1, Minor: 2, Patch: 0},
			expected: -1,
		},
		{
			name:     "patch version greater",
			a:        Version{Major: 1, Minor: 2, Patch: 4},
			b:        Version{Major: 1, Minor: 2, Patch: 3},
			expected: 1,
		},
		{
			name:     "patch version less",
			a:        Version{Major: 1, Minor: 2, Patch: 2},
			b:        Version{Major: 1, Minor: 2, Patch: 3},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Compare(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersion_IsNewer(t *testing.T) {
	tests := []struct {
		a        Version
		b        Version
		expected bool
		name     string
	}{
		{
			name:     "newer version returns true",
			a:        Version{Major: 2, Minor: 0, Patch: 0},
			b:        Version{Major: 1, Minor: 0, Patch: 0},
			expected: true,
		},
		{
			name:     "older version returns false",
			a:        Version{Major: 1, Minor: 0, Patch: 0},
			b:        Version{Major: 2, Minor: 0, Patch: 0},
			expected: false,
		},
		{
			name:     "equal version returns false",
			a:        Version{Major: 1, Minor: 2, Patch: 3},
			b:        Version{Major: 1, Minor: 2, Patch: 3},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.IsNewer(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		expected string
		name     string
		version  Version
	}{
		{
			name:     "standard version",
			version:  Version{Major: 1, Minor: 2, Patch: 3},
			expected: "1.2.3",
		},
		{
			name:     "zero version",
			version:  Version{Major: 0, Minor: 0, Patch: 0},
			expected: "0.0.0",
		},
		{
			name:     "large numbers",
			version:  Version{Major: 100, Minor: 200, Patch: 300},
			expected: "100.200.300",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
