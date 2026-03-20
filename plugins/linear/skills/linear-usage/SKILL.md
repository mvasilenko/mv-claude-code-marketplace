---
name: linear-usage
description: Guidelines for using Linear MCP tools effectively.
---

# Linear Usage Guidelines

## Authentication
On first use, Claude Code will open a browser for Linear OAuth. Authorize access to your workspace. The token is stored in `~/.claude/settings.json` under `mcpServers.linear` and refreshed automatically.

## Guidelines
When working with Linear:
- Use list_issues with filters rather than fetching all issues
- Use list_issue_statuses to understand the workflow before updating issue status
- Always check existing issues before creating duplicates
- When updating issues, fetch the current state first with get_issue
- Use list_teams to discover team IDs before creating issues
