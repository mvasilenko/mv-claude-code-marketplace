package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/config"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/marketplace"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/syncer"
)

//go:embed base_settings.json
var baseSettingsJSON []byte

// InitCmd initializes Claude settings with base configuration
type InitCmd struct {
	Ext   []string `help:"Include plugins from plugins-ext/ for specified namespace (can be repeated)." short:"e"`
	Force bool     `help:"Force re-registration of marketplace with Claude (removes and re-adds)." short:"f" default:"false"`

	p *printer.Printer `kong:"-"`
}

// Run executes the init command
func (c *InitCmd) Run(ctx *Context) error {
	c.p = printer.New(ctx.CLI.JSON, ctx.CLI.Quiet, ctx.CLI.Verbose)

	claudeDir := ResolveClaudeDirFromContext(ctx)
	resolver := registry.NewResolver(claudeDir)

	// Check if already initialized (unless --force is used)
	if !c.Force {
		if err := c.checkAlreadyInitialized(claudeDir); err != nil {
			return err
		}
	}

	plat := platform.New()
	cfgMgr := config.NewManager(config.ResolveConfigDir(plat))
	cfg, _ := cfgMgr.Load(ctx.Context)

	mpService := marketplace.NewService(cfg.GetMarketplaceCachePath(claudeDir))

	// Add default marketplace
	c.p.Section("Setting up marketplace...")
	ctx.Logger.Info("adding marketplace",
		"owner", registry.DefaultMarketplaceOrg,
		"repo", registry.DefaultMarketplaceRepo,
		"protocol", string(cfg.GitProtocol),
		"force", c.Force,
	)

	_, err := mpService.AddMarketplace(ctx.Context, marketplace.AddMarketplaceInput{
		ClaudeDir:     claudeDir,
		ConfigManager: cfgMgr,
		Ext:           c.Ext,
		Force:         c.Force,
		GitProtocol:   string(cfg.GitProtocol),
		Owner:         registry.DefaultMarketplaceOrg,
		Platform:      plat,
		Printer:       c.p,
		Repo:          registry.DefaultMarketplaceRepo,
		SkipRegister:  false,
	})
	if err != nil {
		ctx.Logger.Error("failed to setup marketplace", "error", err)
		return fmt.Errorf("failed to setup marketplace: %w", err)
	}
	c.p.Success("Marketplace setup completed")

	// Merge settings
	c.p.Section("Merging settings...")

	baseSettingsData, source, err := c.loadBaseSettings(resolver)
	if err != nil {
		return err
	}

	var baseSettings any
	if err := json.Unmarshal(baseSettingsData, &baseSettings); err != nil {
		return fmt.Errorf("invalid base settings: %w", err)
	}

	c.p.Verbose().Info(fmt.Sprintf("Loaded base settings from %s (%.1f KB)", source, float64(len(baseSettingsData))/1024))

	result, err := syncer.SyncSettings(ctx.Context, syncer.SettingsSyncInput{
		BaseSettings: baseSettings,
		ClaudeDir:    claudeDir,
		IsUpdate:     false,
		Printer:      c.p,
	})
	if err != nil {
		return err
	}

	// Copy global rules
	c.p.Section("Copying rules...")

	rulesResolver := registry.NewResolver(claudeDir)
	rulesPaths, rulesErr := rulesResolver.GetAllRulesPaths()
	if rulesErr == nil && len(rulesPaths) > 0 {
		c.p.Success(fmt.Sprintf("Found %d marketplace(s) with rules", len(rulesPaths)))
	} else {
		c.p.Verbose().Info("No marketplace rules found")
	}

	if c.p.IsJSON() {
		return c.printJSONOutput(result)
	}

	c.printNormalOutput(result)

	return nil
}

// loadBaseSettings loads base settings from registry or embedded fallback
func (c *InitCmd) loadBaseSettings(resolver registry.PathResolver) ([]byte, string, error) {
	settingsPath := resolver.BaseSettingsPath()

	if data, err := os.ReadFile(settingsPath); err == nil {
		c.p.Verbose().Info("Using base settings from marketplace")
		return data, "marketplace", nil
	}

	c.p.Verbose().Info("Using embedded base settings")
	return baseSettingsJSON, "embedded", nil
}

// printJSONOutput prints JSON formatted output
func (c *InitCmd) printJSONOutput(result *syncer.SettingsSyncResult) error {
	output := map[string]any{
		"created":      result.Created,
		"merged":       !result.Created,
		"settingsPath": result.SettingsPath,
		"stats":        result.MergeStats,
		"success":      true,
	}

	if result.BackupPath != "" {
		output["backupPath"] = result.BackupPath
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

// printNormalOutput prints human-readable output
func (c *InitCmd) printNormalOutput(result *syncer.SettingsSyncResult) {
	if !c.p.IsQuiet() {
		fmt.Println()
	}
	c.p.FinalSuccess("Initialization completed")

	c.p.Info(fmt.Sprintf("Path: %s", result.SettingsPath))

	if result.BackupPath != "" {
		c.p.Info(fmt.Sprintf("Backup: %s", filepath.Base(result.BackupPath)))
	}

	if result.MergeStats.KeysAdded > 0 || result.MergeStats.ArraysMerged > 0 {
		c.p.Info(fmt.Sprintf("Added %d new keys, merged %d arrays", result.MergeStats.KeysAdded, result.MergeStats.ArraysMerged))
	}
}

// checkAlreadyInitialized checks if claudectl init has already been run
func (c *InitCmd) checkAlreadyInitialized(claudeDir string) error {
	settingsPath := filepath.Join(claudeDir, "settings.json")
	marketplacesPath := filepath.Join(claudeDir, "plugins", "known_marketplaces.json")

	settingsExists := false
	if _, err := os.Stat(settingsPath); err == nil {
		settingsExists = true
	}

	marketplaceExists := false
	if data, err := os.ReadFile(marketplacesPath); err == nil {
		var marketplaces map[string]any
		if err := json.Unmarshal(data, &marketplaces); err == nil {
			if _, ok := marketplaces[registry.DefaultMarketplaceID]; ok {
				marketplaceExists = true
			}
		}
	}

	if settingsExists && marketplaceExists {
		return fmt.Errorf("claudectl has already been initialized.\n\nTo force re-initialization (this will reset all settings), use:\n  claudectl init --force")
	}

	return nil
}
