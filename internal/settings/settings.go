package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
)

// ErrMarketplaceNotFound is returned when a marketplace is not found
var ErrMarketplaceNotFound = errors.New("marketplace not found")

// Manager handles Claude Code settings operations
type Manager struct {
	filePath string
}

// NewManager creates a new settings manager with a custom Claude directory
func NewManager(claudeDir string) *Manager {
	return &Manager{
		filePath: filepath.Join(claudeDir, "settings.json"),
	}
}

// NewManagerWithDefaults creates a settings manager using default resolution
func NewManagerWithDefaults() *Manager {
	return NewManager(ResolveClaudeDir("", ""))
}

// ResolveClaudeDir determines the Claude directory with priority:
// 1. flagOverride (--claude-dir flag)
// 2. configValue (saved in claudectl config)
// 3. CLAUDE_CONFIG_DIR environment variable
// 4. Default: ~/.claude
//
// All paths are expanded to resolve ~ to the user's home directory.
func ResolveClaudeDir(flagOverride, configValue string) string {
	if flagOverride != "" {
		return expandPath(flagOverride)
	}

	if configValue != "" {
		return expandPath(configValue)
	}

	if envDir := os.Getenv("CLAUDE_CONFIG_DIR"); envDir != "" {
		return expandPath(envDir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude"
	}
	return filepath.Join(home, ".claude")
}

// Load reads settings from file.
// Returns empty map if file doesn't exist.
func (m *Manager) Load(ctx context.Context) (map[string]any, error) {
	log := logger.FromContext(ctx)

	log.Info("loading settings file",
		"path", m.filePath,
	)

	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		log.Info("settings file not found, using empty settings",
			"path", m.filePath,
		)
		return make(map[string]any), nil
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		log.Error("failed to read settings file",
			"error", err,
			"path", m.filePath,
		)
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		log.Error("failed to parse settings file",
			"error", err,
			"path", m.filePath,
		)
		return nil, fmt.Errorf("invalid settings JSON: %w", err)
	}

	log.Info("settings file loaded successfully",
		"path", m.filePath,
		"keys_count", len(settings),
	)

	return settings, nil
}

// Save writes settings to file atomically
func (m *Manager) Save(ctx context.Context, settings map[string]any) error {
	log := logger.FromContext(ctx)

	log.Info("saving settings file",
		"path", m.filePath,
		"keys_count", len(settings),
	)

	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		log.Error("failed to create settings directory",
			"error", err,
			"directory", filepath.Dir(m.filePath),
		)
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Error("failed to marshal settings",
			"error", err,
		)
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	data = append(data, '\n')

	tmpPath := m.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		log.Error("failed to write temp settings file",
			"error", err,
			"path", tmpPath,
		)
		return fmt.Errorf("failed to write settings: %w", err)
	}

	if err := os.Rename(tmpPath, m.filePath); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // cleanup
		log.Error("failed to rename temp file to final settings",
			"error", err,
			"temp_path", tmpPath,
			"final_path", m.filePath,
		)
		return fmt.Errorf("failed to finalize settings: %w", err)
	}

	log.Info("settings file saved successfully",
		"path", m.filePath,
		"size_bytes", len(data),
	)

	return nil
}

// GetSettingsPath returns the path to the settings file
func (m *Manager) GetSettingsPath() string {
	return m.filePath
}

// AddMarketplace adds a marketplace to extraKnownMarketplaces.
// If marketplace exists with same repo, this is a no-op (idempotent).
// If marketplace exists with different repo, returns an error.
func (m *Manager) AddMarketplace(ctx context.Context, name, owner, repo string) error {
	settings, err := m.Load(ctx)
	if err != nil {
		return err
	}

	extraKnownMarketplaces, ok := settings["extraKnownMarketplaces"].(map[string]any)
	if !ok {
		extraKnownMarketplaces = make(map[string]any)
		settings["extraKnownMarketplaces"] = extraKnownMarketplaces
	}

	if existing, ok := extraKnownMarketplaces[name].(map[string]any); ok {
		if source, ok := existing["source"].(map[string]any); ok {
			existingRepo, _ := source["repo"].(string)
			fullRepo := fmt.Sprintf("%s/%s", owner, repo)
			if existingRepo == fullRepo {
				return nil
			}
			return fmt.Errorf("marketplace '%s' already exists with different repo: %s (trying to add: %s)", name, existingRepo, fullRepo)
		}
	}

	extraKnownMarketplaces[name] = map[string]any{
		"source": map[string]any{
			"source": "github",
			"repo":   fmt.Sprintf("%s/%s", owner, repo),
		},
	}

	return m.Save(ctx, settings)
}

