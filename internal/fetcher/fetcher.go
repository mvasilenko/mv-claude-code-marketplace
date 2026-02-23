package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
)

// Marketplace represents a marketplace.json structure
type Marketplace struct {
	Name  string `json:"name"`
	Owner struct {
		Name string `json:"name"`
	} `json:"owner"`
	Plugins []Plugin `json:"plugins"`
	Version string   `json:"version"`
}

// Plugin represents a plugin entry in marketplace.json
type Plugin struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Version     string `json:"version"`
}

// PullRepoResult contains the result of a pull operation
type PullRepoResult struct {
	UpdatesAvailable bool
	UpdatesPulled    bool
}

// FetchMarketplaceFromPath parses marketplace.json from a local directory
func FetchMarketplaceFromPath(localPath string) (*Marketplace, error) {
	log := logger.FromContext(context.Background())

	log.Info("reading marketplace.json", "path", localPath)

	possiblePaths := []string{
		filepath.Join(localPath, ".claude-plugin", "marketplace.json"),
		filepath.Join(localPath, "marketplace.json"),
	}

	var firstParseError error
	for _, path := range possiblePaths {
		log.Debug("checking for marketplace.json", "path", path)
		marketplace, err := readAndParseMarketplace(path)
		if err == nil {
			log.Info("marketplace.json parsed successfully",
				"name", marketplace.Name,
				"version", marketplace.Version,
				"plugins", len(marketplace.Plugins),
			)
			return marketplace, nil
		}
		if firstParseError == nil && !strings.Contains(err.Error(), "file not found") {
			firstParseError = err
			log.Debug("parse error", "path", path, "error", err)
		}
	}

	if firstParseError != nil {
		log.Error("failed to read marketplace.json",
			"searchedPaths", possiblePaths,
			"error", firstParseError,
		)
		return nil, firstParseError
	}

	log.Error("marketplace.json not found", "searchedPaths", possiblePaths)
	return nil, fmt.Errorf("marketplace.json not found in .claude-plugin/ or root directory")
}

// readAndParseMarketplace reads and parses a marketplace.json file
func readAndParseMarketplace(path string) (*Marketplace, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var marketplace Marketplace
	if err := json.Unmarshal(data, &marketplace); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if err := validateMarketplace(&marketplace); err != nil {
		return nil, err
	}

	return &marketplace, nil
}

// validateMarketplace ensures required fields are present
func validateMarketplace(m *Marketplace) error {
	if m.Name == "" {
		return fmt.Errorf("marketplace.json missing required field: name")
	}
	if m.Plugins == nil {
		return fmt.Errorf("marketplace.json missing required field: plugins")
	}

	for i, plugin := range m.Plugins {
		if plugin.Name == "" {
			return fmt.Errorf("plugin at index %d missing required field: name", i)
		}
	}

	return nil
}

// ParseRepository parses a repository string in format "owner/repo"
func ParseRepository(repoStr string) (owner, repo string, err error) {
	parts := strings.Split(repoStr, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: expected 'owner/repo', got '%s'", repoStr)
	}

	owner = strings.TrimSpace(parts[0])
	repo = strings.TrimSpace(parts[1])

	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid repository format: owner and repo cannot be empty")
	}

	return owner, repo, nil
}

// CloneRepo clones a repository to the destination path (atomic: temp + rename)
func CloneRepo(ctx context.Context, repoURL, destPath string) error {
	log := logger.FromContext(ctx)

	log.Info("cloning repository",
		"url", repoURL,
		"dest", destPath,
	)

	tempPath := destPath + ".tmp"

	_ = os.RemoveAll(tempPath)

	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempPath)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("git clone failed",
			"url", repoURL,
			"error", err,
			"output", string(output),
		)
		_ = os.RemoveAll(tempPath)
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, output)
	}

	if err := os.RemoveAll(destPath); err != nil {
		_ = os.RemoveAll(tempPath)
		return fmt.Errorf("failed to remove existing directory: %w", err)
	}

	if err := os.Rename(tempPath, destPath); err != nil {
		_ = os.RemoveAll(tempPath)
		return fmt.Errorf("failed to finalize clone: %w", err)
	}

	log.Info("repository cloned successfully", "path", destPath)
	return nil
}

