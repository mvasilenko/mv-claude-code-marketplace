#!/usr/bin/env bash
set -euo pipefail

# shellcheck disable=SC2034  # used by functions in setup-lib.sh
CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=setup-lib.sh
source "$SCRIPT_DIR/setup-lib.sh"

check_prerequisites
backup_if_needed

add_or_update_marketplace "$MARKETPLACE_NAME" "$MARKETPLACE_REPO"
install_plugins rules-management hello model-routing programming-skills usage-tracking karpathy-guidelines
# Optional integrations (install manually if needed):
# install_plugins notion linear litellm-backend

add_or_update_marketplace "$SUPERPOWERS_MARKETPLACE" "$SUPERPOWERS_REPO"
install_or_update_plugin "superpowers@$SUPERPOWERS_MARKETPLACE"

# Required by the skill-eval rule (always-on skills)
install_external_plugin DietrichGebert/ponytail ponytail ponytail
install_external_plugin JuliusBrussee/caveman caveman caveman
install_external_plugin ayghri/i-have-adhd i-have-adhd i-have-adhd

enable_auto_update
clean_litellm_settings
configure_settings
install_ccstatusline
setup_shell_aliases

echo ""
echo "Setup complete. Start Claude Code with: claude"
