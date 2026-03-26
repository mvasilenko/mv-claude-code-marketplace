#!/usr/bin/env bash
# Shared helpers for safely mutating ~/.claude/settings.json
# Usage:
#   source this file
#   call acquire_settings_lock (once per script, before any mutations)
#   call patch_settings <tmp-suffix> [jq-options...] '<jq-expression>'

SETTINGS="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/settings.json"
_SETTINGS_LOCK="$SETTINGS.lock"
_SETTINGS_LOCK_HELD=0
_SETTINGS_LOCK_TIMEOUT=10

_release_settings_lock() {
  rmdir "$_SETTINGS_LOCK" 2>/dev/null || true
  _SETTINGS_LOCK_HELD=0
}

# Acquire an exclusive file lock on settings.json. Released automatically on EXIT.
# Call once per script before any patch_settings calls.
# Exits with error if lock cannot be acquired within $_SETTINGS_LOCK_TIMEOUT seconds.
acquire_settings_lock() {
  [ "$_SETTINGS_LOCK_HELD" -eq 1 ] && return 0
  local waited=0
  while ! mkdir "$_SETTINGS_LOCK" 2>/dev/null; do
    sleep 0.1
    waited=$((waited + 1))
    if [ "$waited" -ge $((_SETTINGS_LOCK_TIMEOUT * 10)) ]; then
      echo "settings.sh: timed out waiting for lock on $SETTINGS" >&2
      exit 1
    fi
  done
  _SETTINGS_LOCK_HELD=1
  trap '_release_settings_lock' EXIT
}

# Apply a jq expression to settings.json (lock must already be held).
# First arg is the tmp file suffix; remaining args are forwarded to jq.
# Environment variables are accessible in expressions via $ENV.VAR_NAME.
# Usage: patch_settings <suffix> [jq-options...] '<jq-expression>'
patch_settings() {
  local suffix="${1:?tmp suffix required}"
  shift

  [ -f "$SETTINGS" ] || return 0
  command -v jq >/dev/null 2>&1 || return 0

  jq "$@" "$SETTINGS" > "$SETTINGS.tmp.$suffix" && mv "$SETTINGS.tmp.$suffix" "$SETTINGS"
}
