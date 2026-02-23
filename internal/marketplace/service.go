package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/claude"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/config"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/fetcher"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/settings"
)

// CacheAction indicates what action was taken during caching
type CacheAction string

const (
	CacheActionCloned CacheAction = "cloned"
	CacheActionPulled CacheAction = "pulled"
)

// Service handles marketplace cache operations
type Service struct {
	cachePath string
}

// NewService creates a new marketplace service.
// cachePath must be a fully resolved absolute path (use config.GetMarketplaceCachePath()).
func NewService(cachePath string) *Service {
	return &Service{
		cachePath: cachePath,
	}
}

// GetMarketplacePath returns the path for a specific marketplace
func (s *Service) GetMarketplacePath(name string) string {
	return filepath.Join(s.cachePath, name)
}

// IsCached returns true if marketplace is in the cache
func (s *Service) IsCached(name string) bool {
	cachePath := s.GetMarketplacePath(name)
	_, err := os.Stat(cachePath)
	return err == nil
}

// EnsureCachedResult contains the result of EnsureCached
type EnsureCachedResult struct {
	Action        CacheAction
	CachePath     string
	UpdatesPulled bool
}

// EnsureCached ensures marketplace is cloned to cache (clone if missing, pull if exists).
// Returns the local cache path and action taken.
func (s *Service) EnsureCached(ctx context.Context, name, repoURL, protocol string) (*EnsureCachedResult, error) {
	log := logger.FromContext(ctx)
	cachePath := s.GetMarketplacePath(name)

	log.Info("ensuring marketplace cached",
		"name", name,
		"repoURL", repoURL,
		"protocol", protocol,
		"cachePath", cachePath,
	)

	if err := os.MkdirAll(s.cachePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", s.cachePath, err)
	}

	normalizedURL := fetcher.NormalizeRepoURLWithProtocol(repoURL, protocol)

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		log.Info("cloning marketplace", "url", normalizedURL, "dest", cachePath)
		if err := fetcher.CloneRepo(ctx, normalizedURL, cachePath); err != nil {
			return nil, fmt.Errorf("failed to clone %s: %w", name, err)
		}
		return &EnsureCachedResult{Action: CacheActionCloned, CachePath: cachePath}, nil
	}

	// Update remote URL if protocol changed
	currentURL, err := fetcher.GetRemoteURL(cachePath)
	if err == nil && currentURL != normalizedURL {
		log.Info("updating remote URL",
			"oldURL", currentURL,
			"newURL", normalizedURL,
		)
		if err := fetcher.SetRemoteURL(cachePath, normalizedURL); err != nil {
			return nil, fmt.Errorf("failed to update remote URL: %w", err)
		}
	}

	// Pull latest
	log.Info("updating marketplace", "cachePath", cachePath)
	pullResult, err := fetcher.PullRepo(ctx, cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to update %s: %w", name, err)
	}

	return &EnsureCachedResult{
		Action:        CacheActionPulled,
		CachePath:     cachePath,
		UpdatesPulled: pullResult.UpdatesPulled,
	}, nil
}

// IsRegisteredWithClaude checks if a marketplace is already registered with Claude
func (s *Service) IsRegisteredWithClaude(name, claudeDir string) bool {
	registryPath := registry.GetKnownMarketplacesPath(claudeDir)

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return false
	}

	var knownMarketplaces map[string]any
	if err := json.Unmarshal(data, &knownMarketplaces); err != nil {
		return false
	}

	_, exists := knownMarketplaces[name]
	return exists
}

// UnregisterFromClaude removes a marketplace from Claude.
func (s *Service) UnregisterFromClaude(name, claudeDir string, plat platform.Platform, output io.Writer) error {
	cli, err := claude.New(claudeDir, plat)
	if err != nil {
		return err
	}
	if err := cli.RemoveMarketplace(name, output); err != nil {
		return fmt.Errorf("claude plugin marketplace remove failed: %w", err)
	}
	return nil
}

// RegisterResult contains the result of registering with Claude
type RegisterResult struct {
	AlreadyRegistered bool
	CacheDeleted      bool
	Skipped           bool
}

// DeletePluginCache removes the plugin cache directory for a marketplace.
// This is idempotent - safe to call even if cache doesn't exist.
func (s *Service) DeletePluginCache(ctx context.Context, claudeDir, marketplaceName string) {
	log := logger.FromContext(ctx)
	pluginCachePath := registry.GetPluginCachePath(claudeDir, marketplaceName)

	if _, err := os.Stat(pluginCachePath); os.IsNotExist(err) {
		log.Debug("plugin cache does not exist, nothing to delete",
			"marketplace", marketplaceName,
			"path", pluginCachePath)
		return
	}

	log.Info("removing plugin cache directory",
		"marketplace", marketplaceName,
		"path", pluginCachePath)

	if err := os.RemoveAll(pluginCachePath); err != nil {
		log.Error("failed to delete plugin cache",
			"marketplace", marketplaceName,
			"path", pluginCachePath,
			"error", err)
	}
}

