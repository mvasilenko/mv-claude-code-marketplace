package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
)

// GitProtocol represents the preferred Git access protocol
type GitProtocol string

const (
	GitProtocolSSH   GitProtocol = "ssh"
	GitProtocolHTTPS GitProtocol = "https"
)

// DefaultMarketplaceCacheDir is the default directory name for marketplace cache
const DefaultMarketplaceCacheDir = "marketplace-cache"

// Validate checks if the GitProtocol value is valid
func (p GitProtocol) Validate() error {
	switch p {
	case GitProtocolSSH, GitProtocolHTTPS, "":
		return nil
	default:
		return fmt.Errorf("invalid git protocol: %s (must be 'ssh' or 'https')", p)
	}
}

// Config represents claudectl configuration
type Config struct {
	AutoUpdate           AutoUpdateConfig `json:"autoUpdate"`
	Backend              string           `json:"backend,omitempty"`
	ClaudeConfigDir      string           `json:"claudeConfigDir,omitempty"`
	Debug                bool             `json:"debug,omitempty"`
	ExtensionNamespaces  []string         `json:"extensionNamespaces,omitempty"`
	GitProtocol          GitProtocol      `json:"gitProtocol,omitempty"`
	LastUpdateCheck      time.Time        `json:"lastUpdateCheck"`
	MarketplaceCachePath string           `json:"marketplaceCachePath,omitempty"`
	Marketplaces         []string         `json:"marketplaces"`
	Software             SoftwareConfig   `json:"software"`
}

// AutoUpdateConfig contains auto-update settings
type AutoUpdateConfig struct {
	CheckInterval string `json:"checkInterval"` // Go duration string (e.g., "3h")
	Enabled       bool   `json:"enabled"`
}

