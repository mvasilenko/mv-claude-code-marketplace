---
name: jira-usage
description: Guidelines for using Jira MCP tools effectively.
---

# Jira Usage Guidelines

## Authentication
This plugin runs [`mcp-atlassian`](https://github.com/sooperset/mcp-atlassian) via `uvx` (install `uv` first: https://docs.astral.sh/uv/). It supports both Jira Cloud and Server/Data Center — `mcp-atlassian` auto-detects which one from `JIRA_URL`.

Set these in your shell profile before starting Claude Code:

**Jira Cloud** (`*.atlassian.net`):
```
export JIRA_URL="https://your-org.atlassian.net"
export JIRA_USERNAME="your.email@example.com"
export JIRA_API_TOKEN="your-api-token"
```

**Jira Server/Data Center** (self-hosted):
```
export JIRA_URL="https://your-jira-host.example.com"
export JIRA_PERSONAL_TOKEN="your-personal-access-token"
```

Only set the pair matching your instance — the plugin's `.mcp.json` references all four vars, but unset ones are ignored. Values are expanded by Claude Code from your shell environment at launch; none are stored in this repo.

## Guidelines
When working with Jira:
- Search or list existing issues before creating new ones to avoid duplicates
- Fetch the current state of an issue before updating it
