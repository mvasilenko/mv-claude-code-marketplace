#!/usr/bin/env bash
# Exit 0 if every fixture produces the expected hook exit code; exit 1 otherwise.
set -uo pipefail

HOOK="$(cd "$(dirname "$0")" && pwd)/pre-tool-use.sh"
FIX="$(cd "$(dirname "$0")" && pwd)/fixtures"
FAIL=0

run_case() {
  local fixture="$1"
  local expected="$2"
  local stderr_file
  stderr_file=$(mktemp)
  local actual=0
  "$HOOK" < "$FIX/$fixture" 2>"$stderr_file" >/dev/null || actual=$?
  if [ "$actual" != "$expected" ]; then
    echo "FAIL $fixture: expected exit=$expected got exit=$actual"
    echo "  stderr: $(cat "$stderr_file")"
    FAIL=1
  else
    echo "PASS $fixture (exit=$actual)"
  fi
  rm -f "$stderr_file"
}

# Deny cases (exit 2)
run_case explore-no-model.json 2
run_case explore-opus.json 2

# Allow cases (exit 0)
run_case explore-haiku.json 0
run_case explore-sonnet.json 0
run_case general-purpose-no-model.json 0

# Fail-open on malformed input — documents intentional posture.
run_case malformed-input.json 0

exit "$FAIL"