// SoftwareConfig contains user software preferences
type SoftwareConfig struct {
	ExcludedTools      []string `json:"excludedTools,omitempty"`
	LastInstalledTools []string `json:"lastInstalledTools,omitempty"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		AutoUpdate: AutoUpdateConfig{
			CheckInterval: "3h",
			Enabled:       true,
		},
		LastUpdateCheck: time.Time{},
		Marketplaces:    []string{registry.DefaultMarketplaceID},
		Software: SoftwareConfig{
			ExcludedTools:      []string{},
			LastInstalledTools: []string{},
		},
	}
}

// Manager handles configuration operations
type Manager struct {
	filePath string
}

// NewManager creates a new config manager.
// configDir is typically from ResolveConfigDir() or platform.GetConfigDir().
func NewManager(configDir string) *Manager {
	return &Manager{
		filePath: filepath.Join(configDir, "config.json"),
	}
}

// ResolveConfigDir determines the config directory with priority:
// 1. CLAUDECTL_CONFIG_DIR environment variable
// 2. platform.GetConfigDir() default
func ResolveConfigDir(p platform.Platform) string {
	if envDir := os.Getenv("CLAUDECTL_CONFIG_DIR"); envDir != "" {
		return expandPath(envDir)
	}
	return p.GetConfigDir()
}

// Load reads configuration from file
func (m *Manager) Load(ctx context.Context) (*Config, error) {
	log := logger.FromContext(ctx)

	log.Info("loading config file",
		"path", m.filePath,
	)

	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		log.Info("config file not found, using defaults",
			"path", m.filePath,
		)
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		log.Error("failed to read config file",
			"error", err,
			"path", m.filePath,
		)
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Error("failed to parse config file",
			"error", err,
			"path", m.filePath,
		)
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	log.Info("config file loaded successfully",
		"path", m.filePath,
		"backend", cfg.Backend,
		"debug", cfg.Debug,
	)

	return &cfg, nil
}

// Save writes configuration to file atomically
func (m *Manager) Save(ctx context.Context, cfg *Config) error {
	log := logger.FromContext(ctx)

	log.Info("saving config file",
		"path", m.filePath,
		"backend", cfg.Backend,
		"debug", cfg.Debug,
	)

	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		log.Error("failed to create config directory",
			"error", err,
			"directory", filepath.Dir(m.filePath),
		)
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Error("failed to marshal config",
			"error", err,
		)
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	data = append(data, '\n')

	tmpPath := m.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		log.Error("failed to write temp config file",
			"error", err,
			"path", tmpPath,
		)
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := os.Rename(tmpPath, m.filePath); err != nil {
		_ = os.Remove(tmpPath) //nolint:errcheck // cleanup
		log.Error("failed to rename temp file to final config",
			"error", err,
			"temp_path", tmpPath,
			"final_path", m.filePath,
		)
		return fmt.Errorf("failed to finalize config: %w", err)
	}

	log.Info("config file saved successfully",
		"path", m.filePath,
		"size_bytes", len(data),
	)

	return nil
}

// SaveExtensionNamespaces updates extension namespaces in config and persists.
func (m *Manager) SaveExtensionNamespaces(ctx context.Context, cfg *Config, namespaces []string) (string, error) {
	cfg.ExtensionNamespaces = namespaces
	if err := m.Save(ctx, cfg); err != nil {
		return "", fmt.Errorf("failed to save extension namespaces: %w", err)
	}
	return fmt.Sprintf("Saved extension namespaces: %v", namespaces), nil
}

// Get retrieves a configuration value by key path (dot notation)
func (m *Manager) Get(ctx context.Context, key string) (any, error) {
	cfg, err := m.Load(ctx)
	if err != nil {
		return nil, err
	}

	return getNestedValue(cfg, key)
}

// Set updates a configuration value by key path (dot notation)
func (m *Manager) Set(ctx context.Context, key, value string) error {
	cfg, err := m.Load(ctx)
	if err != nil {
		return err
	}

	if err := setNestedValue(cfg, key, value); err != nil {
		return err
	}

	return m.Save(ctx, cfg)
}

// ListAdd appends a value to an array configuration key
func (m *Manager) ListAdd(ctx context.Context, key, value string) error {
	cfg, err := m.Load(ctx)
	if err != nil {
		return err
	}

	if err := listAddValue(cfg, key, value); err != nil {
		return err
	}

	return m.Save(ctx, cfg)
}

// ListRemove removes a value from an array configuration key
func (m *Manager) ListRemove(ctx context.Context, key, value string) error {
	cfg, err := m.Load(ctx)
	if err != nil {
		return err
	}

	if err := listRemoveValue(cfg, key, value); err != nil {
		return err
	}

	return m.Save(ctx, cfg)
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.filePath
}

// getNestedValue retrieves value using dot notation
func getNestedValue(cfg *Config, key string) (any, error) {
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "autoUpdate":
		if len(parts) == 1 {
			return cfg.AutoUpdate, nil
		}
		switch parts[1] {
		case "enabled":
			return cfg.AutoUpdate.Enabled, nil
		case "checkInterval":
			return cfg.AutoUpdate.CheckInterval, nil
		default:
			return nil, fmt.Errorf("unknown key: %s", key)
		}
	case "backend":
		return cfg.Backend, nil
	case "claudeConfigDir":
		return cfg.ClaudeConfigDir, nil
	case "debug":
		return cfg.Debug, nil
	case "extensionNamespaces":
		return cfg.ExtensionNamespaces, nil
	case "gitProtocol":
		return string(cfg.GitProtocol), nil
	case "lastUpdateCheck":
		return cfg.LastUpdateCheck, nil
	case "marketplaceCachePath":
		return cfg.MarketplaceCachePath, nil
	case "marketplaces":
		return cfg.Marketplaces, nil
	case "software":
		if len(parts) == 1 {
			return cfg.Software, nil
		}
		switch parts[1] {
		case "excludedTools":
			return cfg.Software.ExcludedTools, nil
		case "lastInstalledTools":
			return cfg.Software.LastInstalledTools, nil
		default:
			return nil, fmt.Errorf("unknown key: %s", key)
		}
	default:
		return nil, fmt.Errorf("unknown key: %s", key)
	}
}

// setNestedValue updates value using dot notation
func setNestedValue(cfg *Config, key, value string) error {
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "autoUpdate":
		if len(parts) == 1 {
			return fmt.Errorf("cannot set entire autoUpdate object")
		}
		switch parts[1] {
		case "enabled":
			b, err := parseBool(value)
			if err != nil {
				return err
			}
			cfg.AutoUpdate.Enabled = b
		case "checkInterval":
			if _, err := time.ParseDuration(value); err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			cfg.AutoUpdate.CheckInterval = value
		default:
			return fmt.Errorf("unknown key: %s", key)
		}
	case "backend":
		cfg.Backend = value
	case "claudeConfigDir":
		cfg.ClaudeConfigDir = value
	case "debug":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Debug = b
	case "gitProtocol":
		p := GitProtocol(value)
		if err := p.Validate(); err != nil {
			return err
		}
		cfg.GitProtocol = p
	case "marketplaceCachePath":
		cfg.MarketplaceCachePath = value
	case "marketplaces":
		return fmt.Errorf("cannot set marketplaces directly, use list-add or list-remove")
	case "lastUpdateCheck":
		return fmt.Errorf("cannot set lastUpdateCheck manually")
	default:
		return fmt.Errorf("unknown key: %s", key)
	}

	return nil
}

// parseBool parses a boolean string value
func parseBool(value string) (bool, error) {
	switch value {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s (must be 'true', 'false', '1', or '0')", value)
	}
}

// listAddValue appends a value to an array field
func listAddValue(cfg *Config, key, value string) error {
	switch key {
	case "extensionNamespaces":
		for _, ns := range cfg.ExtensionNamespaces {
			if ns == value {
				return fmt.Errorf("namespace '%s' already exists", value)
			}
		}
		cfg.ExtensionNamespaces = append(cfg.ExtensionNamespaces, value)
	case "marketplaces":
		for _, mp := range cfg.Marketplaces {
			if mp == value {
				return fmt.Errorf("marketplace '%s' already exists", value)
			}
		}
		cfg.Marketplaces = append(cfg.Marketplaces, value)
	case "software.excludedTools":
		for _, tool := range cfg.Software.ExcludedTools {
			if tool == value {
				return fmt.Errorf("tool '%s' already excluded", value)
			}
		}
		cfg.Software.ExcludedTools = append(cfg.Software.ExcludedTools, value)
	default:
		return fmt.Errorf("list-add not supported for key: %s", key)
	}

	return nil
}

// listRemoveValue removes a value from an array field
func listRemoveValue(cfg *Config, key, value string) error {
	switch key {
	case "extensionNamespaces":
		found := false
		newList := make([]string, 0, len(cfg.ExtensionNamespaces))
		for _, ns := range cfg.ExtensionNamespaces {
			if ns != value {
				newList = append(newList, ns)
			} else {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("namespace '%s' not found", value)
		}
		cfg.ExtensionNamespaces = newList
	case "marketplaces":
		found := false
		newList := make([]string, 0, len(cfg.Marketplaces))
		for _, mp := range cfg.Marketplaces {
			if mp != value {
				newList = append(newList, mp)
			} else {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("marketplace '%s' not found", value)
		}
		cfg.Marketplaces = newList
	case "software.excludedTools":
		found := false
		newList := make([]string, 0, len(cfg.Software.ExcludedTools))
		for _, tool := range cfg.Software.ExcludedTools {
			if tool != value {
				newList = append(newList, tool)
			} else {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("tool '%s' not excluded", value)
		}
		cfg.Software.ExcludedTools = newList
	default:
		return fmt.Errorf("list-remove not supported for key: %s", key)
	}

	return nil
}

// IsToolExcluded checks if a tool is in the exclusion list
func (c *Config) IsToolExcluded(toolID string) bool {
	for _, excluded := range c.Software.ExcludedTools {
		if excluded == toolID {
			return true
		}
	}
	return false
}

// AddExcludedTool adds a tool to the exclusion list
func (c *Config) AddExcludedTool(toolID string) {
	if !c.IsToolExcluded(toolID) {
		c.Software.ExcludedTools = append(c.Software.ExcludedTools, toolID)
	}
}

// RemoveExcludedTool removes a tool from the exclusion list
func (c *Config) RemoveExcludedTool(toolID string) {
	filtered := make([]string, 0, len(c.Software.ExcludedTools))
	for _, excluded := range c.Software.ExcludedTools {
		if excluded != toolID {
			filtered = append(filtered, excluded)
		}
	}
	c.Software.ExcludedTools = filtered
}

// UpdateLastInstalledTools updates the list of installed tools
func (c *Config) UpdateLastInstalledTools(tools []string) {
	c.Software.LastInstalledTools = tools
}

// GetNewTools returns tools that are new since last update
func (c *Config) GetNewTools(allTools []string) []string {
	lastInstalled := make(map[string]bool)
	for _, tool := range c.Software.LastInstalledTools {
		lastInstalled[tool] = true
	}

	var newTools []string
	for _, tool := range allTools {
		if !lastInstalled[tool] && !c.IsToolExcluded(tool) {
			newTools = append(newTools, tool)
		}
	}
	return newTools
}

// GetMarketplaceCachePath returns the cache path, expanding ~ to home dir.
// If not configured, returns claudeDir/marketplace-cache.
func (c *Config) GetMarketplaceCachePath(claudeDir string) string {
	if c.MarketplaceCachePath != "" {
		return expandPath(c.MarketplaceCachePath)
	}
	return filepath.Join(claudeDir, DefaultMarketplaceCacheDir)
}

// expandPath expands ~ to home directory
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
