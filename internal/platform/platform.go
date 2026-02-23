package platform

import "context"

// Platform provides OS-specific operations for installing tools
type Platform interface {
	// System information
	Arch() string // "amd64", "arm64", "386"
	OS() string   // "darwin", "linux", "windows"

	// Command operations
	CommandExists(name string) bool
	CommandExistsInBinDir(name string) bool
	GetCommandVersion(name string) (string, error)

	// File system operations
	EnsureBinDir() error
	GetBinDir() string
	GetConfigDir() string
	GetLogsDir() string
	MakeExecutable(path string) error

	// PATH management
	AddBinDirToPath() error
	IsBinDirInPath() bool

	// Download and execute
	DownloadFile(url string, dest string) error
	ExecuteScript(ctx context.Context, scriptPath string) (stdout, stderr string, err error)

	// Package manager operations (macOS only)
	HasHomebrew() bool
	InstallWithHomebrew(packageName string) error

	// Keyring operations (Linux only)
	HasKeyring() bool
	SetupKeyring() error
}

// Archiver handles archive extraction
type Archiver interface {
	Extract(archivePath string, destDir string) error
	GetExpectedFormat() string // "tar.gz" or "zip"
}

// Factory functions New() and NewArchiver() are implemented by OS-specific files:
// - platform_unix.go (for macOS and Linux)
// - platform_windows.go (for Windows)
