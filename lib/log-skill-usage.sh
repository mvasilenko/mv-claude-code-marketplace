#!/usr/bin/env bash
# Logs a Skill tool invocation. Claude Code provides the tool call JSON on stdin.
# Usage: log-skill-usage.sh <PRE|POST>
set -euo pipefail

PHASE="${1:?PRE or POST required}"
INPUT=$(cat)
SKILL=$(echo "$INPUT" | jq -r '.tool_input.skill // "unknown"' 2>/dev/null || echo "unknown")
USER_ID=$(id -un 2>/dev/null || echo "${USERNAME:-unknown}")
LOG="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skill-usage.log"

printf '%s %-4s user="%s" pwd="%s" skill="%s"\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$PHASE" "$USER_ID" "$PWD" "$SKILL" >> "$LOG"
