package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAssetName(t *testing.T) {
	tests := []struct {
		arch     string
		expected string
		name     string
		osName   string
		pattern  string
		version  string
	}{
		{
			name:     "standard pattern with darwin",
			pattern:  "gh_{version}_{os}_{arch}",
			version:  "2.50.0",
			osName:   "darwin",
			arch:     "amd64",
			expected: "gh_2.50.0_macOS_amd64.tar.gz",
		},
		{
			name:     "standard pattern with linux",
			pattern:  "gh_{version}_{os}_{arch}",
			version:  "2.50.0",
			osName:   "linux",
			arch:     "arm64",
			expected: "gh_2.50.0_linux_arm64.tar.gz",
		},
	}

	// Skip Windows-specific tests on non-Windows since archive extension is runtime-dependent
	if runtime.GOOS != "windows" {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := buildAssetName(tt.pattern, tt.version, tt.osName, tt.arch)
				assert.Equal(t, tt.expected, result)
			})
		}
	}
}

func TestBuildAssetNameCustom(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("archive extension is platform-dependent")
	}

	tests := []struct {
		arch     string
		expected string
		name     string
		osName   string
		pattern  string
	}{
		{
			name:     "custom pattern no OS normalization",
			pattern:  "tool_{os}_{arch}",
			osName:   "darwin",
			arch:     "arm64",
			expected: "tool_darwin_arm64.tar.gz",
		},
		{
			name:     "custom pattern with linux",
			pattern:  "tool_{os}_{arch}",
			osName:   "linux",
			arch:     "amd64",
			expected: "tool_linux_amd64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAssetNameCustom(tt.pattern, tt.osName, tt.arch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindBinaryInDir(t *testing.T) {
	tests := []struct {
		binaryName string
		name       string
		setup      func(t *testing.T, dir string)
		wantErr    bool
	}{
		{
			name:       "finds binary in root",
			binaryName: "testbin",
			setup: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "testbin"), []byte("binary"), 0755)
				require.NoError(t, err)
			},
		},
		{
			name:       "finds binary in subdirectory",
			binaryName: "testbin",
			setup: func(t *testing.T, dir string) {
				subdir := filepath.Join(dir, "subdir")
				err := os.MkdirAll(subdir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(subdir, "testbin"), []byte("binary"), 0755)
				require.NoError(t, err)
			},
		},
		{
			name:       "binary not found returns error",
			binaryName: "nonexistent",
			setup:      func(t *testing.T, dir string) {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			result, err := findBinaryInDir(tmpDir, tt.binaryName)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, result, tt.binaryName)
		})
	}
}

func TestCopyFileToDestination(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, srcDir, dstDir string) (src, dst string)
		wantErr bool
	}{
		{
			name: "copies file successfully",
			setup: func(t *testing.T, srcDir, dstDir string) (string, string) {
				src := filepath.Join(srcDir, "source")
				dst := filepath.Join(dstDir, "dest")
				err := os.WriteFile(src, []byte("hello world"), 0644)
				require.NoError(t, err)
				return src, dst
			},
		},
		{
			name: "overwrites existing destination",
			setup: func(t *testing.T, srcDir, dstDir string) (string, string) {
				src := filepath.Join(srcDir, "source")
				dst := filepath.Join(dstDir, "dest")
				err := os.WriteFile(src, []byte("new content"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(dst, []byte("old content"), 0644)
				require.NoError(t, err)
				return src, dst
			},
		},
		{
			name: "source not found returns error",
			setup: func(t *testing.T, srcDir, dstDir string) (string, string) {
				return filepath.Join(srcDir, "nonexistent"), filepath.Join(dstDir, "dest")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcDir := t.TempDir()
			dstDir := t.TempDir()
			src, dst := tt.setup(t, srcDir, dstDir)

			err := copyFileToDestination(src, dst)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify content was copied
			srcContent, _ := os.ReadFile(src)
			dstContent, err := os.ReadFile(dst)
			require.NoError(t, err)
			assert.Equal(t, srcContent, dstContent)
		})
	}
}

func TestIsClaudeLockError(t *testing.T) {
	tests := []struct {
		expected bool
		name     string
		output   string
	}{
		{
			name:     "detects lock error message",
			output:   "Error: another process is currently installing Claude",
			expected: true,
		},
		{
			name:     "detects lock acquisition failure",
			output:   "Lock acquisition failed",
			expected: true,
		},
		{
			name:     "no lock error",
			output:   "Installation completed successfully",
			expected: false,
		},
		{
			name:     "empty output",
			output:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isClaudeLockError(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}
