#!/usr/bin/env bash
# Common functions for setup scripts. Set CLAUDE_DIR before sourcing this file.

MARKETPLACE_NAME="mv-claude-code-marketplace"
MARKETPLACE_REPO="mvasilenko/mv-claude-code-marketplace"
SUPERPOWERS_MARKETPLACE="claude-plugins-official"
SUPERPOWERS_REPO="anthropics/claude-plugins-official"

check_prerequisites() {
  for cmd in claude jq git curl; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      echo "ERROR: $cmd is required but not found. Install it first."
      exit 1
    fi
  done
}

backup_if_needed() {
  local known_marketplaces="$CLAUDE_DIR/plugins/known_marketplaces.json"
  if [ -d "$CLAUDE_DIR" ] && [ -f "$CLAUDE_DIR/settings.json" ]; then
    if ! jq -e ".[\"$MARKETPLACE_NAME\"]" "$known_marketplaces" >/dev/null 2>&1; then
      echo "Existing Claude Code config found at $CLAUDE_DIR with settings.json"
      echo "This will install Company marketplace plugins into this config."
      read -p "Continue? (Y/n) " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Nn]$ ]]; then
        echo "Aborted."
        exit 0
      fi
      local backup="$CLAUDE_DIR-backup-$(date +%Y%m%d-%H%M%S)"
      echo "Backing up to $backup"
      cp -r "$CLAUDE_DIR" "$backup"
    fi
  fi
}

add_or_update_marketplace() {
  local name="$1"
  local repo="$2"
  local known_marketplaces="$CLAUDE_DIR/plugins/known_marketplaces.json"
  if [ -f "$known_marketplaces" ] && jq -e ".[\"$name\"]" "$known_marketplaces" >/dev/null 2>&1; then
    CLAUDE_CONFIG_DIR="$CLAUDE_DIR" claude plugin marketplace update "$name"
  else
    CLAUDE_CONFIG_DIR="$CLAUDE_DIR" claude plugin marketplace add "$repo"
  fi
}

install_or_update_plugin() {
  local plugin="$1"
  local installed_plugins="$CLAUDE_DIR/plugins/installed_plugins.json"
  if [ -f "$installed_plugins" ] && jq -e ".plugins[\"$plugin\"]" "$installed_plugins" >/dev/null 2>&1; then
    CLAUDE_CONFIG_DIR="$CLAUDE_DIR" claude plugin update "$plugin" \
      || CLAUDE_CONFIG_DIR="$CLAUDE_DIR" claude plugin install "$plugin"
  else
    CLAUDE_CONFIG_DIR="$CLAUDE_DIR" claude plugin install "$plugin"
  fi
}

install_plugins() {
  for plugin in "$@"; do
    install_or_update_plugin "$plugin@$MARKETPLACE_NAME"
  done
}

enable_auto_update() {
  local known_marketplaces="$CLAUDE_DIR/plugins/known_marketplaces.json"
  if [ -f "$known_marketplaces" ]; then
    jq ".[\"$MARKETPLACE_NAME\"].autoUpdate = true | .[\"$SUPERPOWERS_MARKETPLACE\"].autoUpdate = true" \
      "$known_marketplaces" > "$known_marketplaces.tmp" \
      && mv "$known_marketplaces.tmp" "$known_marketplaces"
  fi
}

check_litellm_key() {
  if [ -z "${LITELLM_KEY:-}" ]; then
    echo "ERROR: LITELLM_KEY env var is not set."
    echo "Get your key at #your-support-channel and add to your shell profile:"
    echo "  export LITELLM_KEY=\"your-key-here\""
    exit 1
  fi
}

verify_litellm_proxy() {
  echo "Verifying LiteLLM proxy connection..."
  local http_code
  http_code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $LITELLM_KEY" https://litellm.company.example.com/health 2>/dev/null || echo "000")
  if [ "$http_code" = "200" ]; then
    echo "LiteLLM proxy is reachable."
  else
    echo "WARNING: LiteLLM proxy check failed (HTTP $http_code). Verify LITELLM_KEY is correct."
  fi
}

clean_litellm_settings() {
  local settings="$CLAUDE_DIR/settings.json"
  if [ -f "$settings" ]; then
    jq 'del(.companyAnnouncements) | del(.env.ANTHROPIC_AUTH_TOKEN) | del(.env.ANTHROPIC_BASE_URL) | if .env == {} then del(.env) else . end' \
      "$settings" > "$settings.tmp" && mv "$settings.tmp" "$settings"
    echo "Cleaned up any litellm settings from $settings."
  fi
}

configure_settings() {
  local settings="$CLAUDE_DIR/settings.json"
  if [ ! -f "$settings" ]; then
    echo "{}" > "$settings"
  fi
  jq '.env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC = "1"' \
    "$settings" > "$settings.tmp" && mv "$settings.tmp" "$settings"
  echo "Configured settings.json with base env vars."
}

setup_shell_aliases() {
  local profile
  case "${SHELL:-}" in
    */zsh)  profile="$HOME/.zshrc" ;;
    */bash) profile="$HOME/.bashrc" ;;
    *)      profile="$HOME/.profile" ;;
  esac

  # Remove existing alias definitions (idempotent)
  if [ -f "$profile" ]; then
    local tmp
    tmp=$(mktemp)
    grep -vE "^alias (claude|claudep|claudel)=" "$profile" > "$tmp" || true
    mv "$tmp" "$profile"
  fi

  {
    echo ""
    echo "# Claude Code config aliases"
    echo "alias claude='unset ANTHROPIC_AUTH_TOKEN; unset ANTHROPIC_BASE_URL; CLAUDE_CONFIG_DIR=~/.claude command claude'"
    echo "alias claudel='unset ANTHROPIC_AUTH_TOKEN; unset ANTHROPIC_BASE_URL; CLAUDE_CONFIG_DIR=~/.claude-litellm command claude'"
  } >> "$profile"

  echo "Aliases added to $profile. Run: source $profile"
}
