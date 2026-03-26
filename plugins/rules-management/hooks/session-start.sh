#!/usr/bin/env bash
set -euo pipefail

MARKET="$(cd "$(dirname "$0")/../../.." && pwd)"

# Sync rules to ~/.claude/rules/
"$MARKET/lib/sync-rules.sh" rules-management

# Remove legacy unscoped rule files if they match plugin source
DST="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/rules"
SRC="$MARKET/plugins/rules-management/rules"

for legacy_name in critical-thinking.md skill-eval.md; do
  legacy="$DST/$legacy_name"
  [ -f "$legacy" ] || continue
  src_for_legacy="$SRC/$legacy_name"
  [ -f "$src_for_legacy" ] || continue
  legacy_hash=$(shasum -a 256 "$legacy" | awk '{print $1}')
  src_hash=$(shasum -a 256 "$src_for_legacy" | awk '{print $1}')
  [ "$legacy_hash" = "$src_hash" ] && rm "$legacy"
done
