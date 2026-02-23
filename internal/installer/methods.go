package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/software"
)

// installViaHomebrew installs software using Homebrew package manager
func (i *Installer) installViaHomebrew(ctx context.Context, method software.InstallMethod, sw *software.Software) error {
	log := logger.FromContext(ctx)

	if !i.platform.HasHomebrew() {
		return fmt.Errorf("homebrew not available")
	}

	log.Info("installing via Homebrew", "package", method.Package, "software", sw.ID)
	i.printer.Info(fmt.Sprintf("Installing via Homebrew: %s", method.Package))

	if err := i.platform.InstallWithHomebrew(method.Package); err != nil {
		return fmt.Errorf("brew install %s failed: %w", method.Package, err)
	}

	return nil
}

// installViaScript downloads and executes an installer script
func (i *Installer) installViaScript(ctx context.Context, method software.InstallMethod, sw *software.Software) (string, string, error) {
	log := logger.FromContext(ctx)
	log.Debug("starting installer script installation", "software", sw.ID)

	// Determine script URL based on OS
	scriptURL := method.URLUnix
	scriptExt := ".sh"
	if i.platform.OS() == "windows" {
		scriptURL = method.URLWindows
		scriptExt = ".ps1"
	}

	if scriptURL == "" {
		log.Debug("no script URL for platform", "platform", i.platform.OS())
		return "", "", fmt.Errorf("no script URL for platform: %s", i.platform.OS())
	}

	// Download script to temp file
	scriptPath := filepath.Join(os.TempDir(), sw.ID+"-install"+scriptExt)
	defer os.Remove(scriptPath) //nolint:errcheck // cleanup

	i.printer.Info("Downloading installer script")
	log.Info("downloading installer script", "url", scriptURL, "destination", scriptPath)
	if err := i.platform.DownloadFile(scriptURL, scriptPath); err != nil {
		log.Error("failed to download installer script", "error", err, "url", scriptURL)
		return "", "", fmt.Errorf("failed to download installer script: %w", err)
	}
	log.Debug("installer script downloaded successfully")

	// Execute script with retry on lock failure
	const maxRetries = 2
	var err error
	var stderr, stdout string
	var allOutput string

	for attempt := 0; attempt < maxRetries; attempt++ {
		i.printer.Info("Executing installer script")
		log.Info("executing installer script", "attempt", attempt+1, "maxRetries", maxRetries, "scriptPath", scriptPath)
		stdout, stderr, err = i.platform.ExecuteScript(ctx, scriptPath)

		if stdout != "" {
			allOutput += stdout
		}
		if stderr != "" {
			if allOutput != "" {
				allOutput += "\n"
			}
			allOutput += stderr
		}

		if err == nil {
			log.Info("installer script executed successfully")
			break
		}

		// Check for Claude lock error and retry after cleanup
		combined := stdout + stderr
		if attempt < maxRetries-1 && isClaudeLockError(combined) {
			log.Warn("claude lock error detected, retrying after cleanup", "attempt", attempt+1)
			i.printer.Warning("Clearing stale lock files and retrying")
			clearClaudeLocks()
			continue
		}

		log.Error("installer script execution failed", "error", err, "attempt", attempt+1, "stdout", stdout, "stderr", stderr)
	}

	stdout = allOutput
	stderr = ""

	if err != nil {
		log.Error("all installer script execution attempts failed", "error", err, "maxRetries", maxRetries)
		return stdout, stderr, fmt.Errorf("failed to execute installer script: %w", err)
	}

	log.Debug("installer script installation completed successfully")
	return stdout, stderr, nil
}

