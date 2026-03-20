# mv-claude-code-marketplace

Company Claude Code Marketplace — a shared collection of Claude Code plugins, skills, hooks, and rules used across Company engineering teams.

## Structure

- `plugins/` — Claude Code plugins (skills, commands, hooks)
- `rules/` — Global Claude Code rules installed into `~/.claude/rules/`

## Quick Setup

Prerequisites: `claude`, `jq`, `git`, `curl` installed.

Choose the setup script that matches your use case:

| Script | Config dir | Use case |
|--------|------------|----------|
| `setup.sh` | `~/.claude` | Default — personal and Company team accounts sharing one context |
| `setup-company-litellm.sh` | `~/.claude-litellm` | Company LiteLLM proxy (requires `LITELLM_KEY`) |

```
git clone git@github.com:mvasilenko/mv-claude-code-marketplace.git
cd mv-claude-code-marketplace
./setup.sh   # or ./setup-company-litellm.sh
```

Each script installs all relevant plugins from this marketplace plus the [`superpowers`](https://github.com/anthropics/claude-plugins-official) plugin, enables auto-update for both marketplaces, and sets up shell aliases.

## Shell Aliases

Both setup scripts add the following aliases to your shell profile (`~/.zshrc`, `~/.bashrc`, or `~/.profile`):

| Alias | Config dir | Description |
|-------|------------|-------------|
| `claude` | `~/.claude` | Default shared config |
| `claudel` | `~/.claude-litellm` | LiteLLM proxy config |

Each alias unsets `ANTHROPIC_AUTH_TOKEN` and `ANTHROPIC_BASE_URL` before launching to avoid leaking settings across configs.

After running setup, reload your shell profile or open a new terminal session.

## Manual Installation

Add the marketplace:

```
/plugin marketplace add mvasilenko/mv-claude-code-marketplace
```

Enable auto-update: `/plugin` > Marketplaces > mv-claude-code-marketplace > Enable auto-update.

Install a plugin:

```
/plugin install litellm-backend@mv-claude-code-marketplace
```

## Available Plugins

| Plugin | Description | Category |
|--------|-------------|----------|
| `rules-management` | Distributes and auto-syncs Claude Code rules to `~/.claude/rules/` | utility |
| `hello` | Simple test plugin that greets the user | utility |
| `litellm-backend` | Configures Claude Code to use the Company LiteLLM proxy — sets `ANTHROPIC_AUTH_TOKEN` from `LITELLM_KEY` and `ANTHROPIC_BASE_URL` to `https://litellm.company.example.com/` on session start. Requires `jq`. | utility |
| `model-routing` | Enforces cost-effective model selection when spawning subagents | utility |
| `model-display` | Shows current Claude model in session announcement and statusline | utility |
| `linear` | Configures the Linear MCP server for Claude Code | integration |
| `notion` | Configures the Notion MCP server for Claude Code | integration |
| `programming-skills` | Programming language skill guidelines (Go) | programming |

## Environment Variables

- `CLAUDE_CONFIG_DIR` — Claude Code config directory, defaults to `$HOME/.claude`
- `LITELLM_KEY` — (required for `litellm-backend` / `setup-company-litellm.sh`) LiteLLM proxy API key. Request one in the `#your-support-channel` Slack channel

## Contributing

Add new plugins, skills, hooks, or rules via PR against `main`. On any plugin change, bump the patch version in that plugin's `.claude-plugin/plugin.json` (e.g. `1.1.0` → `1.1.1`) — required for auto-update cache invalidation.
