# Enforce Explore → haiku model routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the `model-routing` plugin's advisory "Explore → haiku" rule into an *enforced* rule by adding a `PreToolUse` hook that blocks subagent spawns which violate the policy, and bring the rule-file text in sync with the fuller cost-optimization skill so the guidance is always in session context rather than only after the skill is invoked.

**Architecture:** Add a `PreToolUse` hook to the existing `plugins/model-routing/hooks/` directory that parses the JSON stdin payload Claude Code sends on every subagent-spawning tool call, and exits non-zero with a message when `subagent_type == "Explore"` is called without `model` set to `haiku` or `sonnet`. The hook is wired via `hooks.json` alongside the existing `SessionStart` entry. The rule file `rules/explore-haiku.md` is expanded so it (a) covers the parent-model-inheritance trap, (b) covers `general-purpose` routing for search work, and (c) documents the `tool_reference` / large-catalog exception that legitimately requires `sonnet`. The existing `skills/cost-optimization/SKILL.md` stays as the on-demand deep-dive; the rule file carries the always-loaded short form.

**Tech Stack:** Bash (hook scripts), `jq` (JSON parsing from stdin — already a repo prerequisite per `README.md`), Claude Code `PreToolUse` hook contract, Claude Code plugin manifest (`hooks.json`, `.claude-plugin/plugin.json`).

---

## File Structure

Changes are contained entirely within `plugins/model-routing/` plus docs/plan files. No shared library changes, no install-script changes.

**Create:**
- `plugins/model-routing/hooks/pre-tool-use.sh` — the enforcement hook (executable bash)
- `plugins/model-routing/hooks/pre-tool-use.test.sh` — lightweight bash test harness (executable bash)
- `plugins/model-routing/hooks/fixtures/explore-no-model.json` — golden input: Explore call with no `model`
- `plugins/model-routing/hooks/fixtures/explore-opus.json` — golden input: Explore call with `model: "opus"`
- `plugins/model-routing/hooks/fixtures/explore-haiku.json` — golden input: Explore call with `model: "haiku"` (should pass)
- `plugins/model-routing/hooks/fixtures/explore-sonnet.json` — golden input: Explore call with `model: "sonnet"` (should pass — legit `tool_reference` case)
- `plugins/model-routing/hooks/fixtures/general-purpose-no-model.json` — golden input: non-Explore call (should pass)
- `plugins/model-routing/hooks/fixtures/probe-any.json` — diagnostic payload used only in Task 1

**Modify:**
- `plugins/model-routing/hooks/hooks.json` — add a `PreToolUse` entry alongside the existing `SessionStart` entry
- `plugins/model-routing/rules/explore-haiku.md` — expand content (keep filename unchanged so the installed-file hash tracker in `lib/sync-rules.sh` still works)
- `plugins/model-routing/.claude-plugin/plugin.json` — bump version `1.0.6` → `1.0.7` (per `CLAUDE.md` / `README.md` convention)

