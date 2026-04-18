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
