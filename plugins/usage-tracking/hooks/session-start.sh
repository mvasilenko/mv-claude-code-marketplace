#!/usr/bin/env bash
# Seeds the statusline "Skills:" cache with always-on skills that hooks inject (never via the Skill tool).
set -euo pipefail

CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
SESSION_ID=$(jq -r '.session_id // empty' 2>/dev/null || true)
[ -n "$SESSION_ID" ] || exit 0

CACHE_DIR="$HOME/.cache/ccstatusline/skills"
FILE="$CACHE_DIR/skills-$SESSION_ID.jsonl"
mkdir -p "$CACHE_DIR"

seed() {
  grep -qs "\"$1\"" "$FILE" || jq -nc --arg skill "$1" '{skill: $skill}' >> "$FILE"
}

[ -d "$CLAUDE_DIR/plugins/cache/caveman" ] && seed caveman
[ -d "$CLAUDE_DIR/plugins/cache/ponytail" ] && seed ponytail
{ [ -d "$CLAUDE_DIR/plugins/cache/i-have-adhd" ] && [ -f "$CLAUDE_DIR/.i-have-adhd-always" ]; } && seed i-have-adhd
exit 0