// installViaGitHubRelease installs software from GitHub releases
func (i *Installer) installViaGitHubRelease(ctx context.Context, method software.InstallMethod, sw *software.Software) error {
	log := logger.FromContext(ctx)
	log.Info("starting github_release install",
		"software", sw.ID,
		"repo", method.Repo,
		"assetPattern", method.AssetPattern)

	// Fetch latest release info
	i.printer.Info("Fetching latest release information")
	version, downloadURL, assetName, err := i.fetchGitHubReleaseAsset(ctx, method)
	if err != nil {
		log.Error("failed to fetch GitHub release", "error", err, "repo", method.Repo)
		return fmt.Errorf("failed to fetch GitHub release: %w", err)
	}
	log.Debug("fetched GitHub release", "version", version)

	// Download archive
	archiveExt := i.archiver.GetExpectedFormat()
	if archiveExt == "zip" {
		archiveExt = ".zip"
	} else {
		archiveExt = ".tar.gz"
	}

	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s%s", sw.ID, version, archiveExt))
	defer os.Remove(archivePath) //nolint:errcheck // cleanup

	i.printer.Info(fmt.Sprintf("Downloading %s %s", sw.DisplayName, version))

	// Use GitHubClient to download (tries gh CLI first, then HTTP)
	parts := strings.Split(method.Repo, "/")
	if len(parts) != 2 {
		log.Error("invalid repo format", "repo", method.Repo)
		return fmt.Errorf("invalid repo format: %s (expected owner/repo)", method.Repo)
	}

	client := NewGitHubClient(parts[0], parts[1])
	log.Info("downloading release asset", "repo", method.Repo, "version", version, "asset", assetName)
	if err := client.DownloadAsset(ctx, version, assetName, downloadURL, archivePath, i.platform.DownloadFile); err != nil {
		log.Error("failed to download release asset", "error", err, "repo", method.Repo, "asset", assetName)
		return fmt.Errorf("failed to download release asset: %w", err)
	}
	log.Debug("release asset downloaded successfully", "archivePath", archivePath)

	// Extract archive
	extractDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s_extract_%s", sw.ID, version))
	defer os.RemoveAll(extractDir) //nolint:errcheck // cleanup

	i.printer.Info("Extracting archive")
	if err := i.archiver.Extract(archivePath, extractDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Find binary in extracted files
	binaryName := sw.GetBinaryName(i.platform.OS())
	searchName := binaryName
	if i.platform.OS() == "windows" && !strings.HasSuffix(binaryName, ".exe") {
		searchName = binaryName + ".exe"
	}
	binaryPath, err := findBinaryInDir(extractDir, searchName)
	if err != nil {
		return fmt.Errorf("failed to find binary %s: %w", searchName, err)
	}

	// Copy binary to bin directory
	i.printer.Info(fmt.Sprintf("Installing to %s", i.platform.GetBinDir()))
	destPath := filepath.Join(i.platform.GetBinDir(), searchName)
	if err := copyFileToDestination(binaryPath, destPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	if err := i.platform.MakeExecutable(destPath); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	return nil
}

// fetchGitHubReleaseAsset fetches the appropriate asset for the current platform
func (i *Installer) fetchGitHubReleaseAsset(ctx context.Context, method software.InstallMethod) (version string, downloadURL string, assetName string, err error) {
	parts := strings.Split(method.Repo, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid repo format: %s (expected owner/repo)", method.Repo)
	}

	client := NewGitHubClient(parts[0], parts[1])
	release, err := client.GetLatestRelease(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to fetch GitHub release: %w", err)
	}

	version = release.version.String()

	if method.AssetPattern == "" {
		method.AssetPattern = "{id}_{version}_{os}_{arch}"
	}

	if strings.Contains(method.AssetPattern, "{version}") {
		assetName = buildAssetName(method.AssetPattern, version, i.platform.OS(), i.platform.Arch())
	} else {
		assetName = buildAssetNameCustom(method.AssetPattern, i.platform.OS(), i.platform.Arch())
	}

	for _, asset := range release.assets {
		if asset.name == assetName {
			return version, asset.browserDownloadURL, assetName, nil
		}
	}

	var availableNames []string
	for _, asset := range release.assets {
		availableNames = append(availableNames, asset.name)
	}

	return "", "", "", fmt.Errorf("no matching asset found for %s (wanted: %s, available: %v)",
		assetName, assetName, availableNames)
}

// buildAssetName constructs the asset filename from pattern
func buildAssetName(pattern, version, osName, arch string) string {
	displayOS := osName
	if displayOS == "darwin" {
		displayOS = "macOS"
	}

	archiveExt := ".tar.gz"
	if osName == "windows" {
		archiveExt = ".zip"
	}

	result := pattern
	result = strings.ReplaceAll(result, "{version}", version)
	result = strings.ReplaceAll(result, "{os}", displayOS)
	result = strings.ReplaceAll(result, "{arch}", arch)

	return result + archiveExt
}

// buildAssetNameCustom constructs the asset filename without OS normalization
func buildAssetNameCustom(pattern, osName, arch string) string {
	archiveExt := ".tar.gz"
	if osName == "windows" {
		archiveExt = ".zip"
	}

	result := pattern
	result = strings.ReplaceAll(result, "{os}", osName)
	result = strings.ReplaceAll(result, "{arch}", arch)

	return result + archiveExt
}

// findBinaryInDir recursively searches for a binary file
func findBinaryInDir(dir string, name string) (string, error) {
	var result string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == name {
			result = path
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if result == "" {
		return "", fmt.Errorf("binary %s not found in %s", name, dir)
	}
	return result, nil
}

// copyFileToDestination copies a file from src to dst
func copyFileToDestination(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close() //nolint:errcheck // read-only

	// Handle existing destination file
	if _, err := os.Stat(dst); err == nil {
		if runtime.GOOS == "windows" {
			oldPath := dst + ".old"
			_ = os.Remove(oldPath) //nolint:errcheck // best-effort cleanup
			if err := os.Rename(dst, oldPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to rename existing file: %w", err)
			}
		} else {
			if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove existing file: %w", err)
			}
		}
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close() //nolint:errcheck // best-effort close

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

// isClaudeLockError checks if the output indicates a Claude lock issue
func isClaudeLockError(output string) bool {
	return strings.Contains(output, "another process is currently installing") ||
		strings.Contains(output, "Lock acquisition failed")
}

// clearClaudeLocks removes Claude lock files/directories that may be corrupted
func clearClaudeLocks() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Clear lock files in ~/.local/state/claude/locks/
	locksDir := filepath.Join(home, ".local", "state", "claude", "locks")
	if entries, err := os.ReadDir(locksDir); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".lock") {
				path := filepath.Join(locksDir, entry.Name())
				os.RemoveAll(path) //nolint:errcheck // best-effort cleanup
			}
		}
	}

	// Clear empty/corrupted version files in ~/.local/share/claude/versions/
	versionsDir := filepath.Join(home, ".local", "share", "claude", "versions")
	if entries, err := os.ReadDir(versionsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(versionsDir, entry.Name())
			if info, err := entry.Info(); err == nil && info.Size() == 0 {
				os.Remove(path) //nolint:errcheck // best-effort cleanup
			}
		}
	}
}
