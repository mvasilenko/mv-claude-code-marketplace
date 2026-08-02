#!/usr/bin/env bash
# Validates repo structure invariants: plugin.json fields/naming, marketplace.json
# consistency, and SKILL.md frontmatter. Run from repo root.
set -euo pipefail

fail=0

for plugin_json in plugins/*/.claude-plugin/plugin.json; do
  dir_name=$(basename "$(dirname "$(dirname "$plugin_json")")")

  if ! jq empty "$plugin_json" 2>/dev/null; then
    echo "ERROR: $plugin_json is not valid JSON" >&2
    fail=1
    continue
  fi

  name=$(jq -r '.name // empty' "$plugin_json")
  version=$(jq -r '.version // empty' "$plugin_json")
  description=$(jq -r '.description // empty' "$plugin_json")

  if [[ -z "$name" || -z "$version" || -z "$description" ]]; then
    echo "ERROR: $plugin_json missing required field(s): name/description/version" >&2
    fail=1
  fi

  if [[ "$name" != "$dir_name" ]]; then
    echo "ERROR: $plugin_json name '$name' does not match directory '$dir_name'" >&2
    fail=1
  fi
done

marketplace=".claude-plugin/marketplace.json"
if ! jq empty "$marketplace" 2>/dev/null; then
  echo "ERROR: $marketplace is not valid JSON" >&2
  fail=1
else
  while IFS=$'\t' read -r name source; do
    plugin_json="${source#./}/.claude-plugin/plugin.json"
    if [[ ! -f "$plugin_json" ]]; then
      echo "ERROR: $marketplace references '$source' but $plugin_json is missing" >&2
      fail=1
      continue
    fi
    if ! actual_name=$(jq -r '.name // empty' "$plugin_json" 2>/dev/null); then
      echo "ERROR: $marketplace references '$source' but $plugin_json is not valid JSON" >&2
      fail=1
      continue
    fi
    if [[ "$actual_name" != "$name" ]]; then
      echo "ERROR: $marketplace entry '$name' does not match $plugin_json name '$actual_name'" >&2
      fail=1
    fi
  done < <(jq -r '.plugins[] | [.name, .source] | @tsv' "$marketplace")
fi

while IFS= read -r -d '' skill; do
  if [[ "$(sed -n '1p' "$skill")" != "---" ]]; then
    echo "ERROR: $skill has no frontmatter (must start with ---)" >&2
    fail=1
    continue
  fi
  frontmatter=$(sed -n '/^---$/,/^---$/p' "$skill" | sed '1d;$d')
  if ! grep -q '^name:' <<<"$frontmatter" || ! grep -q '^description:' <<<"$frontmatter"; then
    echo "ERROR: $skill missing 'name:' or 'description:' in frontmatter" >&2
    fail=1
  fi
done < <(find plugins -name SKILL.md -print0)

# CLAUDE.md requires a patch version bump on any plugin change. Only checked
# when a base ref is given (e.g. in CI, against the MR merge-base).
if [[ -n "${VALIDATE_BASE_REF:-}" ]]; then
  changed_plugins=$(git diff --name-only "$VALIDATE_BASE_REF"..HEAD -- plugins \
    | awk -F/ '{print $2}' | sort -u)
  while IFS= read -r plugin; do
    [[ -n "$plugin" ]] || continue
    plugin_json="plugins/$plugin/.claude-plugin/plugin.json"
    [[ -f "$plugin_json" ]] || continue

    if git diff --quiet "$VALIDATE_BASE_REF"..HEAD -- "$plugin_json"; then
      echo "ERROR: plugins/$plugin changed but $plugin_json was not updated (bump version)" >&2
      fail=1
      continue
    fi

    if ! git cat-file -e "$VALIDATE_BASE_REF:$plugin_json" 2>/dev/null; then
      continue # newly added plugin, no prior version to compare
    fi

    old_version=$(git show "$VALIDATE_BASE_REF:$plugin_json" | jq -r '.version // empty')
    new_version=$(jq -r '.version // empty' "$plugin_json")
    if [[ "$old_version" == "$new_version" ]]; then
      echo "ERROR: plugins/$plugin changed but version in $plugin_json was not bumped ($old_version)" >&2
      fail=1
    fi
  done <<<"$changed_plugins"
fi

exit "$fail"
