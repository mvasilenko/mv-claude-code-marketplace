---
name: confluence-usage
description: Guidelines for using Confluence MCP tools effectively.
---

# Confluence Usage Guidelines

## Authentication
This plugin runs [`mcp-atlassian`](https://github.com/sooperset/mcp-atlassian) via `uvx` (install `uv` first: https://docs.astral.sh/uv/). It supports both Confluence Cloud and Server/Data Center — `mcp-atlassian` auto-detects which one from `CONFLUENCE_URL`.

Set these in your shell profile before starting Claude Code:

**Confluence Cloud** (`*.atlassian.net/wiki`):
```
export CONFLUENCE_URL="https://your-org.atlassian.net/wiki"
export CONFLUENCE_USERNAME="your.email@example.com"
export CONFLUENCE_API_TOKEN="your-api-token"
```

**Confluence Server/Data Center** (self-hosted):
```
export CONFLUENCE_URL="https://your-confluence-host.example.com"
export CONFLUENCE_PERSONAL_TOKEN="your-personal-access-token"
```

Only set the pair matching your instance — the plugin's `.mcp.json` references all four vars, but unset ones are ignored. Values are expanded by Claude Code from your shell environment at launch; none are stored in this repo.

## Guidelines
When working with Confluence:
- Search existing pages before creating new ones to avoid duplicates
- Fetch the current state of a page before updating it