// RegisterWithClaude registers a marketplace with Claude using its local cache path.
// If force is true and marketplace is already registered, it will be removed first.
func (s *Service) RegisterWithClaude(ctx context.Context, localPath, name, claudeDir string, force bool, plat platform.Platform, output io.Writer) (*RegisterResult, error) {
	log := logger.FromContext(ctx)

	log.Info("registering marketplace with Claude",
		"name", name,
		"path", localPath,
		"force", force,
	)

	cli, err := claude.New(claudeDir, plat)
	if err != nil {
		return nil, err
	}

	result := &RegisterResult{}

	if s.IsRegisteredWithClaude(name, claudeDir) {
		log.Info("marketplace already registered", "name", name)
		result.AlreadyRegistered = true
		if !force {
			result.Skipped = true
			return result, nil
		}
		// Force mode: remove first
		log.Info("force mode: removing existing marketplace", "name", name)
		if err := s.UnregisterFromClaude(name, claudeDir, plat, output); err != nil {
			return nil, fmt.Errorf("failed to remove existing marketplace: %w", err)
		}
		s.DeletePluginCache(ctx, claudeDir, name)
		result.CacheDeleted = true
	}

	log.Info("calling claude plugin marketplace add", "path", localPath)
	if err := cli.AddMarketplace(localPath, output); err != nil {
		return nil, fmt.Errorf("claude plugin marketplace add failed: %w", err)
	}

	return result, nil
}

// AddMarketplaceInput contains parameters for adding a marketplace
type AddMarketplaceInput struct {
	// ClaudeDir is the Claude config directory (e.g., ~/.claude)
	ClaudeDir string
	// CommandOutput is the writer for external command output (optional)
	CommandOutput io.Writer
	// ConfigManager manages claudectl configuration
	ConfigManager *config.Manager
	// Ext lists extension namespaces from plugins-ext/ to include
	Ext []string
	// Force re-registration if marketplace is already registered with Claude
	Force bool
	// GitProtocol specifies the preferred git protocol ("ssh" or "https")
	GitProtocol string
	// LocalPath is the absolute path to a local directory (mutually exclusive with Owner/Repo)
	LocalPath string
	// Owner is the GitHub owner
	Owner string
	// Platform provides OS-specific operations
	Platform platform.Platform
	// Printer is used for output during sync
	Printer *printer.Printer
	// Repo is the GitHub repository name
	Repo string
	// SkipRegister skips registering with Claude CLI
	SkipRegister bool
}

// AddMarketplaceResult contains the result of adding a marketplace
type AddMarketplaceResult struct {
	// CacheAction indicates if marketplace was cloned or pulled (for GitHub sources)
	CacheAction CacheAction
	// CachePath is the local path where marketplace is cached
	CachePath string
	// DisabledPlugins is the list of plugins that were disabled (when Force=true)
	DisabledPlugins []string
	// Marketplace contains the parsed marketplace.json data
	Marketplace *fetcher.Marketplace
	// PluginsEnabled is the list of plugins that were enabled
	PluginsEnabled []string
	// RegisterResult contains the result of registering with Claude (nil if skipped)
	RegisterResult *RegisterResult
	// SourceType is either "github" or "directory"
	SourceType string
}

