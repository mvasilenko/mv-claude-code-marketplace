---
name: notion-usage
description: Guidelines for using Notion MCP tools effectively.
---

# Notion Usage Guidelines

## Authentication
On first use, Claude Code will open a browser for Notion OAuth. Grant access to the workspaces you need. The token is stored in `~/.claude/settings.json` under `mcpServers.notion` and refreshed automatically.

## Guidelines
When working with Notion:
- Use notion-search to find pages before creating new ones to avoid duplicates
- Use notion-fetch to read page content before making updates
- Keep page content concise and structured
- Use notion-query-data-sources for database queries rather than fetching individual pages
- When creating pages, always specify a parent page or database
