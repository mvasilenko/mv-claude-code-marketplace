//go:build unix

package platform

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
)

type unixPlatform struct {
	arch string
	os   string
}

// New creates a new Unix platform instance
func New() Platform {
	return &unixPlatform{
		arch: normalizeArch(runtime.GOARCH),
		os:   runtime.GOOS,
	}
}

// OS returns the operating system name
func (u *unixPlatform) OS() string {
	return u.os
}

// Arch returns the normalized architecture
func (u *unixPlatform) Arch() string {
	return u.arch
}

// CommandExists checks if a command is available in PATH
func (u *unixPlatform) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// CommandExistsInBinDir checks if a command exists in the bin directory
func (u *unixPlatform) CommandExistsInBinDir(name string) bool {
	binaryPath := filepath.Join(u.GetBinDir(), name)
	info, err := os.Stat(binaryPath)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// GetCommandVersion gets the version of a command
func (u *unixPlatform) GetCommandVersion(name string) (string, error) {
	cmd := exec.Command(name, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %w", name, err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		return lines[0], nil
	}
	return strings.TrimSpace(string(output)), nil
}

// GetBinDir returns the user's local bin directory
func (u *unixPlatform) GetBinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

// GetConfigDir returns the OS-specific configuration directory for claudectl
func (u *unixPlatform) GetConfigDir() string {
	if u.os == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "claudectl")
	}
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "claudectl")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "claudectl")
}

// GetLogsDir returns the OS-specific logs directory for claudectl
func (u *unixPlatform) GetLogsDir() string {
	home, _ := os.UserHomeDir()
	if u.os == "darwin" {
		return filepath.Join(home, "Library", "Logs", "claudectl")
	}
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "claudectl", "logs")
	}
	return filepath.Join(home, ".local", "share", "claudectl", "logs")
}

// EnsureBinDir creates the bin directory if it doesn't exist
func (u *unixPlatform) EnsureBinDir() error {
	binDir := u.GetBinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	return nil
}

// MakeExecutable makes a file executable (chmod +x)
func (u *unixPlatform) MakeExecutable(path string) error {
	if err := os.Chmod(path, 0755); err != nil {
		return fmt.Errorf("failed to make %s executable: %w", path, err)
	}
	return nil
}

// IsBinDirInPath checks if the bin directory is in PATH
func (u *unixPlatform) IsBinDirInPath() bool {
	binDir := u.GetBinDir()
	path := os.Getenv("PATH")
	return strings.Contains(path, binDir)
}

// AddBinDirToPath adds the bin directory to PATH persistently
func (u *unixPlatform) AddBinDirToPath() error {
	binDir := u.GetBinDir()

	var shellProfile string
	shell := os.Getenv("SHELL")

	if strings.Contains(shell, "zsh") {
		home, _ := os.UserHomeDir()
		shellProfile = filepath.Join(home, ".zshrc")
	} else {
		home, _ := os.UserHomeDir()
		shellProfile = filepath.Join(home, ".bashrc")
	}

	if _, err := os.Stat(shellProfile); os.IsNotExist(err) {
		if err := os.WriteFile(shellProfile, []byte{}, 0644); err != nil {
			return fmt.Errorf("failed to create shell profile: %w", err)
		}
	}

	content, err := os.ReadFile(shellProfile)
	if err != nil {
		return fmt.Errorf("failed to read shell profile: %w", err)
	}

	if strings.Contains(string(content), binDir) {
		return nil
	}

	pathExport := fmt.Sprintf("\n# Added by claudectl\nexport PATH=\"%s:$PATH\"\n", binDir)
	f, err := os.OpenFile(shellProfile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open shell profile for writing: %w", err)
	}
	defer f.Close() //nolint:errcheck

	if _, err := f.WriteString(pathExport); err != nil {
		return fmt.Errorf("failed to write to shell profile: %w", err)
	}

	currentPath := os.Getenv("PATH")
	if !strings.Contains(currentPath, binDir) {
		_ = os.Setenv("PATH", binDir+":"+currentPath)
	}

	return nil
}

// DownloadFile downloads a file from a URL to a destination path
func (u *unixPlatform) DownloadFile(url string, dest string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download from %s: HTTP %d", url, resp.StatusCode)
	}

	outFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", dest, err)
	}
	defer outFile.Close() //nolint:errcheck

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write to %s: %w", dest, err)
	}

	return nil
}

// ExecuteScript executes a shell script using bash and returns its output
func (u *unixPlatform) ExecuteScript(ctx context.Context, scriptPath string) (string, string, error) {
	log := logger.FromContext(ctx)

	log.Info("executing external script",
		"script_path", scriptPath,
		"shell", "bash",
	)
	startTime := time.Now()

	if err := u.MakeExecutable(scriptPath); err != nil {
		log.Error("failed to make script executable",
			"error", err,
			"script_path", scriptPath,
		)
		return "", "", err
	}

	cmd := exec.Command("bash", scriptPath)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	duration := time.Since(startTime)
	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	log.Info("external script completed",
		"script_path", scriptPath,
		"duration_ms", duration.Milliseconds(),
		"exit_code", exitCode,
		"stdout_length", len(stdout),
		"stderr_length", len(stderr),
	)

	if err != nil {
		log.Error("script execution failed",
			"error", err,
			"script_path", scriptPath,
			"exit_code", exitCode,
		)
		return stdout, stderr, fmt.Errorf("failed to execute script %s: %w", scriptPath, err)
	}

	return stdout, stderr, nil
}

// HasHomebrew checks if Homebrew is installed
func (u *unixPlatform) HasHomebrew() bool {
	return u.CommandExists("brew")
}

// InstallWithHomebrew installs a package using Homebrew
func (u *unixPlatform) InstallWithHomebrew(packageName string) error {
	if !u.HasHomebrew() {
		return fmt.Errorf("homebrew is not installed")
	}

	cmd := exec.Command("brew", "install", packageName)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s with Homebrew: %w", packageName, err)
	}

	return nil
}

// HasKeyring checks if a keyring daemon is available (Linux only)
func (u *unixPlatform) HasKeyring() bool {
	if u.os != "linux" {
		return false
	}
	return u.CommandExists("gnome-keyring-daemon") || u.CommandExists("secret-tool")
}

// SetupKeyring sets up keyring on Linux
func (u *unixPlatform) SetupKeyring() error {
	if u.os != "linux" {
		return fmt.Errorf("keyring setup only applicable on Linux")
	}
	return fmt.Errorf("automatic keyring setup not yet implemented")
}

// normalizeArch normalizes architecture names
func normalizeArch(arch string) string {
	switch arch {
	case "amd64", "x86_64":
		return "amd64"
	case "arm64", "aarch64":
		return "arm64"
	default:
		return arch
	}
}
