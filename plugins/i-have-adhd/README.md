# i-have-adhd

Vendored from [ayghri/i-have-adhd](https://github.com/ayghri/i-have-adhd) (MIT, see `LICENSE`)
at commit `cbe69fb`, upstream version `0.2.0`.

## Local modifications

- Dropped `disable-model-invocation: true` from the skill frontmatter. Upstream gates the
  skill behind an explicit `/i-have-adhd` invocation or an opt-in `~/.claude/.i-have-adhd-always`
  flag file; here the `rules-management` skill-eval rule activates it unconditionally, which
  requires model invocation to be allowed.
- Upstream's SessionStart hooks, multi-harness manifests (Gemini/Codex/opencode/Kimi), evals,
  and tests are not vendored — the skill is the only part this marketplace uses.

The `version` field tracks this copy, not upstream. Bump it on any local change so auto-update
invalidates the cache.
