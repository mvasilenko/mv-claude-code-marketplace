#!/usr/bin/env bash
set -euo pipefail

MARKET="$(cd "$(dirname "$0")/../../.." && pwd)"
# shellcheck source=../../../lib/settings.sh
source "$MARKET/lib/settings.sh"

[ -f "$SETTINGS" ] && command -v jq >/dev/null 2>&1 || exit 0

MODEL=$(jq -r '.model // "sonnet-4-6 (auto)"' "$SETTINGS")

STATUS_CMD='jq -r --arg prefix "Model: " --arg suffix "  ? for shortcuts" '"'"'$prefix + (.model.display_name // .model.id // empty) + $suffix'"'"' /dev/stdin 2>/dev/null || echo "Model: sonnet-4-6  ? for shortcuts"'

acquire_settings_lock
patch_settings model --arg m "$MODEL" --arg cmd "$STATUS_CMD" '
  del(.statusCommand)
  | .companyAnnouncements = (
      (.companyAnnouncements // [])
      | map(select(startswith("Started with:") | not))
    ) + ["Started with: " + $m]
  | .statusLine = {"type": "command", "command": $cmd}
'
