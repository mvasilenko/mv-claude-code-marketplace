---
name: cost-optimization
description: Enforces cost-effective model selection when spawning Task subagents. Always active.
---

# Model Routing Policy

When spawning Task subagents, set the `model` parameter based on task complexity. Sonnet is the default:

## haiku (for exploration and search)
Use for:
- Explore-type agents (codebase exploration, file search, keyword search)
- Simple lookups, grep, glob operations
- Reading and summarizing files
- Answering straightforward questions about code
- Any task that is primarily search/read with minimal reasoning

> **Limitation:** haiku does not support `tool_reference` blocks (Sonnet 4+ / Opus 4+ only). Tool search — the API feature that lets Claude dynamically discover tools from large catalogs, reducing context by 85%+ — is automatically disabled for haiku subagents. If an agent needs to work with a large or deferred tool catalog, use sonnet instead.

## sonnet (default for most tasks)
Use for:
- Writing or modifying code
- General-purpose agents doing moderate complexity work
- Refactoring, test writing, documentation generation
- Plan-type agents for implementation planning
- Tasks requiring multi-step reasoning but not architectural decisions

## opus (only for complex tasks)
Use only when:
- The user explicitly requests it
- The task involves complex architectural decisions across multiple systems
- Debugging requires deep reasoning across many files
- The task is clearly beyond sonnet's capability

## MCP tool routing (Notion, Linear)
When the user asks to interact with Notion or Linear, use the dedicated agents:
- **Notion tasks**: spawn Task with `subagent_type: "notion-handler"` (runs on haiku, has all Notion MCP tools)
- **Linear tasks**: spawn Task with `subagent_type: "linear-handler"` (runs on haiku, has all Linear MCP tools)
- These agents already have `model: haiku` configured, no need to override
- Only skip delegation and use the main session model if the task requires complex reasoning about the results combined with code changes
- Simple lookups, status updates, issue creation, page fetches must always go through these agents

## Bash validation delegation
When a task involves multiple validation commands (JSON checks, syntax checks, version checks, hook extraction, etc.) that are read-only and produce small output, bundle them into a single haiku subagent instead of running them in the main session. This keeps the main context clean and uses the cheaper model for work that requires no reasoning beyond pass/fail interpretation.

## Rules
- Default to sonnet for most tasks
- Use haiku for exploration, search, read-only lookups, and simple file operations (as defined above)
- Use opus only for the most complex and demanding tasks or individual steps — architectural decisions spanning multiple systems, deep multi-file debugging, tasks clearly beyond sonnet's capability
- When uncertain between sonnet and opus, use sonnet
- Never use opus for search, exploration, or straightforward code changes
- Delegate read-only validation work to haiku subagents rather than running it in the main session
