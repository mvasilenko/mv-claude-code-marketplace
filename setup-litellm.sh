#!/usr/bin/env bash
set -euo pipefail

CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude-litellm}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=setup-lib.sh
source "$SCRIPT_DIR/setup-lib.sh"

check_prerequisites
check_litellm_key
backup_if_needed

add_or_update_marketplace "$MARKETPLACE_NAME" "$MARKETPLACE_REPO"
install_plugins rules-management hello litellm-backend model-routing model-display programming-skills usage-tracking
# Optional integrations (install manually if needed):
# install_plugins notion linear

add_or_update_marketplace "$SUPERPOWERS_MARKETPLACE" "$SUPERPOWERS_REPO"
install_or_update_plugin "superpowers@$SUPERPOWERS_MARKETPLACE"

enable_auto_update
verify_litellm_proxy
setup_shell_aliases

echo ""
echo "Setup complete. Start Claude Code with: claudel"
echo "NOTE: On first run Claude Code may show a /login error. Ignore it and restart - it will work on second run."
