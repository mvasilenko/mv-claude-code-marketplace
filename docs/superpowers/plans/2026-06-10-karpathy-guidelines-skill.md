# karpathy-guidelines Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a new `karpathy-guidelines` plugin (copied from `tmp/skills/karpathy-guidelines`) to this marketplace, register it in the root marketplace manifest, and make it always-active via the `skill-eval` rule.

**Architecture:** Mirror the existing `plugins/programming-skills` layout: a `.claude-plugin/plugin.json` manifest plus a `skills/karpathy-guidelines/SKILL.md`. Register the plugin in `.claude-plugin/marketplace.json`. Add an unconditional "YES always" line to `plugins/rules-management/rules/skill-eval.md` (the canonical source synced to `~/.claude/rules/` and `~/.claude-personal/rules/` via `lib/sync-rules.sh` — do not edit the synced copies directly).

**Tech Stack:** Plain JSON (plugin manifests, marketplace manifest) and Markdown (skill definitions, rules).

---

### Task 1: Create the karpathy-guidelines plugin

**Files:**
- Create: `plugins/karpathy-guidelines/.claude-plugin/plugin.json`
- Create: `plugins/karpathy-guidelines/skills/karpathy-guidelines/SKILL.md`

- [ ] **Step 1: Create the plugin manifest**

Create `plugins/karpathy-guidelines/.claude-plugin/plugin.json`:

```json
{
  "name": "karpathy-guidelines",
  "description": "Behavioral guidelines to reduce common LLM coding mistakes: Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution",
  "version": "1.0.0",
  "author": {
    "name": "Company"
  }
}
```

- [ ] **Step 2: Validate the manifest is valid JSON**

Run: `jq . plugins/karpathy-guidelines/.claude-plugin/plugin.json`
Expected: pretty-printed JSON output, no error.

- [ ] **Step 3: Copy the SKILL.md content**

Create `plugins/karpathy-guidelines/skills/karpathy-guidelines/SKILL.md` with this exact content (copied verbatim from `tmp/skills/karpathy-guidelines/SKILL.md`):

```markdown
---
name: karpathy-guidelines
description: Behavioral guidelines to reduce common LLM coding mistakes. Use when writing, reviewing, or refactoring code to avoid overcomplication, make surgical changes, surface assumptions, and define verifiable success criteria.
license: MIT
---

# Karpathy Guidelines

Behavioral guidelines to reduce common LLM coding mistakes, derived from [Andrej Karpathy's observations](https://x.com/karpathy/status/2015883857489522876) on LLM coding pitfalls.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.
```

- [ ] **Step 4: Verify the copy is byte-identical to the source**

Run: `diff plugins/karpathy-guidelines/skills/karpathy-guidelines/SKILL.md tmp/skills/karpathy-guidelines/SKILL.md`
Expected: no output (files identical).

- [ ] **Step 5: Commit**

```bash
git add plugins/karpathy-guidelines
git commit -s --trailer "Assisted-by: claude-code/claude-sonnet-4-6" -m "Add karpathy-guidelines plugin"
```

---

### Task 2: Register the plugin in the marketplace manifest

**Files:**
- Modify: `.claude-plugin/marketplace.json`

- [ ] **Step 1: Add a new plugin entry**

In `.claude-plugin/marketplace.json`, the `plugins` array currently ends with the `usage-tracking` entry:

```json
    {
      "name": "usage-tracking",
      "description": "Logs skill invocations (PreToolUse and PostToolUse) with user, pwd, and skill name",
      "source": "./plugins/usage-tracking",
      "category": "utility"
    }
  ]
}
```

Change it to add a new entry after `usage-tracking`:

```json
    {
      "name": "usage-tracking",
      "description": "Logs skill invocations (PreToolUse and PostToolUse) with user, pwd, and skill name",
      "source": "./plugins/usage-tracking",
      "category": "utility"
    },
    {
      "name": "karpathy-guidelines",
      "description": "Behavioral guidelines to reduce common LLM coding mistakes: Think Before Coding, Simplicity First, Surgical Changes, Goal-Driven Execution",
      "source": "./plugins/karpathy-guidelines",
      "category": "workflow"
    }
  ]
}
```

