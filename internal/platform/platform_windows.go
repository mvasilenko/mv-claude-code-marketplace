//go:build windows

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

	"golang.org/x/sys/windows/registry"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
)

type windowsPlatform struct {
	arch string
}

// New creates a new Windows platform instance
func New() Platform {
	return &windowsPlatform{
		arch: normalizeArch(runtime.GOARCH),
	}
}

// OS returns the operating system name
func (w *windowsPlatform) OS() string {
	return "windows"
}

// Arch returns the normalized architecture
func (w *windowsPlatform) Arch() string {
	return w.arch
}

// CommandExists checks if a command is available in PATH
func (w *windowsPlatform) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// CommandExistsInBinDir checks if a command exists in the bin directory
func (w *windowsPlatform) CommandExistsInBinDir(name string) bool {
	binaryName := name
	if !strings.HasSuffix(name, ".exe") {
		binaryName = name + ".exe"
	}
	binaryPath := filepath.Join(w.GetBinDir(), binaryName)
	_, err := os.Stat(binaryPath)
	return err == nil
}

// GetCommandVersion gets the version of a command
func (w *windowsPlatform) GetCommandVersion(name string) (string, error) {
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
func (w *windowsPlatform) GetBinDir() string {
	home := os.Getenv("USERPROFILE")
	return filepath.Join(home, ".local", "bin")
}

// GetConfigDir returns the OS-specific configuration directory for claudectl
func (w *windowsPlatform) GetConfigDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home := os.Getenv("USERPROFILE")
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	return filepath.Join(appData, "claudectl")
}

// GetLogsDir returns the OS-specific logs directory for claudectl
func (w *windowsPlatform) GetLogsDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home := os.Getenv("USERPROFILE")
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(localAppData, "claudectl", "logs")
}

// EnsureBinDir creates the bin directory if it doesn't exist
func (w *windowsPlatform) EnsureBinDir() error {
	binDir := w.GetBinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	return nil
}

// MakeExecutable is a no-op on Windows
func (w *windowsPlatform) MakeExecutable(_ string) error {
	return nil
}

// IsBinDirInPath checks if the bin directory is in PATH
func (w *windowsPlatform) IsBinDirInPath() bool {
	binDir := w.GetBinDir()
	path := os.Getenv("PATH")
	return strings.Contains(path, binDir)
}

// AddBinDirToPath adds the bin directory to PATH persistently via registry
func (w *windowsPlatform) AddBinDirToPath() error {
	binDir := w.GetBinDir()

	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close() //nolint:errcheck

	currentPath, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to read PATH from registry: %w", err)
	}

	if strings.Contains(currentPath, binDir) {
		return nil
	}

	var newPath string
	if currentPath == "" {
		newPath = binDir
	} else {
		newPath = currentPath + ";" + binDir
	}

	if err := key.SetStringValue("Path", newPath); err != nil {
		return fmt.Errorf("failed to update PATH in registry: %w", err)
	}

	_ = os.Setenv("PATH", os.Getenv("PATH")+";"+binDir)

	return nil
}

// DownloadFile downloads a file from a URL to a destination path
func (w *windowsPlatform) DownloadFile(url string, dest string) error {
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

// ExecuteScript executes a PowerShell script and returns its output
func (w *windowsPlatform) ExecuteScript(ctx context.Context, scriptPath string) (string, string, error) {
	log := logger.FromContext(ctx)

	shell := "pwsh"
	if _, err := exec.LookPath("pwsh"); err != nil {
		shell = "powershell"
	}

	log.Info("executing external script",
		"script_path", scriptPath,
		"shell", shell,
	)
	startTime := time.Now()

	cmd := exec.Command(shell, "-ExecutionPolicy", "Bypass", "-File", scriptPath)
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
		"shell", shell,
		"duration_ms", duration.Milliseconds(),
		"exit_code", exitCode,
		"stdout_length", len(stdout),
		"stderr_length", len(stderr),
	)

	if err != nil {
		log.Error("script execution failed",
			"error", err,
			"script_path", scriptPath,
			"shell", shell,
			"exit_code", exitCode,
		)
		return stdout, stderr, fmt.Errorf("failed to execute script %s: %w", scriptPath, err)
	}

	return stdout, stderr, nil
}

// HasHomebrew always returns false on Windows
func (w *windowsPlatform) HasHomebrew() bool {
	return false
}

// InstallWithHomebrew is not supported on Windows
func (w *windowsPlatform) InstallWithHomebrew(_ string) error {
	return fmt.Errorf("Homebrew is not available on Windows")
}

// HasKeyring always returns false on Windows
func (w *windowsPlatform) HasKeyring() bool {
	return false
}

// SetupKeyring is not supported on Windows
func (w *windowsPlatform) SetupKeyring() error {
	return fmt.Errorf("keyring setup not supported on Windows")
}

// normalizeArch normalizes architecture names
func normalizeArch(arch string) string {
	switch arch {
	case "amd64", "x86_64":
		return "amd64"
	case "arm64", "aarch64":
		return "arm64"
	case "386", "x86":
		return "386"
	default:
		return arch
	}
}
