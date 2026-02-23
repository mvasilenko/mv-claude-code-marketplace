package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/alecthomas/kong"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/config"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/settings"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/style"
)

// Version information (set by main)
var (
	Commit  = "none"
	Date    = "unknown"
	Version = "dev"
)

// CLI is the root command structure for claudectl.
type CLI struct {
	// Global flags
	ClaudeDir string `help:"Path to Claude config directory (overrides saved config and CLAUDE_CONFIG_DIR env var)" type:"path"`
	Debug     bool   `help:"Enable debug logging to file."`
	DebugFile string `help:"Enable debug logging to specified file (implies --debug)." type:"path"`
	JSON      bool   `help:"Output in JSON format."`
	Quiet     bool   `short:"q" help:"Suppress non-essential output."`
	Verbose   bool   `short:"v" help:"Enable verbose output."`

	// Commands
	Config      ConfigCmd      `cmd:"" help:"Manage claudectl configuration."`
	Init        InitCmd        `cmd:"" help:"Initialize Claude settings with base configuration."`
	Install     InstallCmd     `cmd:"" help:"Install required tools (Claude Code CLI, GitHub CLI)."`
	Marketplace MarketplaceCmd `cmd:"" help:"Manage marketplaces."`
	Update      UpdateCmd      `cmd:"" help:"Update marketplace plugins and settings."`
	Version     VersionCmd     `cmd:"" help:"Show version information."`
}

// Context provides shared context for command execution.
type Context struct {
	CLI     *CLI
	Context context.Context
	Logger  *slog.Logger
}

// VersionCmd displays version information.
type VersionCmd struct {
	p *printer.Printer `kong:"-"`
}

// Run executes the version command.
func (v *VersionCmd) Run(ctx *Context) error {
	v.p = printer.New(ctx.CLI.JSON, ctx.CLI.Quiet, ctx.CLI.Verbose)

	if v.p.IsJSON() {
		info := map[string]string{
			"commit":  Commit,
			"date":    Date,
			"version": Version,
		}
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("claudectl version %s\n", style.Success.Render(Version))
	if !v.p.IsQuiet() {
		fmt.Printf("  commit: %s\n", style.Dim.Render(Commit))
		fmt.Printf("  built:  %s\n", style.Dim.Render(Date))
	}
	return nil
}

// ConfigCmd manages claudectl configuration.
type ConfigCmd struct {
	Get        ConfigGetCmd        `cmd:"" help:"Get a configuration value."`
	ListAdd    ConfigListAddCmd    `cmd:"" name:"list-add" help:"Add a value to a list configuration."`
	ListRemove ConfigListRemoveCmd `cmd:"" name:"list-remove" help:"Remove a value from a list configuration."`
	Set        ConfigSetCmd        `cmd:"" help:"Set a configuration value."`
}

// ConfigGetCmd gets a configuration value.
type ConfigGetCmd struct {
	Key string `arg:"" help:"Configuration key (dot notation)."`
}

// Run executes config get.
func (c *ConfigGetCmd) Run(ctx *Context) error {
	cfgMgr := resolveConfigManager(ctx)
	val, err := cfgMgr.Get(ctx.Context, c.Key)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// ConfigSetCmd sets a configuration value.
type ConfigSetCmd struct {
	Key   string `arg:"" help:"Configuration key (dot notation)."`
	Value string `arg:"" help:"Value to set."`
}

// Run executes config set.
func (c *ConfigSetCmd) Run(ctx *Context) error {
	cfgMgr := resolveConfigManager(ctx)
	return cfgMgr.Set(ctx.Context, c.Key, c.Value)
}

// ConfigListAddCmd adds a value to a list configuration.
type ConfigListAddCmd struct {
	Key   string `arg:"" help:"Configuration key."`
	Value string `arg:"" help:"Value to add."`
}

// Run executes config list-add.
func (c *ConfigListAddCmd) Run(ctx *Context) error {
	cfgMgr := resolveConfigManager(ctx)
	return cfgMgr.ListAdd(ctx.Context, c.Key, c.Value)
}

// ConfigListRemoveCmd removes a value from a list configuration.
type ConfigListRemoveCmd struct {
	Key   string `arg:"" help:"Configuration key."`
	Value string `arg:"" help:"Value to remove."`
}

// Run executes config list-remove.
func (c *ConfigListRemoveCmd) Run(ctx *Context) error {
	cfgMgr := resolveConfigManager(ctx)
	return cfgMgr.ListRemove(ctx.Context, c.Key, c.Value)
}

// NewParser creates a new Kong parser for the CLI.
func NewParser(cli *CLI) (*kong.Kong, error) {
	return kong.New(cli,
		kong.Name("claudectl"),
		kong.Description("CLI for managing Claude Code plugins, marketplaces, and team standards"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
}

// ResolveClaudeDirFromContext resolves the Claude config directory with priority:
// 1. --claude-dir flag (from ctx.CLI.ClaudeDir)
// 2. Saved value in claudectl config
// 3. CLAUDE_CONFIG_DIR environment variable
// 4. Default: ~/.claude
func ResolveClaudeDirFromContext(ctx *Context) string {
	plat := platform.New()
	cfgMgr := config.NewManager(config.ResolveConfigDir(plat))
	cfg, err := cfgMgr.Load(ctx.Context)
	savedValue := ""
	if err == nil && cfg != nil {
		savedValue = cfg.ClaudeConfigDir
	}

	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	resolvedDir := settings.ResolveClaudeDir(ctx.CLI.ClaudeDir, savedValue)

	// If flag was explicitly provided and differs from saved value, persist it
	if ctx.CLI.ClaudeDir != "" && ctx.CLI.ClaudeDir != savedValue {
		cfg.ClaudeConfigDir = ctx.CLI.ClaudeDir
		_ = cfgMgr.Save(ctx.Context, cfg) //nolint:errcheck // best effort
	}

	return resolvedDir
}

// resolveConfigManager creates a config manager for the current context.
func resolveConfigManager(ctx *Context) *config.Manager {
	plat := platform.New()
	return config.NewManager(config.ResolveConfigDir(plat))
}