// AddMarketplaceWithSource adds a marketplace with explicit source type.
// sourceType should be "github" or "directory".
// repoOrPath is either "owner/repo" for GitHub or absolute path for directory.
func (m *Manager) AddMarketplaceWithSource(ctx context.Context, name, sourceType, repoOrPath string, force bool) error {
	settings, err := m.Load(ctx)
	if err != nil {
		return err
	}

	extraKnownMarketplaces, ok := settings["extraKnownMarketplaces"].(map[string]any)
	if !ok {
		extraKnownMarketplaces = make(map[string]any)
		settings["extraKnownMarketplaces"] = extraKnownMarketplaces
	}

	if existing, ok := extraKnownMarketplaces[name].(map[string]any); ok && !force {
		if source, ok := existing["source"].(map[string]any); ok {
			existingSourceType, _ := source["source"].(string)
			if existingSourceType == sourceType {
				if sourceType == "directory" {
					existingPath, _ := source["path"].(string)
					if existingPath == repoOrPath {
						return nil
					}
				} else {
					existingRepo, _ := source["repo"].(string)
					if existingRepo == repoOrPath {
						return nil
					}
				}
			}
			return fmt.Errorf("marketplace '%s' already exists (use --force to replace)", name)
		}
	}

	var sourceData map[string]any
	if sourceType == "directory" {
		sourceData = map[string]any{
			"source": "directory",
			"path":   repoOrPath,
		}
	} else {
		sourceData = map[string]any{
			"source": "github",
			"repo":   repoOrPath,
		}
	}

	extraKnownMarketplaces[name] = map[string]any{
		"source": sourceData,
	}

	return m.Save(ctx, settings)
}

// EnablePlugins adds plugins to enabledPlugins.
// Plugins that already exist keep their current value (user preference wins).
func (m *Manager) EnablePlugins(ctx context.Context, plugins []string, marketplace string) error {
	return m.EnablePluginsWithDefaults(ctx, plugins, marketplace, nil)
}

// EnablePluginsWithDefaults adds plugins to enabledPlugins with optional default values.
// Plugins that already exist keep their current value (user preference always wins).
// defaults maps "plugin@marketplace" to true/false (from base_settings.json).
func (m *Manager) EnablePluginsWithDefaults(ctx context.Context, plugins []string, marketplace string, defaults map[string]bool) error {
	settings, err := m.Load(ctx)
	if err != nil {
		return err
	}

	enabledPlugins, ok := settings["enabledPlugins"].(map[string]any)
	if !ok {
		enabledPlugins = make(map[string]any)
		settings["enabledPlugins"] = enabledPlugins
	}

	for _, plugin := range plugins {
		pluginKey := fmt.Sprintf("%s@%s", plugin, marketplace)
		if _, exists := enabledPlugins[pluginKey]; !exists {
			defaultValue := true
			if defaults != nil {
				if val, ok := defaults[pluginKey]; ok {
					defaultValue = val
				}
			}
			enabledPlugins[pluginKey] = defaultValue
		}
	}

	return m.Save(ctx, settings)
}

// DisablePluginsForMarketplace removes all plugins for a given marketplace from enabledPlugins.
// Returns the list of disabled plugin keys.
func (m *Manager) DisablePluginsForMarketplace(ctx context.Context, marketplace string) ([]string, error) {
	settings, err := m.Load(ctx)
	if err != nil {
		return nil, err
	}

	enabledPlugins, ok := settings["enabledPlugins"].(map[string]any)
	if !ok {
		return nil, nil
	}

	suffix := "@" + marketplace
	var disabled []string
	for pluginKey := range enabledPlugins {
		if strings.HasSuffix(pluginKey, suffix) {
			disabled = append(disabled, pluginKey)
			delete(enabledPlugins, pluginKey)
		}
	}

	if len(disabled) > 0 {
		if err := m.Save(ctx, settings); err != nil {
			return nil, err
		}
	}

	return disabled, nil
}

