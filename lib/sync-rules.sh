#!/usr/bin/env bash
# Syncs rules from a plugin's rules/ directory to ~/.claude/rules/
# Usage: sync-rules.sh <plugin-name>
set -euo pipefail

PLUGIN="${1:?plugin name required}"
MARKET="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$MARKET/plugins/$PLUGIN/rules"
DST="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/rules"

[ -d "$SRC" ] || exit 0
mkdir -p "$DST"

for f in "$SRC"/*.md; do
  [ -f "$f" ] || continue
  name=$(basename "$f")
  dst_name="${PLUGIN}-${name}"
  dst="$DST/$dst_name"
  hash_f="$DST/.${dst_name}.hash"
  new_hash=$(shasum -a 256 "$f" | awk '{print $1}')

  if [ -f "$dst" ]; then
    cur_hash=$(shasum -a 256 "$dst" | awk '{print $1}')
    stored=$(cat "$hash_f" 2>/dev/null || echo "")
    [ "$cur_hash" != "$stored" ] && continue
  fi

  cp "$f" "$dst"
  printf '%s' "$new_hash" > "$hash_f"
done
