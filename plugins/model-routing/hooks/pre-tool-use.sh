#!/usr/bin/env bash
# PreToolUse hook for the subagent-spawning tool.
# Enforces: when subagent_type == "Explore", the model parameter must be
# explicitly set to "haiku" or "sonnet". Missing model (parent-inheritance
# trap) or "opus" are denied.
#
# Contract:
#   stdin   — JSON with .tool_name and .tool_input (from Claude Code).
#   exit 0  — allow the tool call to proceed.
#   exit 2  — deny the tool call; stderr is shown to the model.
#
# Fail-open on malformed input: if jq is missing or the JSON is unparseable,
# SUBAGENT falls through to "" and the hook exits 0 (allow). This is
# intentional — the hook is a cost-routing nudge, not a security gate.
set -uo pipefail

INPUT=$(cat)

SUBAGENT=$(printf '%s' "$INPUT" | jq -r '.tool_input.subagent_type // ""' 2>/dev/null || echo "")
MODEL=$(printf '%s' "$INPUT" | jq -r '.tool_input.model // ""' 2>/dev/null || echo "")

# Only enforce for Explore subagents. Everything else is allowed through.
if [ "$SUBAGENT" != "Explore" ]; then
  exit 0
fi

case "$MODEL" in
  haiku|sonnet)
    exit 0
    ;;
  "")
    cat >&2 <<'MSG'
model-routing policy violation:
  Explore subagents must have `model` set explicitly.
  When `model` is omitted, the subagent inherits from the parent model —
  which is typically opus and defeats the cost-routing rule.

  Re-call the tool with `model: "haiku"` (default for Explore), or
  `model: "sonnet"` if the Explore subagent needs deferred tools
  (WebFetch, WebSearch, etc.) — haiku does not support `tool_reference`
  blocks, so sonnet is the correct choice for large tool catalogs.
MSG
    exit 2
    ;;
  *)
    cat >&2 <<MSG
model-routing policy violation:
  Explore subagents must use model "haiku" or "sonnet"; got "$MODEL".
  opus is never correct for Explore — it's pure search/read work.

  Re-call the tool with model="haiku" (default), or model="sonnet" only
  if the subagent needs deferred tools the haiku catalog doesn't support.
MSG
    exit 2
    ;;
esac