**Do not modify:**
- `lib/sync-rules.sh` — filename is unchanged, so the hash-tracked update path keeps working
- `setup.sh`, `setup-lib.sh`, `setup-litellm.sh` — install entry points unchanged
- `plugins/model-routing/skills/cost-optimization/SKILL.md` — left as the on-demand deep-dive (duplication is fine; rule file is the short form that's always loaded)

---

## Key design decisions locked here

1. **Matcher string uses a defensive regex `"Task|Agent"`.** Claude Code hook matchers are regex-evaluated against the tool name. Historically the subagent-spawning tool is emitted as `Task`; the top-level tool identifier in the current SDK schema is `Agent`. Using the alternation covers both without an empirical probe and gives us a safe default. Task 6's live verification is the fallback check if neither name fires.

2. **Block policy is: Explore with `model` missing OR `model == "opus"` → deny.** Both `haiku` and `sonnet` are allowed because the `cost-optimization` skill legitimately calls for `sonnet` when the Explore subagent needs a large/deferred tool catalog (haiku doesn't support `tool_reference` blocks). Forcing `haiku`-only would break those cases.

3. **Scope is `Explore` only, not `general-purpose`.** `general-purpose` has too many legitimate uses (including non-search work) to block on `model` absence. The rule *text* steers `general-purpose` toward correct model choice; the *hook* only enforces the narrowly-scoped rule so false positives stay low.

4. **Block mechanism: exit code 2 + stderr message.** Claude Code shows PreToolUse-hook stderr to the model when the hook exits non-zero, which is exactly the surface we need to nudge a corrected retry. Structured JSON `hookSpecificOutput` is more powerful but version-sensitive; we use the simple contract.

5. **Rule file keeps its name.** Renaming would leave an orphan `model-routing-explore-haiku.md` in every user's `~/.claude/rules/` (the sync script's hash gate prevents overwriting, and has no deletion logic for renamed files). The content broadens; the filename stays.

6. **No shared-library test framework is introduced.** This repo is bash-only with no existing test harness. A small in-repo `pre-tool-use.test.sh` that pipes fixture JSON into the hook and asserts exit codes is sufficient — introducing `bats` would be scope creep.

---

### Task 1: Write the enforcement hook with a TDD-style fixture-driven harness

**Why TDD here:** This hook is pure data-in / exit-code-out — the cleanest possible unit to test with a handful of golden JSON payloads piped through it. Writing the harness and fixtures first makes the acceptance criteria explicit before any logic is written.

**Files:**
- Create: `plugins/model-routing/hooks/fixtures/explore-no-model.json`
- Create: `plugins/model-routing/hooks/fixtures/explore-opus.json`
- Create: `plugins/model-routing/hooks/fixtures/explore-haiku.json`
- Create: `plugins/model-routing/hooks/fixtures/explore-sonnet.json`
- Create: `plugins/model-routing/hooks/fixtures/general-purpose-no-model.json`
- Create: `plugins/model-routing/hooks/pre-tool-use.test.sh`
- Create: `plugins/model-routing/hooks/pre-tool-use.sh`

- [ ] **Step 1: Write the fixtures (failing inputs first)**

Create `plugins/model-routing/hooks/fixtures/explore-no-model.json`:

```json
{
  "tool_name": "Task",
  "tool_input": {
    "subagent_type": "Explore",
    "description": "List files in plugins dir",
    "prompt": "List files in plugins/."
  }
}
```

Create `plugins/model-routing/hooks/fixtures/explore-opus.json`:

```json
{
  "tool_name": "Task",
  "tool_input": {
    "subagent_type": "Explore",
    "model": "opus",
    "description": "List files",
    "prompt": "List files."
  }
}
```

Create `plugins/model-routing/hooks/fixtures/explore-haiku.json`:

```json
{
  "tool_name": "Task",
  "tool_input": {
    "subagent_type": "Explore",
    "model": "haiku",
    "description": "List files",
    "prompt": "List files."
  }
}
```

Create `plugins/model-routing/hooks/fixtures/explore-sonnet.json`:

```json
{
  "tool_name": "Task",
  "tool_input": {
    "subagent_type": "Explore",
    "model": "sonnet",
    "description": "Explore with deferred tools",
    "prompt": "Use WebFetch to read docs and summarize."
  }
}
```

Create `plugins/model-routing/hooks/fixtures/general-purpose-no-model.json`:

```json
{
  "tool_name": "Task",
  "tool_input": {
    "subagent_type": "general-purpose",
    "description": "General task",
    "prompt": "Do stuff."
  }
}
```

The hook matches on `tool_input.subagent_type` / `tool_input.model`, not on `tool_name`, so the exact `tool_name` value in the fixtures doesn't affect the hook logic — it's shown here for realism.

- [ ] **Step 2: Write the test harness**

Create `plugins/model-routing/hooks/pre-tool-use.test.sh`:

```bash
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

exit "$FAIL"
```

Make it executable:

```bash
chmod +x plugins/model-routing/hooks/pre-tool-use.test.sh
```

- [ ] **Step 3: Run the test harness — expect all five FAIL**

Run:

```bash
./plugins/model-routing/hooks/pre-tool-use.test.sh
```

Expected: harness errors because `pre-tool-use.sh` doesn't exist yet. This is the failing-test state.

- [ ] **Step 4: Write the hook implementation**

Create `plugins/model-routing/hooks/pre-tool-use.sh`:

```bash
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
```

Make it executable:

```bash
chmod +x plugins/model-routing/hooks/pre-tool-use.sh
```

- [ ] **Step 5: Run the test harness — expect all PASS**

Run:

```bash
./plugins/model-routing/hooks/pre-tool-use.test.sh
```

Expected output contains five `PASS` lines and no `FAIL` lines, exit code 0:

```
PASS explore-no-model.json (exit=2)
PASS explore-opus.json (exit=2)
PASS explore-haiku.json (exit=0)
PASS explore-sonnet.json (exit=0)
PASS general-purpose-no-model.json (exit=0)
```

- [ ] **Step 6: Commit the hook and its tests**

```bash
git add plugins/model-routing/hooks/pre-tool-use.sh plugins/model-routing/hooks/pre-tool-use.test.sh plugins/model-routing/hooks/fixtures/
git commit -m "feat(model-routing): add PreToolUse hook enforcing Explore→haiku/sonnet"
```

---

### Task 2: Wire the real hook into hooks.json

**Files:**
- Modify: `plugins/model-routing/hooks/hooks.json`

- [ ] **Step 1: Replace `hooks.json` with the production wiring**

Overwrite `plugins/model-routing/hooks/hooks.json` with:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "\"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/plugins/marketplaces/mv-claude-code-marketplace/plugins/model-routing/hooks/session-start.sh\""
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Task|Agent",
        "hooks": [
          {
            "type": "command",
            "command": "\"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/plugins/marketplaces/mv-claude-code-marketplace/plugins/model-routing/hooks/pre-tool-use.sh\""
          }
        ]
      }
    ]
  }
}
```

The matcher uses the regex alternation `Task|Agent` to cover both the historical tool name (`Task`) and the current SDK-level name (`Agent`) without needing an empirical probe.

- [ ] **Step 2: Commit the wiring change**

```bash
git add plugins/model-routing/hooks/hooks.json
git commit -m "feat(model-routing): wire PreToolUse enforcement for Explore subagents"
```

---

### Task 3: Expand the rule file to cover inheritance and the sonnet exception

**Why:** The current 74-byte `explore-haiku.md` says "Always use haiku for Explore subagents." That's insufficient in two ways: (a) it doesn't warn that omitting `model` inherits from parent (main bug source), and (b) it doesn't surface the `tool_reference` / deferred-tool-catalog exception that the cost-optimization skill correctly carves out for sonnet. Expanding the rule means this guidance is in context on every session, not just after `Skill(cost-optimization)` is invoked.

**Files:**
- Modify: `plugins/model-routing/rules/explore-haiku.md`

- [ ] **Step 1: Replace the file contents**

Overwrite `plugins/model-routing/rules/explore-haiku.md` with:

```markdown
# Agent Model Routing

