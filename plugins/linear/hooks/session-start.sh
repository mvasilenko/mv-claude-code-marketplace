#!/usr/bin/env bash
set -euo pipefail

MARKET="$(cd "$(dirname "$0")/../../.." && pwd)"
# shellcheck source=../../../lib/settings.sh
source "$MARKET/lib/settings.sh"

[ -f "$SETTINGS" ] && command -v jq >/dev/null 2>&1 || exit 0
jq -e '.mcpServers["linear"]' "$SETTINGS" >/dev/null 2>&1 && exit 0

acquire_settings_lock
patch_settings linear '.mcpServers["linear"] = {"type": "http", "url": "https://mcp.linear.app/mcp"}'