- [ ] **Step 2: Validate the manifest is valid JSON**

Run: `jq . .claude-plugin/marketplace.json`
Expected: pretty-printed JSON output, no error, and the new entry visible at the end of `.plugins`.

- [ ] **Step 3: Commit**

```bash
git add .claude-plugin/marketplace.json
git commit -s --trailer "Assisted-by: claude-code/claude-sonnet-4-6" -m "Register karpathy-guidelines plugin in marketplace"
```

---

### Task 3: Make karpathy-guidelines always-active via skill-eval

**Files:**
- Modify: `plugins/rules-management/rules/skill-eval.md`
- Modify: `plugins/rules-management/.claude-plugin/plugin.json`

- [ ] **Step 1: Add the always-on evaluation line**

In `plugins/rules-management/rules/skill-eval.md`, the Step 1 list currently reads:

```
Step 1 - EVALUATE (do this in your response):
For each skill in <available_skills>, state: [skill-name] - YES/NO - [reason]
- cost-optimization: YES if task involves spawning subagents, NO otherwise
- programming-skills:golang-dev-guidelines: YES if task involves writing, reviewing, or refactoring Go code, NO otherwise
```

Change it to:

```
Step 1 - EVALUATE (do this in your response):
For each skill in <available_skills>, state: [skill-name] - YES/NO - [reason]
- cost-optimization: YES if task involves spawning subagents, NO otherwise
- programming-skills:golang-dev-guidelines: YES if task involves writing, reviewing, or refactoring Go code, NO otherwise
- karpathy-guidelines: YES always - apply these coding guidelines to every task
```

- [ ] **Step 2: Update the worked example to include the always-on skill**

The example at the bottom of the same file currently reads:

```
Example of correct sequence:
- research: NO - not a research task
- svelte5-runes: YES - need reactive state
- sveltekit-structure: YES - creating routes

[Then IMMEDIATELY use Skill() tool:]
> Skill(svelte5-runes)
> Skill(sveltekit-structure)

[THEN and ONLY THEN start implementation]
```

Change it to:

```
Example of correct sequence:
- research: NO - not a research task
- svelte5-runes: YES - need reactive state
- sveltekit-structure: YES - creating routes
- karpathy-guidelines: YES always - apply these coding guidelines

[Then IMMEDIATELY use Skill() tool:]
> Skill(svelte5-runes)
> Skill(sveltekit-structure)
> Skill(karpathy-guidelines)

[THEN and ONLY THEN start implementation]
```

- [ ] **Step 3: Bump the rules-management plugin version**

In `plugins/rules-management/.claude-plugin/plugin.json`, change:

```json
  "version": "1.0.9",
```

to:

```json
  "version": "1.0.10",
```

(Per this repo's `CLAUDE.md`: any plugin change requires a patch version bump for auto-update cache invalidation.)

- [ ] **Step 4: Validate the manifest is valid JSON**

Run: `jq . plugins/rules-management/.claude-plugin/plugin.json`
Expected: pretty-printed JSON, `"version": "1.0.10"`.

- [ ] **Step 5: Commit**

```bash
git add plugins/rules-management/rules/skill-eval.md plugins/rules-management/.claude-plugin/plugin.json
git commit -s --trailer "Assisted-by: claude-code/claude-sonnet-4-6" -m "Activate karpathy-guidelines on every task via skill-eval"
```

---

### Task 4: Final verification

**Files:**
- None (verification only)

- [ ] **Step 1: Review the full diff against main**

Run: `git diff main --stat`
Expected: shows the 4 changed/new files from Tasks 1-3 (plus the spec doc from the design phase).

- [ ] **Step 2: Final status check**

Run: `git status --porcelain`
Expected: no uncommitted changes. `tmp/` is gitignored and no longer shows as untracked.
