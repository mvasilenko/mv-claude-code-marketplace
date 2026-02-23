package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/claude"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/config"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/fetcher"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/marketplace"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/settings"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/style"
)

// MarketplaceCmd manages marketplaces
type MarketplaceCmd struct {
	Add    *MarketplaceAddCmd    `cmd:"" help:"Add a marketplace from GitHub or local path."`
	List   *MarketplaceListCmd   `cmd:"" aliases:"ls" help:"List installed marketplaces."`
	Remove *MarketplaceRemoveCmd `cmd:"" aliases:"rm" help:"Remove a marketplace."`
}

// MarketplaceAddCmd adds a marketplace
type MarketplaceAddCmd struct {
	Ext        []string `help:"Include plugins from plugins-ext/ for specified namespace (can be repeated)." short:"e"`
	Force      bool     `short:"f" help:"Force replace existing marketplace with the same name."`
	Repository string   `arg:"" required:"" help:"GitHub repository (owner/repo) or local path."`
	SkipUpdate bool     `short:"s" help:"Skip calling 'claude plugin marketplace update'."`

	p *printer.Printer `kong:"-"`
}

// MarketplaceRemoveCmd removes a marketplace
type MarketplaceRemoveCmd struct {
	Marketplace string `arg:"" required:"" help:"Marketplace name to remove."`
	SkipUpdate  bool   `short:"s" help:"Skip calling 'claude plugin marketplace remove'."`

	p *printer.Printer `kong:"-"`
}

// MarketplaceListCmd lists installed marketplaces
type MarketplaceListCmd struct {
	p *printer.Printer `kong:"-"`
}

// MarketplaceAddOutput is the JSON output format
type MarketplaceAddOutput struct {
	Error          string            `json:"error,omitempty"`
	Marketplace    *MarketplaceInfo  `json:"marketplace,omitempty"`
	PluginsEnabled []string          `json:"pluginsEnabled,omitempty"`
	Success        bool              `json:"success"`
	UpdatedFiles   map[string]string `json:"updatedFiles,omitempty"`
}

// MarketplaceInfo contains marketplace details
type MarketplaceInfo struct {
	LocalPath    string `json:"localPath,omitempty"`
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	PluginsCount int    `json:"pluginsCount"`
	Repo         string `json:"repo,omitempty"`
	SourceType   string `json:"sourceType"`
	Version      string `json:"version"`
}

// MarketplaceRemoveOutput is the JSON output format for remove command
type MarketplaceRemoveOutput struct {
	Error           string            `json:"error,omitempty"`
	Marketplace     string            `json:"marketplace,omitempty"`
	PluginsDisabled []string          `json:"pluginsDisabled,omitempty"`
	Success         bool              `json:"success"`
	UpdatedFiles    map[string]string `json:"updatedFiles,omitempty"`
}

// MarketplaceListOutput is the JSON output format for list command
type MarketplaceListOutput struct {
	Error        string                `json:"error,omitempty"`
	Marketplaces []MarketplaceListItem `json:"marketplaces"`
}

