#!/usr/bin/env bash
set -euo pipefail

MARKET="$(cd "$(dirname "$0")/../../.." && pwd)"
"$MARKET/lib/log-skill-usage.sh" POST
