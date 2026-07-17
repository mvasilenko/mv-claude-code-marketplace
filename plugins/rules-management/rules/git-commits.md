INSTRUCTION: BRANCH NAMING

- Version bump / maintenance / config: `chore/project-name-description`
- New feature: `feat/project-name-description`
- Bug fix: `fix/project-name-description`
- Breaking / major change: `breaking/project-name-description`

---

INSTRUCTION: BRANCH CREATION — ORDER OF OPERATIONS

Before creating a feature branch, follow this order (unless the user explicitly
asks for a different order or base):

1. Determine the repo's default branch — do not assume `main`. Resolve it, e.g.
   `git symbolic-ref refs/remotes/origin/HEAD --short` (falls back to
   `git remote show origin | sed -n 's/.*HEAD branch: //p'`). It may be `main`,
   `master`, or something else.
2. Fetch the latest default branch from the remote before branching, e.g.
   `git fetch origin <default-branch>`.
3. Create the feature branch from the freshly fetched default branch, e.g.
   `git checkout -b <type>/<project-name-description> origin/<default-branch>`.

This guarantees new work starts from up-to-date remote state, not a stale local
copy. Skip or reorder only when the user explicitly requests it.

---

INSTRUCTION: GIT COMMIT STYLE

Keep commit messages short and focused on WHY the change is needed, not what was done.
One-line subject is usually enough. No multi-line body unless truly necessary.

CONDITIONAL — before any `git commit`, run `git config user.email` and decide which
account type applies, then branch accordingly:

**Case A — private / OSS contribution account (DCO required):** apply these
OSS contribution requirements:

- Always commit with `-s` flag (DCO Signed-off-by using git user.name/user.email)
- Add `Assisted-by: claude-code/<current-model-id>` trailer to every commit.
  Resolve `<current-model-id>` from your session's environment block — the line
  "The exact model ID is `<id>`" (e.g. `claude-opus-4-7`, `claude-sonnet-4-6`,
  `claude-haiku-4-5-20251001`). Do not hardcode a specific model name; it will
  go stale the moment you or the user upgrades. Example:
  `git commit -s --trailer "Assisted-by: claude-code/claude-opus-4-7"`
- Disclose AI usage in PR descriptions with the `Assisted-by` tag, using the
  same dynamically-resolved model ID
- If the repo contains an `AGENTS.md` file, read and follow its conventions
- Keep commit messages, PR descriptions, and code comments terse — no AI verbosity

**Case B — company account:** do NOT apply DCO signoff (`-s`) or `Assisted-by`
trailers. Company repositories do not require DCO certification and should not
disclose AI assistance via commit/PR metadata. Use plain commits:
`git commit -m "<message>"`. Keep commit messages terse regardless of account.

---

INSTRUCTION: PR DESCRIPTION STYLE

Keep PR descriptions very brief — 1-3 bullet points maximum.
No verbose "Root cause" sections, no code blocks, no extended explanations.
If context is needed, one short sentence is enough.

---

INSTRUCTION: NO SENSITIVE OR LOCAL-SPECIFIC CONTENT IN COMMITTED FILES

Before committing any plan, doc, or config file, strip all local-specific or sensitive information:
- No local filesystem paths (e.g. /Users/username/Documents/...)
- No personal tokens, credentials, or secrets
- No machine-specific or environment-specific values that would not apply to other contributors

Use generic placeholders or relative paths instead.