## Rule: Explore subagents must have `model` set explicitly

When spawning a subagent with `subagent_type: "Explore"`:

- Default to `model: "haiku"` — Explore is search/read work and cheap by design.
- Use `model: "sonnet"` only when the Explore subagent needs *deferred tools*
  (e.g. `WebFetch`, `WebSearch`) or a large tool catalog: haiku does not
  support `tool_reference` blocks and auto-disables dynamic tool discovery.
- Never use `model: "opus"` for Explore.
- **Never omit `model`.** When the parameter is missing, the subagent inherits
  from the parent model — which in typical sessions is `opus` or `sonnet` —
  and silently defeats the whole cost-routing rule. This is the #1 cause of
  "my Explore subagent ran on opus" surprises.

A `PreToolUse` hook (shipped by this same plugin) rejects Explore calls that
violate this rule and asks the model to retry with a correct `model` value.

## General-purpose subagents used for search

The `general-purpose` subagent is *not* covered by the enforcement hook,
because it has many legitimate non-search uses. But when you're using it
for search-like work, apply the same choice manually:

- `haiku` for straightforward search / read / grep-style work.
- `sonnet` when deferred tools or a large tool catalog are needed.

If the work is purely search, prefer `subagent_type: "Explore"` over
`general-purpose` so the rule (and its hook) applies.

## Other subagent types

- `Plan` — use `sonnet` (multi-step reasoning, not simple search).
- MCP handler agents (`notion-handler`, `linear-handler`) have `model: haiku`
  baked into their definition; you do not need to override.
- `opus` is only correct for architectural decisions across multiple systems,
  deep multi-file debugging, or work clearly beyond sonnet's capability.
