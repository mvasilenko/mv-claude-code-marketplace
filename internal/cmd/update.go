package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/claude"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/config"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/lock"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/marketplace"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/rules"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/settings"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/style"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/syncer"
)

//go:embed base_settings.json
var updateBaseSettingsJSON []byte

// UpdateCmd updates marketplace plugins and settings.
type UpdateCmd struct {
	Force bool `help:"Force update regardless of interval check." short:"f"`

	p *printer.Printer `kong:"-"`
}

// marketplaceResult tracks the outcome of updating a single marketplace.
type marketplaceResult struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	UpdatesPulled bool   `json:"updatesPulled"`
	Error         string `json:"error,omitempty"`
}

// updateOutput is the JSON output structure for the update command.
type updateOutput struct {
	Marketplaces []marketplaceResult `json:"marketplaces,omitempty"`
	Reason       string              `json:"reason,omitempty"`
	Status       string              `json:"status"`
}

// Run executes the update command.
func (c *UpdateCmd) Run(ctx *Context) error {
	c.p = printer.New(ctx.CLI.JSON, ctx.CLI.Quiet, ctx.CLI.Verbose)

	claudeDir := ResolveClaudeDirFromContext(ctx)
	plat := platform.New()
	cfgMgr := config.NewManager(config.ResolveConfigDir(plat))

	cfg, err := cfgMgr.Load(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if auto-update is enabled (skip check when forced)
	if !c.Force && !cfg.AutoUpdate.Enabled {
		return c.outputSkipped("auto-update disabled")
	}

	// Check interval (skip check when forced)
	if !c.Force {
		interval, err := time.ParseDuration(cfg.AutoUpdate.CheckInterval)
		if err != nil {
			ctx.Logger.Warn("invalid check interval, using default",
				"interval", cfg.AutoUpdate.CheckInterval,
				"error", err,
			)
			interval = 3 * time.Hour
		}

		if time.Since(cfg.LastUpdateCheck) < interval {
			return c.outputSkipped("interval not elapsed")
		}
	}

	// Acquire file lock to prevent concurrent updates
	configDir := config.ResolveConfigDir(plat)
	lockPath := filepath.Join(configDir, ".update.lock")
	fileLock, err := lock.NewFileLock(lockPath)
	if err != nil {
		ctx.Logger.Error("failed to create file lock", "error", err)
		return c.outputSkipped("failed to create lock")
	}
	defer fileLock.Close() //nolint:errcheck // cleanup

	acquired, err := fileLock.TryLock()
	if err != nil {
		ctx.Logger.Error("failed to acquire file lock", "error", err)
		return c.outputSkipped("failed to acquire lock")
	}
	if !acquired {
		return c.outputSkipped("another update in progress")
	}
	defer fileLock.Unlock() //nolint:errcheck // cleanup

	// Update marketplaces
	c.p.Section("Updating marketplaces...")
	resolver := registry.NewResolver(claudeDir)
	mpService := marketplace.NewService(cfg.GetMarketplaceCachePath(claudeDir))
	settingsMgr := settings.NewManager(claudeDir)

	var mpResults []marketplaceResult

	for _, mpName := range cfg.Marketplaces {
		result := c.updateMarketplace(ctx, mpName, mpService, settingsMgr, resolver, claudeDir, cfg, plat)
		mpResults = append(mpResults, result)
	}

	// Merge settings
	c.p.Section("Merging settings...")

	baseSettingsData, source := c.loadBaseSettings(resolver)
	ctx.Logger.Info("loaded base settings",
		"source", source,
		"sizeBytes", len(baseSettingsData),
	)

	var baseSettings any
	if err := json.Unmarshal(baseSettingsData, &baseSettings); err != nil {
		ctx.Logger.Error("failed to parse base settings", "error", err)
		c.p.Warning(fmt.Sprintf("Failed to parse base settings: %v", err))
	} else {
		c.p.Verbose().Info(fmt.Sprintf("Loaded base settings from %s (%.1f KB)", source, float64(len(baseSettingsData))/1024))

		_, err := syncer.SyncSettings(ctx.Context, syncer.SettingsSyncInput{
			BaseSettings: baseSettings,
			ClaudeDir:    claudeDir,
			IsUpdate:     true,
			Printer:      c.p,
		})
		if err != nil {
			ctx.Logger.Error("failed to sync settings", "error", err)
			c.p.Warning(fmt.Sprintf("Failed to sync settings: %v", err))
		}
	}

	// Copy rules
	c.p.Section("Syncing rules...")
	rulesManager := rules.NewManager(resolver)
	rulesResult, err := rulesManager.CopyRules()
	if err != nil {
		ctx.Logger.Error("failed to copy rules", "error", err)
		c.p.Warning(fmt.Sprintf("Failed to sync rules: %v", err))
	} else if len(rulesResult.CopiedFiles) > 0 {
		c.p.Success(fmt.Sprintf("Synced %d rule(s)", len(rulesResult.CopiedFiles)))
	} else {
		c.p.Verbose().Info("No rule changes")
	}

	// Update last check time
	cfg.LastUpdateCheck = time.Now()
	if err := cfgMgr.Save(ctx.Context, cfg); err != nil {
		ctx.Logger.Error("failed to save config after update", "error", err)
	}

	// Output results
	return c.outputSuccess(mpResults)
}

// updateMarketplace updates a single marketplace: git pull, Claude registry update.
func (c *UpdateCmd) updateMarketplace(
	ctx *Context,
	mpName string,
	mpService *marketplace.Service,
	settingsMgr *settings.Manager,
	resolver *registry.Resolver,
	claudeDir string,
	cfg *config.Config,
	plat platform.Platform,
) marketplaceResult {
	result := marketplaceResult{Name: mpName, Status: "success"}

	// Resolve repo URL
	repoURL, err := settingsMgr.GetMarketplaceRepo(ctx.Context, mpName)
	if err != nil {
		// Fallback for the default marketplace
		if registry.IsDefaultMarketplace(mpName, "", "") {
			repoURL = fmt.Sprintf("%s/%s", registry.DefaultMarketplaceOrg, registry.DefaultMarketplaceRepo)
			ctx.Logger.Info("using default marketplace repo as fallback",
				"name", mpName,
				"repoURL", repoURL,
			)
		} else {
			ctx.Logger.Error("failed to get marketplace repo URL",
				"name", mpName,
				"error", err,
			)
			result.Status = "error"
			result.Error = err.Error()
			c.p.Warning(fmt.Sprintf("Skipping %s: %v", mpName, err))
			return result
		}
	}

	// Git pull
	cacheResult, err := mpService.EnsureCached(ctx.Context, mpName, repoURL, string(cfg.GitProtocol))
	if err != nil {
		ctx.Logger.Error("failed to update marketplace cache",
			"name", mpName,
			"error", err,
		)
		result.Status = "error"
		result.Error = err.Error()
		c.p.Warning(fmt.Sprintf("Failed to update %s: %v", mpName, err))
		return result
	}
	result.UpdatesPulled = cacheResult.UpdatesPulled

	if cacheResult.UpdatesPulled {
		c.p.Success(fmt.Sprintf("Updated %s (new changes pulled)", mpName))
	} else {
		c.p.Success(fmt.Sprintf("Updated %s (already up to date)", mpName))
	}

	// Update Claude registry
	cli, err := claude.New(claudeDir, plat)
	if err != nil {
		ctx.Logger.Warn("Claude CLI not available, skipping registry update",
			"name", mpName,
			"error", err,
		)
		return result
	}

	var cmdOutput io.Writer
	if !c.p.IsQuiet() {
		cmdOutput = style.NewCommandOutputWriter()
	}
	if err := cli.UpdateMarketplace(mpName, cmdOutput); err != nil {
		ctx.Logger.Warn("failed to update Claude registry",
			"name", mpName,
			"error", err,
		)
		c.p.Warning(fmt.Sprintf("Failed to update Claude registry for %s: %v", mpName, err))
	}

	return result
}

// loadBaseSettings loads base settings from the marketplace or embedded fallback.
func (c *UpdateCmd) loadBaseSettings(resolver registry.PathResolver) ([]byte, string) {
	settingsPath := resolver.BaseSettingsPath()

	if data, err := os.ReadFile(settingsPath); err == nil {
		return data, "marketplace"
	}

	return updateBaseSettingsJSON, "embedded"
}

// outputSkipped outputs a skipped result and returns nil.
func (c *UpdateCmd) outputSkipped(reason string) error {
	if c.p.IsJSON() {
		output := updateOutput{
			Reason: reason,
			Status: "skipped",
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	c.p.Info(fmt.Sprintf("Update skipped: %s", reason))
	return nil
}

// outputSuccess outputs the final success result.
func (c *UpdateCmd) outputSuccess(mpResults []marketplaceResult) error {
	if c.p.IsJSON() {
		output := updateOutput{
			Marketplaces: mpResults,
			Status:       "success",
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if !c.p.IsQuiet() {
		fmt.Println()
	}
	c.p.FinalSuccess("Update completed")

	return nil
}
