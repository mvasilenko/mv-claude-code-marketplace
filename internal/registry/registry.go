package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultMarketplaceID is the identifier for the default marketplace
	DefaultMarketplaceID = "mv-claude-code-marketplace"
	// DefaultMarketplaceOrg is the GitHub organization for the default marketplace
	DefaultMarketplaceOrg = "mvasilenko"
	// DefaultMarketplaceRepo is the repository name for the default marketplace
	DefaultMarketplaceRepo = "mv-claude-code-marketplace"
)

// Path constants for resource locations within a marketplace
var (
	baseSettingsRelativePath   = []string{"internal", "cmd", "base_settings.json"}
	rulesRelativePath          = []string{"rules"}
	softwareConfigRelativePath = []string{"internal", "software", "software_config.json"}
)

// GetKnownMarketplacesPath returns the path to known_marketplaces.json
func GetKnownMarketplacesPath(claudeDir string) string {
	return filepath.Join(claudeDir, "plugins", "known_marketplaces.json")
}

// GetPluginCachePath returns the path to the plugin cache directory for a specific marketplace
func GetPluginCachePath(claudeDir, marketplaceName string) string {
	return filepath.Join(claudeDir, "plugins", "cache", marketplaceName)
}

// IsDefaultMarketplace returns true if the marketplace is the default marketplace.
// For GitHub sources, pass owner and repo for strict matching.
// For local paths, pass empty strings - name match is sufficient.
func IsDefaultMarketplace(name, owner, repo string) bool {
	if name != DefaultMarketplaceID {
		return false
	}
	if owner != "" || repo != "" {
		return owner == DefaultMarketplaceOrg && repo == DefaultMarketplaceRepo
	}
	return true
}

// PathResolver provides methods for resolving marketplace paths
type PathResolver interface {
	BaseSettingsPath() string
	BaseSettingsPathForMarketplace(marketplaceName string) string
	ClaudeDir() string
	RulesPath() string
	SoftwareConfigPath() string
}

// Compile-time verification that Resolver implements PathResolver
var _ PathResolver = (*Resolver)(nil)

// MarketplaceEntry represents a marketplace entry in known_marketplaces.json
type MarketplaceEntry struct {
	InstallLocation string            `json:"installLocation"`
	LastUpdated     time.Time         `json:"lastUpdated"`
	Source          MarketplaceSource `json:"source"`
}

// MarketplaceSource represents the source information for a marketplace
type MarketplaceSource struct {
	Path   string `json:"path,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Source string `json:"source"`
}

// Resolver provides marketplace path resolution
type Resolver struct {
	claudeDir string
}

// NewResolver creates a new marketplace resolver
// claudeDir is the base Claude directory (e.g., ~/.claude)
func NewResolver(claudeDir string) *Resolver {
	return &Resolver{
		claudeDir: claudeDir,
	}
}

// ClaudeDir returns the base Claude directory
func (r *Resolver) ClaudeDir() string {
	return r.claudeDir
}

// BaseSettingsPath returns the path to base_settings.json for the default marketplace
func (r *Resolver) BaseSettingsPath() string {
	return r.BaseSettingsPathForMarketplace(DefaultMarketplaceID)
}

// BaseSettingsPathForMarketplace returns the path to base_settings.json for a specific marketplace
func (r *Resolver) BaseSettingsPathForMarketplace(marketplaceName string) string {
	path, found, err := r.getResourcePath(
		marketplaceName,
		baseSettingsRelativePath...,
	)

	if err != nil || !found {
		parts := append([]string{r.claudeDir, "plugins", "marketplaces", marketplaceName}, baseSettingsRelativePath...)
		return filepath.Join(parts...)
	}

	return path
}

// RulesPath returns the path to the rules directory for the default marketplace
func (r *Resolver) RulesPath() string {
	path, found, err := r.getResourcePath(
		DefaultMarketplaceID,
		rulesRelativePath...,
	)

	if err != nil || !found {
		parts := append([]string{r.claudeDir, "plugins", "marketplaces", DefaultMarketplaceID}, rulesRelativePath...)
		return filepath.Join(parts...)
	}

	return path
}

// SoftwareConfigPath returns the path to software_config.json for the default marketplace
func (r *Resolver) SoftwareConfigPath() string {
	path, found, err := r.getResourcePath(
		DefaultMarketplaceID,
		softwareConfigRelativePath...,
	)

	if err == nil && found {
		if _, statErr := os.Stat(path); statErr == nil {
			return path
		}
	}

	// Fall back to hardcoded path
	parts := append([]string{r.claudeDir, "plugins", "marketplaces", DefaultMarketplaceID}, softwareConfigRelativePath...)
	return filepath.Join(parts...)
}

// GetAllRulesPaths returns the rules directory paths for all known marketplaces.
// Returns a map of marketplace name -> rules directory path.
// Only includes marketplaces that have a rules directory.
func (r *Resolver) GetAllRulesPaths() (map[string]string, error) {
	registry, err := r.loadRegistry()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for marketplaceName, entry := range registry {
		rulesPath := filepath.Join(entry.InstallLocation, "rules")
		if _, err := os.Stat(rulesPath); err == nil {
			result[marketplaceName] = rulesPath
		}
	}

	return result, nil
}

// GetInstallLocation resolves the installation location for a marketplace.
// Returns (path, found, error).
func (r *Resolver) GetInstallLocation(marketplaceName string) (string, bool, error) {
	registry, err := r.loadRegistry()
	if err != nil {
		return "", false, err
	}

	entry, found := registry[marketplaceName]
	if !found {
		return "", false, nil
	}

	return entry.InstallLocation, true, nil
}

// getResourcePath constructs a path within a marketplace installation.
// Returns (fullPath, found, error).
func (r *Resolver) getResourcePath(marketplaceName string, pathParts ...string) (string, bool, error) {
	installLocation, found, err := r.GetInstallLocation(marketplaceName)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}

	parts := append([]string{installLocation}, pathParts...)
	fullPath := filepath.Join(parts...)

	return fullPath, true, nil
}

// loadRegistry loads the known_marketplaces.json file.
// Returns empty map if file doesn't exist (not an error).
func (r *Resolver) loadRegistry() (map[string]MarketplaceEntry, error) {
	registryPath := GetKnownMarketplacesPath(r.claudeDir)

	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return make(map[string]MarketplaceEntry), nil
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	var registry map[string]MarketplaceEntry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry file: %w", err)
	}

	return registry, nil
}
