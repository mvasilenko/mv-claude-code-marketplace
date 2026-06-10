# karpathy-guidelines skill: design

## Overview

Add a new marketplace plugin, `karpathy-guidelines`, that packages a set of
general coding guidelines (Think Before Coding, Simplicity First, Surgical
Changes, Goal-Driven Execution) as a Claude Code skill, sourced from the
upstream `andrej-karpathy-skills` plugin. Wire it into the existing
`skill-eval` rule so it is always activated, alongside the conditional
entries already there (e.g. `golang-dev-guidelines`).

## Components

- `plugins/karpathy-guidelines/.claude-plugin/plugin.json`
  - `name: karpathy-guidelines`, `version: 1.0.0`, `author: Company`,
    description summarizing the guidelines.
- `plugins/karpathy-guidelines/skills/karpathy-guidelines/SKILL.md`
  - Copied verbatim from the upstream plugin's
    `skills/karpathy-guidelines/SKILL.md` (frontmatter + four guideline
    sections: Think Before Coding, Simplicity First, Surgical Changes,
    Goal-Driven Execution).
- Root `.claude-plugin/marketplace.json`
  - New entry: `name: karpathy-guidelines`,
    `source: ./plugins/karpathy-guidelines`, `category: workflow`,
    description.
- `plugins/rules-management/rules/skill-eval.md`
  - New always-on line in the Step 1 evaluation list:
    `- karpathy-guidelines: YES always - apply these coding guidelines to every task`
  - Add `Skill(karpathy-guidelines)` to the example activation sequence.
- `plugins/rules-management/.claude-plugin/plugin.json`
  - Version bump `1.0.9` -> `1.0.10` (rule file changed; required for
    auto-update cache invalidation per repo CLAUDE.md).

## Data flow / activation flow

1. Session start: `rules-management`'s session-start hook syncs
   `plugins/rules-management/rules/*.md` to `~/.claude/rules/` (and the
   user's personal rules dir), including the updated `skill-eval.md`.
2. On any task, the assistant follows the `skill-eval.md` mandatory
   sequence: it now always evaluates `karpathy-guidelines` as YES and
   invokes `Skill(karpathy-guidelines)`, loading the guideline content
   for that task.
3. `karpathy-guidelines` is a standalone skill (own plugin), independent
   of `programming-skills` — no shared code or dependencies.

## Error handling

None needed — this is static content distribution (markdown + JSON
config). No runtime logic is added.

## Testing / verification

- `marketplace.json` and both `plugin.json` files are valid JSON
  (`jq . <file>` or similar).
- `skill-eval.md` diff shows the new line added without altering existing
  conditional entries.
- Manual check: starting a new session and confirming the skill appears
  in the available-skills list and the eval sequence references it.