// MarketplaceListItem represents a single marketplace in list output
type MarketplaceListItem struct {
	Error       string `json:"error,omitempty"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	PluginCount int    `json:"pluginCount"`
	Version     string `json:"version"`
}

// Run executes the marketplace add command
func (m *MarketplaceAddCmd) Run(ctx *Context) error {
	m.p = printer.New(ctx.CLI.JSON, ctx.CLI.Quiet, ctx.CLI.Verbose)

	plat := platform.New()
	cfgMgr := config.NewManager(config.ResolveConfigDir(plat))
	cfg, err := cfgMgr.Load(ctx.Context)
	if err != nil {
		return m.outputError(err)
	}

	claudeDir := ResolveClaudeDirFromContext(ctx)
	cachePath := cfg.GetMarketplaceCachePath(claudeDir)
	mpService := marketplace.NewService(cachePath)

	m.p.Section("Adding marketplace...")

	m.Repository = strings.TrimSpace(m.Repository)

	var owner, repo string
	var localPath string
	if fetcher.IsLocalPath(m.Repository) {
		localPath = m.Repository
		m.p.Info(fmt.Sprintf("Source: %s", m.Repository))
	} else {
		owner, repo, err = fetcher.ParseRepository(m.Repository)
		if err != nil {
			return m.outputError(err)
		}
		m.p.Info(fmt.Sprintf("Fetching from %s/%s", owner, repo))
	}

	// Determine extension namespaces
	extNamespaces := m.Ext
	if len(extNamespaces) == 0 && registry.IsDefaultMarketplace(repo, owner, repo) {
		extNamespaces = cfg.ExtensionNamespaces
	}

	input := marketplace.AddMarketplaceInput{
		ClaudeDir:     claudeDir,
		ConfigManager: cfgMgr,
		Ext:           extNamespaces,
		Force:         m.Force,
		GitProtocol:   string(cfg.GitProtocol),
		LocalPath:     localPath,
		Owner:         owner,
		Platform:      plat,
		Printer:       m.p,
		Repo:          repo,
		SkipRegister:  m.SkipUpdate,
	}

	if m.p.IsVerbose() {
		input.CommandOutput = style.NewCommandOutputWriter()
	}

	result, err := mpService.AddMarketplace(ctx.Context, input)
	if err != nil {
		return m.outputError(err)
	}

	mpLabel := result.Marketplace.Name
	if result.Marketplace.Version != "" {
		mpLabel += "@" + result.Marketplace.Version
	}
	m.p.Success(fmt.Sprintf("Marketplace %s added", mpLabel))

	// Save extension namespaces to config if provided for default marketplace
	if len(m.Ext) > 0 && registry.IsDefaultMarketplace(result.Marketplace.Name, owner, repo) {
		msg, err := cfgMgr.SaveExtensionNamespaces(ctx.Context, cfg, m.Ext)
		if err != nil {
			return m.outputError(err)
		}
		m.p.Verbose().Info(msg)
	}

	// Show enabled plugins
	if len(result.PluginsEnabled) > 0 {
		m.p.Section("Enabled plugins:")
		for _, pluginStr := range result.PluginsEnabled {
			m.p.Bullet(pluginStr)
		}
	}

	return m.outputSuccess(result, settings.NewManager(claudeDir), cfgMgr)
}

// outputSuccess outputs success message for add command
func (m *MarketplaceAddCmd) outputSuccess(result *marketplace.AddMarketplaceResult, settingsMgr *settings.Manager, cfgMgr *config.Manager) error {
	if m.p.IsJSON() {
		info := &MarketplaceInfo{
			Name:         result.Marketplace.Name,
			Owner:        result.Marketplace.Owner.Name,
			PluginsCount: len(result.PluginsEnabled),
			SourceType:   result.SourceType,
			Version:      result.Marketplace.Version,
		}

		if result.SourceType == "directory" {
			info.LocalPath = result.CachePath
		} else {
			info.Repo = result.CachePath
		}

		output := MarketplaceAddOutput{
			Marketplace:    info,
			PluginsEnabled: result.PluginsEnabled,
			Success:        true,
			UpdatedFiles: map[string]string{
				"config":   cfgMgr.GetConfigPath(),
				"settings": settingsMgr.GetSettingsPath(),
			},
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON output: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if !m.p.IsQuiet() {
		fmt.Println()
	}
	m.p.FinalSuccess("Marketplace added successfully!")

	return nil
}

// outputError outputs error message for add command
func (m *MarketplaceAddCmd) outputError(err error) error {
	if m.p.IsJSON() {
		output := MarketplaceAddOutput{
			Error:   err.Error(),
			Success: false,
		}
		data, marshalErr := json.MarshalIndent(output, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v (failed to marshal JSON: %v)\n", err, marshalErr)
			return err
		}
		fmt.Println(string(data))
	}

	return err
}

// Run executes the marketplace remove command
func (m *MarketplaceRemoveCmd) Run(ctx *Context) error {
	m.p = printer.New(ctx.CLI.JSON, ctx.CLI.Quiet, ctx.CLI.Verbose)

	if !m.p.IsJSON() {
		m.p.Section("Removing marketplace...")
	}

	claudeDir := ResolveClaudeDirFromContext(ctx)
	settingsMgr := settings.NewManager(claudeDir)

	if !m.p.IsJSON() {
		m.p.Info(fmt.Sprintf("Removing %s", m.Marketplace))
	}

	repo, disabledPlugins, err := settingsMgr.RemoveMarketplace(ctx.Context, m.Marketplace)
	if err != nil {
		return m.outputError(err)
	}

	if !m.p.IsJSON() {
		m.p.Verbose().Info(fmt.Sprintf("Removed from settings (repo: %s)", repo))
	}

	if !m.p.IsJSON() && len(disabledPlugins) > 0 {
		m.p.Section("Disabled plugins:")
		for _, plugin := range disabledPlugins {
			m.p.Bullet(plugin)
		}
	}

	// Remove from claudectl config
	plat := platform.New()
	cfgMgr := config.NewManager(config.ResolveConfigDir(plat))

	err = cfgMgr.ListRemove(ctx.Context, "marketplaces", m.Marketplace)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return m.outputError(fmt.Errorf("failed to update config: %w", err))
	}

	// Clean up plugin cache
	cfg, err := cfgMgr.Load(ctx.Context)
	if err != nil {
		return m.outputError(fmt.Errorf("failed to load config: %w", err))
	}
	cachePath := cfg.GetMarketplaceCachePath(claudeDir)
	mpService := marketplace.NewService(cachePath)
	mpService.DeletePluginCache(ctx.Context, claudeDir, m.Marketplace)

	// Trigger Claude CLI remove (unless --skip-update)
	if !m.SkipUpdate {
		if !m.p.IsJSON() {
			m.p.Info("Unregistering from Claude...")
		}
		if err := m.callClaudeRemove(m.Marketplace, claudeDir); err != nil {
			if !m.p.IsJSON() {
				m.p.Warning(fmt.Sprintf("failed to unregister from Claude: %v", err))
			}
		}
	}

	return m.outputSuccess(m.Marketplace, disabledPlugins, settingsMgr, cfgMgr)
}

// callClaudeRemove executes claude plugin marketplace remove
func (m *MarketplaceRemoveCmd) callClaudeRemove(marketplaceName, claudeDir string) error {
	plat := platform.New()
	cli, err := claude.New(claudeDir, plat)
	if err != nil {
		return err
	}

	if m.p.IsVerbose() {
		output := style.NewCommandOutputWriter()
		err = cli.RemoveMarketplace(marketplaceName, output)
		output.Flush()
		return err
	}
	return cli.RemoveMarketplace(marketplaceName, nil)
}

// outputSuccess outputs success message for remove command
func (m *MarketplaceRemoveCmd) outputSuccess(mpName string, disabledPlugins []string, settingsMgr *settings.Manager, cfgMgr *config.Manager) error {
	if m.p.IsJSON() {
		output := MarketplaceRemoveOutput{
			Marketplace:     mpName,
			PluginsDisabled: disabledPlugins,
			Success:         true,
			UpdatedFiles: map[string]string{
				"config":   cfgMgr.GetConfigPath(),
				"settings": settingsMgr.GetSettingsPath(),
			},
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON output: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if !m.p.IsQuiet() {
		fmt.Println()
	}
	m.p.FinalSuccess("Marketplace removed successfully!")

	return nil
}

// outputError outputs error message for remove command
func (m *MarketplaceRemoveCmd) outputError(err error) error {
	if m.p.IsJSON() {
		output := MarketplaceRemoveOutput{
			Error:   err.Error(),
			Success: false,
		}
		data, marshalErr := json.MarshalIndent(output, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v (failed to marshal JSON: %v)\n", err, marshalErr)
			return err
		}
		fmt.Println(string(data))
	}

	return err
}

// Run executes the marketplace list command
func (m *MarketplaceListCmd) Run(ctx *Context) error {
	m.p = printer.New(ctx.CLI.JSON, ctx.CLI.Quiet, ctx.CLI.Verbose)

	plat := platform.New()
	cfgMgr := config.NewManager(config.ResolveConfigDir(plat))
	cfg, err := cfgMgr.Load(ctx.Context)
	if err != nil {
		return m.outputError(fmt.Errorf("failed to load config: %w", err))
	}

	claudeDir := ResolveClaudeDirFromContext(ctx)
	resolver := registry.NewResolver(claudeDir)

	items := make([]MarketplaceListItem, 0)
	for _, name := range cfg.Marketplaces {
		item := MarketplaceListItem{Name: name}

		path, found, err := resolver.GetInstallLocation(name)
		if err != nil {
			item.Error = fmt.Sprintf("failed to resolve path: %v", err)
			items = append(items, item)
			continue
		}
		if !found {
			item.Error = "not installed"
			items = append(items, item)
			continue
		}
		item.Path = path

		mp, err := fetcher.FetchMarketplaceFromPath(path)
		if err != nil {
			item.Error = fmt.Sprintf("failed to read marketplace.json: %v", err)
			items = append(items, item)
			continue
		}

		item.PluginCount = len(mp.Plugins)
		item.Version = mp.Version
		items = append(items, item)
	}

	return m.outputResults(items)
}

// outputResults outputs the marketplace list
func (m *MarketplaceListCmd) outputResults(items []MarketplaceListItem) error {
	if m.p.IsJSON() {
		output := MarketplaceListOutput{
			Marketplaces: items,
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON output: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if len(items) == 0 {
		m.p.Info("No marketplaces installed")
		return nil
	}

	if m.p.IsQuiet() {
		for _, item := range items {
			fmt.Println(item.Name)
		}
		return nil
	}

	// Table output
	nameWidth := 4
	verWidth := 7
	plugWidth := 7
	pathWidth := 4

	for _, item := range items {
		if len(item.Name) > nameWidth {
			nameWidth = len(item.Name)
		}
		ver := item.Version
		if item.Error != "" {
			ver = "-"
		}
		if len(ver) > verWidth {
			verWidth = len(ver)
		}
		path := item.Path
		if item.Error != "" {
			path = item.Error
		}
		if len(path) > pathWidth {
			pathWidth = len(path)
		}
	}

	fmt.Printf("%-*s  %-*s  %-*s  %s\n", nameWidth, style.Bold.Render("NAME"), verWidth, style.Bold.Render("VERSION"), plugWidth, style.Bold.Render("PLUGINS"), style.Bold.Render("PATH"))

	for _, item := range items {
		ver := item.Version
		plugins := fmt.Sprintf("%d", item.PluginCount)
		path := item.Path

		if item.Error != "" {
			ver = "-"
			plugins = "-"
			path = fmt.Sprintf("(error: %s)", item.Error)
		}

		fmt.Printf("%-*s  %-*s  %-*s  %s\n", nameWidth, item.Name, verWidth, ver, plugWidth, plugins, path)
	}

	return nil
}

// outputError outputs error message for list command
func (m *MarketplaceListCmd) outputError(err error) error {
	if m.p.IsJSON() {
		output := MarketplaceListOutput{
			Error:        err.Error(),
			Marketplaces: []MarketplaceListItem{},
		}
		data, marshalErr := json.MarshalIndent(output, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v (failed to marshal JSON: %v)\n", err, marshalErr)
			return err
		}
		fmt.Println(string(data))
	}

	return err
}
