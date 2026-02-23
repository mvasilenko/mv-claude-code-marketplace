package cmd

import (
	"fmt"
	"runtime"
	"slices"
	"sort"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/config"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/installer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/software"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/style"
)

// InstallCmd handles tool installation
type InstallCmd struct {
	Skip []string `help:"Skip specific tools (e.g., --skip gh)"`
	Tool string   `arg:"" optional:"" default:"all" help:"Tool to install: claude, gh, or all (default: all)"`

	p *printer.Printer `kong:"-"`
}

// Run executes the install command
func (c *InstallCmd) Run(ctx *Context) error {
	c.p = printer.New(ctx.CLI.JSON, ctx.CLI.Quiet, ctx.CLI.Verbose)
	log := logger.FromContext(ctx.Context)

	log.Info("starting install command", "tool", c.Tool, "skip", c.Skip)

	plat := platform.New()
	cfgMgr := config.NewManager(config.ResolveConfigDir(plat))

	// Resolve Claude directory and create resolver
	claudeDir := ResolveClaudeDirFromContext(ctx)
	resolver := registry.NewResolver(claudeDir)
	swLoader := software.NewLoader(resolver)

	// Load software config
	swCfg, err := swLoader.Load()
	if err != nil {
		log.Error("failed to load software config", "error", err)
		return fmt.Errorf("failed to load software config: %w", err)
	}
	log.Debug("loaded software config", "softwareCount", len(swCfg.Software))

	// Load user config for exclusions
	userCfg, err := cfgMgr.Load(ctx.Context)
	if err != nil {
		log.Error("failed to load user config", "error", err)
		return fmt.Errorf("failed to load user config: %w", err)
	}

	// Parse software to install
	softwareList, err := c.parseSoftware(swCfg, userCfg)
	if err != nil {
		log.Error("failed to parse software list", "error", err)
		return err
	}

	log.Info("resolved software to install", "count", len(softwareList))

	if len(softwareList) == 0 {
		c.p.FinalSuccess("No tools to install")
		return nil
	}

	// Create installer
	archiver := platform.NewArchiver()
	inst := installer.New(plat, archiver, c.p)

	// Install each software
	var hasError bool
	for _, sw := range softwareList {
		log.Info("installing software", "id", sw.ID, "displayName", sw.DisplayName)
		result := inst.InstallSoftware(ctx.Context, sw)

		// Print script output when verbose or when there's an error
		if result.ScriptOutput != "" && (c.p.IsVerbose() || result.Error != nil) {
			w := style.NewCommandOutputWriter()
			_, _ = w.Write([]byte(result.ScriptOutput))
			w.Flush()
		}

		if result.Error != nil {
			log.Error("software installation failed", "id", sw.ID, "error", result.Error)
			hasError = true
			if c.Tool != "all" {
				return result.Error
			}
		} else {
			log.Info("software installation completed", "id", sw.ID, "version", result.Version, "alreadyInstalled", result.AlreadyInstalled)
		}
	}

	// Update lastInstalledTools
	allInstalledIDs := c.getAllInstalledTools(plat, swCfg)
	log.Debug("updating last installed tools", "installedIDs", allInstalledIDs)
	userCfg.UpdateLastInstalledTools(allInstalledIDs)
	if err := cfgMgr.Save(ctx.Context, userCfg); err != nil {
		log.Warn("failed to save config", "error", err)
		c.p.Warning(fmt.Sprintf("failed to update config: %v", err))
	}

	if hasError && c.Tool == "all" {
		return fmt.Errorf("some tools failed to install")
	}

	return nil
}

// parseSoftware converts tool argument to list of software to install
func (c *InstallCmd) parseSoftware(swCfg *software.Config, userCfg *config.Config) ([]*software.Software, error) {
	if c.Tool == "all" {
		toolIDs := swCfg.GetAllIDs()

		var result []*software.Software
		for _, toolID := range toolIDs {
			sw, err := swCfg.GetSoftware(toolID)
			if err != nil {
				continue
			}

			if c.isSkipped(sw.ID) {
				continue
			}

			if userCfg.IsToolExcluded(sw.ID) {
				continue
			}

			if !sw.SupportsCurrentPlatform(runtime.GOOS) {
				continue
			}

			result = append(result, sw)
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].Priority < result[j].Priority
		})

		return result, nil
	}

	// Install specific tool
	sw, err := swCfg.GetSoftware(c.Tool)
	if err != nil {
		availableIDs := swCfg.GetAllIDs()
		return nil, fmt.Errorf("unknown tool: %s (available: %v)", c.Tool, availableIDs)
	}

	if !sw.SupportsCurrentPlatform(runtime.GOOS) {
		return nil, fmt.Errorf("tool %s is not supported on %s", sw.ID, runtime.GOOS)
	}

	return []*software.Software{sw}, nil
}

// getAllInstalledTools returns IDs of all currently installed tools
func (c *InstallCmd) getAllInstalledTools(plat platform.Platform, swCfg *software.Config) []string {
	var installed []string
	for _, sw := range swCfg.Software {
		binaryName := sw.GetBinaryName(runtime.GOOS)
		if plat.CommandExists(binaryName) || plat.CommandExistsInBinDir(binaryName) {
			installed = append(installed, sw.ID)
		}
	}
	return installed
}

// isSkipped checks if a tool ID is in the skip list
func (c *InstallCmd) isSkipped(toolID string) bool {
	return slices.Contains(c.Skip, toolID)
}
