package platform

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	p := New()
	require.NotNil(t, p)
	assert.Equal(t, runtime.GOOS, p.OS())
	assert.Contains(t, []string{"amd64", "arm64", "386"}, p.Arch())
}

func TestCommandExists(t *testing.T) {
	p := New()

	tests := []struct {
		command string
		exists  bool
		name    string
	}{
		{name: "go exists", command: "go", exists: true},
		{name: "nonexistent command", command: "definitely-not-a-real-command-xyz", exists: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.exists, p.CommandExists(tt.command))
		})
	}
}

func TestGetBinDir(t *testing.T) {
	p := New()
	binDir := p.GetBinDir()
	assert.Contains(t, binDir, ".local")
	assert.Contains(t, binDir, "bin")
}

func TestGetConfigDir(t *testing.T) {
	p := New()
	configDir := p.GetConfigDir()
	assert.Contains(t, configDir, "claudectl")
}

func TestGetLogsDir(t *testing.T) {
	p := New()
	logsDir := p.GetLogsDir()
	assert.Contains(t, logsDir, "claudectl")
}

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		expected string
		input    string
		name     string
	}{
		{name: "amd64 passes through", input: "amd64", expected: "amd64"},
		{name: "x86_64 normalizes to amd64", input: "x86_64", expected: "amd64"},
		{name: "arm64 passes through", input: "arm64", expected: "arm64"},
		{name: "aarch64 normalizes to arm64", input: "aarch64", expected: "arm64"},
		{name: "unknown passes through", input: "riscv64", expected: "riscv64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeArch(tt.input))
		})
	}
}
