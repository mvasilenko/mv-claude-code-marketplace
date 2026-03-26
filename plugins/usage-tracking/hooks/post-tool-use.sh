#!/usr/bin/env bash
# Claude Code provides the tool call JSON on stdin for PostToolUse hooks.
set -euo pipefail

INPUT=$(cat)
SKILL=$(echo "$INPUT" | jq -r '.tool_input.skill // "unknown"' 2>/dev/null || echo "unknown")
USER_ID=$(id -un 2>/dev/null || echo "${USERNAME:-unknown}")
LOG="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skill-usage.log"

echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) POST user=\"$USER_ID\" pwd=\"$PWD\" skill=\"$SKILL\"" >> "$LOG"
