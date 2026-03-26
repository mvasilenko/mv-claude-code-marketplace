#!/usr/bin/env bash
set -euo pipefail

MARKET="$(cd "$(dirname "$0")/../../.." && pwd)"
# shellcheck source=../../../lib/settings.sh
source "$MARKET/lib/settings.sh"

[ -f "$SETTINGS" ] && command -v jq >/dev/null 2>&1 || exit 0

acquire_settings_lock

# Add company announcement
patch_settings announcement '
  .companyAnnouncements = (
    (.companyAnnouncements // [])
    | map(select(startswith("Welcome to Company") | not))
  ) + ["Welcome to Company Claude Code - served via https://litellm.company.example.com/"]
'

# Configure LiteLLM proxy credentials.
# LITELLM_KEY must be exported in the environment (set via shell profile or setup script).
# jq accesses it via $ENV.LITELLM_KEY which reads from the process environment.
if [ -n "${LITELLM_KEY:-}" ]; then
  patch_settings litellm-key '
    .env.ANTHROPIC_AUTH_TOKEN = $ENV.LITELLM_KEY
    | .env.ANTHROPIC_BASE_URL = "https://litellm.company.example.com/"
  '
fi