// AddMarketplace adds a marketplace to the system. This is the unified method used by
// both `claudectl init` (for default marketplace) and `claudectl marketplace add` (for custom ones).
//
// Steps:
// 1. Determine source path (local path directly or clone GitHub to cache)
// 2. Read marketplace.json from source
// 3. Add marketplace to settings (extraKnownMarketplaces)
// 4. Update claudectl config (marketplaces list)
// 5. Register with Claude CLI (unless SkipRegister is true)
// 6. Enable plugins from the marketplace
func (s *Service) AddMarketplace(ctx context.Context, input AddMarketplaceInput) (*AddMarketplaceResult, error) {
	log := logger.FromContext(ctx)

	log.Info("adding marketplace",
		"localPath", input.LocalPath,
		"owner", input.Owner,
		"repo", input.Repo,
		"force", input.Force,
		"ext", input.Ext,
		"skipRegister", input.SkipRegister,
	)

	result := &AddMarketplaceResult{}

	var sourcePath string
	var sourceType string
	var sourceValue string

	// Step 1: Determine source path
	if input.LocalPath != "" {
		absPath, err := filepath.Abs(input.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path: %w", err)
		}

		info, err := os.Stat(absPath)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", absPath)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to access path: %w", err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("path is not a directory: %s", absPath)
		}

		sourcePath = absPath
		sourceType = "directory"
		sourceValue = absPath
	} else {
		repoURL := fmt.Sprintf("%s/%s", input.Owner, input.Repo)

		cacheResult, err := s.EnsureCached(ctx, input.Repo, repoURL, input.GitProtocol)
		if err != nil {
			return nil, fmt.Errorf("failed to cache marketplace: %w", err)
		}

		sourcePath = cacheResult.CachePath
		sourceType = "github"
		sourceValue = repoURL
		result.CacheAction = cacheResult.Action
	}

	// Step 2: Read marketplace.json
	log.Info("reading marketplace metadata", "sourcePath", sourcePath)
	mp, err := fetcher.FetchMarketplaceFromPath(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read marketplace: %w", err)
	}
	log.Info("marketplace metadata loaded",
		"name", mp.Name,
		"version", mp.Version,
		"pluginCount", len(mp.Plugins),
	)

	// Step 3: Add marketplace to settings
	settingsMgr := settings.NewManager(input.ClaudeDir)
	if err := settingsMgr.AddMarketplaceWithSource(ctx, mp.Name, sourceType, sourceValue, input.Force); err != nil {
		return nil, fmt.Errorf("failed to add marketplace to settings: %w", err)
	}

	// Step 4: Update claudectl config
	if input.ConfigManager != nil {
		err = input.ConfigManager.ListAdd(ctx, "marketplaces", mp.Name)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("failed to update config: %w", err)
		}
	}

	// Step 5: Register with Claude CLI
	if !input.SkipRegister {
		regResult, err := s.RegisterWithClaude(ctx, sourcePath, mp.Name, input.ClaudeDir, input.Force, input.Platform, input.CommandOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to register marketplace with Claude: %w", err)
		}
		result.RegisterResult = regResult

		if regResult.AlreadyRegistered && regResult.Skipped {
			return nil, fmt.Errorf("marketplace '%s' already exists (use --force to replace)", mp.Name)
		}
	}

	// Step 6: Enable plugins
	if input.Force {
		disabled, err := settingsMgr.DisablePluginsForMarketplace(ctx, mp.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to disable old plugins: %w", err)
		}
		result.DisabledPlugins = disabled
	}

	// Load disabled plugins from base_settings.json
	var disabledPlugins map[string]bool
	if mp.Name == registry.DefaultMarketplaceID {
		disabledPlugins = getDisabledPluginsFromSource(sourcePath)
	}

	pluginNames := make([]string, len(mp.Plugins))
	for i, p := range mp.Plugins {
		pluginNames[i] = p.Name
	}

	// Include extension plugins if specified
	if len(input.Ext) > 0 {
		extPlugins := discoverExtPlugins(sourcePath, input.Ext)
		pluginNames = append(pluginNames, extPlugins...)
	}

	if err := settingsMgr.EnablePluginsWithDefaults(ctx, pluginNames, mp.Name, disabledPlugins); err != nil {
		return nil, fmt.Errorf("failed to enable plugins: %w", err)
	}

	result.CachePath = sourcePath
	result.Marketplace = mp
	result.PluginsEnabled = pluginNames
	result.SourceType = sourceType

	log.Info("marketplace added successfully",
		"name", mp.Name,
		"version", mp.Version,
		"pluginsEnabled", len(pluginNames),
	)

	return result, nil
}

// getDisabledPluginsFromSource loads base_settings.json from the marketplace source
// and returns plugins set to false.
func getDisabledPluginsFromSource(marketplaceSourcePath string) map[string]bool {
	path := filepath.Join(marketplaceSourcePath, "internal", "cmd", "base_settings.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var s map[string]any
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}

	enabledPlugins, ok := s["enabledPlugins"].(map[string]any)
	if !ok {
		return nil
	}

	disabled := make(map[string]bool)
	for key, val := range enabledPlugins {
		if enabled, ok := val.(bool); ok && !enabled {
			disabled[key] = true
		}
	}
	return disabled
}

// discoverExtPlugins finds plugins under plugins-ext/ for the given namespaces.
func discoverExtPlugins(marketplacePath string, namespaces []string) []string {
	var plugins []string
	extDir := filepath.Join(marketplacePath, "plugins-ext")

	for _, ns := range namespaces {
		nsDir := filepath.Join(extDir, ns)
		entries, err := os.ReadDir(nsDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				plugins = append(plugins, ns+"."+entry.Name())
			}
		}
	}

	return plugins
}
