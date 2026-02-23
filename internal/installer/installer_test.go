package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform/mocks"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/software"
)

func TestNew(t *testing.T) {
	mockPlat := mocks.NewMockPlatform(t)
	mockArch := mocks.NewMockArchiver(t)
	p := printer.New(false, true, false)

	inst := New(mockPlat, mockArch, p)

	require.NotNil(t, inst)
	assert.Equal(t, mockPlat, inst.platform)
	assert.Equal(t, mockArch, inst.archiver)
	assert.Equal(t, p, inst.printer)
}

func TestPlatformSupportsMethod(t *testing.T) {
	tests := []struct {
		currentOS string
		expected  bool
		method    software.InstallMethod
		name      string
	}{
		{
			name:      "method supports current OS",
			method:    software.InstallMethod{Platforms: []string{"darwin", "linux"}},
			currentOS: "darwin",
			expected:  true,
		},
		{
			name:      "method does not support current OS",
			method:    software.InstallMethod{Platforms: []string{"linux"}},
			currentOS: "windows",
			expected:  false,
		},
		{
			name:      "empty platforms returns false",
			method:    software.InstallMethod{Platforms: []string{}},
			currentOS: "darwin",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlat := mocks.NewMockPlatform(t)
			mockArch := mocks.NewMockArchiver(t)
			p := printer.New(false, true, false)
			inst := New(mockPlat, mockArch, p)

			mockPlat.EXPECT().OS().Return(tt.currentOS)

			result := inst.platformSupportsMethod(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}