// MarketplaceExists checks if a marketplace exists.
// Returns (exists, repo, error).
// The repo is normalized to "owner/repo" format for comparison.
func (m *Manager) MarketplaceExists(ctx context.Context, name string) (bool, string, error) {
	settings, err := m.Load(ctx)
	if err != nil {
		return false, "", err
	}

	extraKnownMarketplaces, ok := settings["extraKnownMarketplaces"].(map[string]any)
	if !ok {
		return false, "", nil
	}

	marketplace, ok := extraKnownMarketplaces[name].(map[string]any)
	if !ok {
		return false, "", nil
	}

	source, ok := marketplace["source"].(map[string]any)
	if !ok {
		return true, "", nil
	}

	sourceType, _ := source["source"].(string)
	if sourceType == "directory" {
		path, _ := source["path"].(string)
		return true, path, nil
	}

	repo, _ := source["repo"].(string)
	return true, normalizeGitHubRepo(repo), nil
}

// GetMarketplaceRepo returns the repo URL for a marketplace from settings
func (m *Manager) GetMarketplaceRepo(ctx context.Context, name string) (string, error) {
	settings, err := m.Load(ctx)
	if err != nil {
		return "", err
	}

	extraKnownMarketplaces, ok := settings["extraKnownMarketplaces"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("marketplace %s: %w", name, ErrMarketplaceNotFound)
	}

	mp, ok := extraKnownMarketplaces[name].(map[string]any)
	if !ok {
		return "", fmt.Errorf("marketplace %s: %w", name, ErrMarketplaceNotFound)
	}

	source, ok := mp["source"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("marketplace %s has no source", name)
	}

	repo, ok := source["repo"].(string)
	if !ok {
		return "", fmt.Errorf("marketplace %s has no repo URL", name)
	}

	return repo, nil
}

// RemoveMarketplace removes a marketplace from extraKnownMarketplaces
// and disables all associated plugins from enabledPlugins.
// Returns (repo, disabledPlugins, error).
func (m *Manager) RemoveMarketplace(ctx context.Context, name string) (string, []string, error) {
	settings, err := m.Load(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load settings: %w", err)
	}

	extraKnownMarketplaces, ok := settings["extraKnownMarketplaces"].(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("marketplace '%s': %w", name, ErrMarketplaceNotFound)
	}

	marketplace, ok := extraKnownMarketplaces[name].(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("marketplace '%s': %w", name, ErrMarketplaceNotFound)
	}

	repo := ""
	if source, ok := marketplace["source"].(map[string]any); ok {
		repo, _ = source["repo"].(string)
	}

	delete(extraKnownMarketplaces, name)

	enabledPlugins, ok := settings["enabledPlugins"].(map[string]any)
	if !ok {
		enabledPlugins = make(map[string]any)
	}

	var disabledPlugins []string
	suffix := "@" + name
	for pluginKey := range enabledPlugins {
		if strings.HasSuffix(pluginKey, suffix) {
			disabledPlugins = append(disabledPlugins, pluginKey)
			delete(enabledPlugins, pluginKey)
		}
	}

	if err := m.Save(ctx, settings); err != nil {
		return "", nil, fmt.Errorf("failed to save settings after removing marketplace '%s': %w", name, err)
	}

	return repo, disabledPlugins, nil
}

// expandPath expands ~ to home directory.
// Uses filepath.FromSlash to normalize separators before checking,
// so both ~/ and ~\ work on all platforms.
func expandPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
		return path
	}
	normalized := filepath.FromSlash(path)
	tildePrefix := "~" + string(filepath.Separator)
	if strings.HasPrefix(normalized, tildePrefix) {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, normalized[2:])
		}
	}
	return path
}

// normalizeGitHubRepo extracts "owner/repo" from various GitHub URL formats
func normalizeGitHubRepo(repo string) string {
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "git@github.com:")
	return repo
}