// PullRepo pulls latest changes in an existing repository
func PullRepo(ctx context.Context, repoPath string) (*PullRepoResult, error) {
	log := logger.FromContext(ctx)

	log.Info("checking for repository updates", "path", repoPath)

	localHeadBytes, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get local HEAD: %w\nOutput: %s", err, localHeadBytes)
	}
	localHead := strings.TrimSpace(string(localHeadBytes))

	fetchCmd := exec.Command("git", "-C", repoPath, "fetch", "origin")
	fetchCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git fetch failed: %w\nOutput: %s", err, fetchOutput)
	}

	branchBytes, err := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch name: %w\nOutput: %s", err, branchBytes)
	}
	branchName := strings.TrimSpace(string(branchBytes))

	remoteRef := fmt.Sprintf("origin/%s", branchName)
	remoteHeadBytes, err := exec.Command("git", "-C", repoPath, "rev-parse", remoteRef).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote HEAD: %w\nOutput: %s", err, remoteHeadBytes)
	}
	remoteHead := strings.TrimSpace(string(remoteHeadBytes))

	if localHead == remoteHead {
		log.Info("repository is up to date", "path", repoPath)
		return &PullRepoResult{
			UpdatesAvailable: false,
			UpdatesPulled:    false,
		}, nil
	}

	log.Info("updates available, pulling changes", "path", repoPath)

	pullCmd := exec.Command("git", "-C", repoPath, "pull", "--ff-only")
	pullCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	pullOutput, err := pullCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git pull failed: %w\nOutput: %s", err, pullOutput)
	}

	log.Info("repository updated successfully", "path", repoPath)
	return &PullRepoResult{
		UpdatesAvailable: true,
		UpdatesPulled:    true,
	}, nil
}

// SetRemoteURL updates the origin remote URL for a repository
func SetRemoteURL(repoPath, url string) error {
	cmd := exec.Command("git", "-C", repoPath, "remote", "set-url", "origin", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set remote URL: %w\nOutput: %s", err, output)
	}
	return nil
}

// GetRemoteURL gets the origin remote URL for a repository
func GetRemoteURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w\nOutput: %s", err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

// NormalizeRepoURL converts various input formats to a git-clone-able URL.
// Supports: owner/repo, git@github.com:owner/repo.git, https://github.com/owner/repo.git
// Defaults to HTTPS for owner/repo format.
func NormalizeRepoURL(input string) string {
	if strings.HasPrefix(input, "git@") || strings.HasPrefix(input, "https://") {
		return input
	}
	return fmt.Sprintf("https://github.com/%s.git", input)
}

// NormalizeRepoURLWithProtocol converts various input formats to a git-clone-able URL
// using the specified protocol preference for owner/repo format.
func NormalizeRepoURLWithProtocol(input, protocol string) string {
	if strings.HasPrefix(input, "git@") || strings.HasPrefix(input, "https://") {
		return input
	}
	if protocol == "ssh" {
		return fmt.Sprintf("git@github.com:%s.git", input)
	}
	return fmt.Sprintf("https://github.com/%s.git", input)
}

// IsLocalPath determines if input is a local file path or a GitHub repository.
// Returns true for local paths, false for GitHub owner/repo format.
func IsLocalPath(input string) bool {
	if strings.HasPrefix(input, "/") ||
		strings.HasPrefix(input, "./") ||
		strings.HasPrefix(input, "../") ||
		strings.HasPrefix(input, "~") {
		return true
	}

	if info, err := os.Stat(input); err == nil && info.IsDir() {
		return true
	}

	parts := strings.Split(input, "/")
	if len(parts) == 2 && !strings.Contains(parts[0], ".") && parts[0] != "" && parts[1] != "" {
		return false
	}

	return true
}