```

- [ ] **Step 2: Commit the rule update**

```bash
git add plugins/model-routing/rules/explore-haiku.md
git commit -m "docs(model-routing): expand rule to cover inheritance and sonnet exception"
```

---

### Task 4: Bump the plugin version for auto-update cache invalidation

**Why:** The repo's `CLAUDE.md` mandates a patch bump on any plugin change so Claude Code's auto-update path invalidates the cached plugin. Without the bump, existing users won't pick up the hook or the updated rule.

**Files:**
- Modify: `plugins/model-routing/.claude-plugin/plugin.json`

- [ ] **Step 1: Bump `1.0.6` → `1.0.7`**

Replace `plugins/model-routing/.claude-plugin/plugin.json` with:

```json
{
  "name": "model-routing",
  "description": "Enforces cost-effective model routing for subagent tasks",
  "version": "1.0.7",
  "author": {
    "name": "Company"
  }
}
```

- [ ] **Step 2: Commit the bump**

```bash
git add plugins/model-routing/.claude-plugin/plugin.json
git commit -m "chore(model-routing): bump to 1.0.7 for hook + rule update"
```

---

### Task 5: End-to-end verification in a real Claude Code session

**Why:** The harness in Task 2 verifies the hook contract in isolation. This task verifies that Claude Code actually wires the hook, that the matcher fires on a real subagent call, and that both a violating call is denied and a compliant call succeeds.

**Files:** none (manual verification with live CC session).

- [ ] **Step 1: Reinstall the plugin into the local config**

```bash
./setup.sh
```

This re-syncs the rules via the `SessionStart` hook on next launch and ensures the new plugin version is registered.

- [ ] **Step 2: Violation case — confirm the hook blocks**

In a fresh Claude Code session, ask:

> Spawn an Explore subagent (no model override) to list files in `plugins/`.

Expected: Claude Code shows the hook stderr ("model-routing policy violation: Explore subagents must have `model` set explicitly…") as tool feedback, and Claude retries the call with `model: "haiku"`.

- [ ] **Step 3: Allow case (haiku) — confirm pass-through**

Ask:

> Spawn an Explore subagent with `model: "haiku"` to list files in `plugins/`.

Expected: the subagent spawns normally; no hook feedback surfaces.

- [ ] **Step 4: Allow case (sonnet) — confirm the exception works**

Ask:

> Spawn an Explore subagent with `model: "sonnet"` to fetch https://example.com and summarize.

Expected: the subagent spawns normally; no hook feedback surfaces.

- [ ] **Step 5: Negative guardrail — non-Explore subagent unaffected**

Ask:

> Spawn a `general-purpose` subagent (no model override) to describe this repo.

Expected: the subagent spawns normally; no hook feedback surfaces. (The hook intentionally does not constrain non-Explore types.)

- [ ] **Step 6: If any check fails, narrow the matcher**

If the violation case does not block, the most likely cause is a matcher mismatch. Inspect `~/.claude/skill-usage.log` and `~/.claude/hooks.log` (if present). Adjust the `"matcher"` value in `plugins/model-routing/hooks/hooks.json` — try `"Task"` alone, then `"Agent"` alone — and commit a follow-up.

---

### Task 6: Open the PR

**Files:** none (git/gh operations).

- [ ] **Step 1: Push the branch**

```bash
git push -u origin feat/model-routing-enforce-explore-haiku
```

- [ ] **Step 2: Open the PR against main**

```bash
gh pr create --base main --title "enforce Explore→haiku/sonnet via PreToolUse hook" --body "$(cat <<'EOF'
- Add PreToolUse hook that denies Explore subagent spawns with missing/opus `model` (parent-inheritance trap was the root cause of silent opus usage for Explore).
- Expand `rules/explore-haiku.md` to cover the inheritance trap and the legitimate `sonnet` exception for deferred-tool catalogs.
- Bump `model-routing` to 1.0.7.
EOF
)"
```

- [ ] **Step 3: Share the PR URL**

Copy the URL returned by `gh pr create` into the chat for review.

---

## Self-Review

**1. Spec coverage** (against the three changes scoped in the prior conversation turn):

- "PreToolUse hook that blocks Explore without haiku/sonnet" → Tasks 1 (TDD implementation) and 2 (wiring).
- "Strengthen rule text to cover inheritance and general-purpose" → Task 3.
- "Fold the tool_reference / sonnet-exception nuance from the skill into the rule" → Task 3 (included in the rewritten rule).
- Repo convention: plugin version bump → Task 4.
- Live-session verification → Task 5.
- Ship via PR (CLAUDE.md rule "never push directly to main") → Task 6.

No gaps.

**2. Placeholder scan:** No TBDs, no "add appropriate error handling," no "similar to Task N" without code. All fixtures, the hook script, the test harness, and the rule file are shown in full.

**3. Type / name consistency:**

- Fixture `tool_name` values match the matcher string the plan wires (`"Task"`, adjusted per Task 1 findings).
- Hook reads `.tool_input.subagent_type` and `.tool_input.model`; fixtures all set those fields at the same path.
- Rule file and hook message agree on the allowed set `{haiku, sonnet}` and the denied set `{missing, opus}`.
- Version bump (`1.0.6` → `1.0.7`) matches the repo's documented convention.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-18-enforce-explore-haiku.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
