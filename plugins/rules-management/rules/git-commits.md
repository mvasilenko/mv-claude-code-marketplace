INSTRUCTION: BRANCH NAMING

- Version bump / maintenance / config: `chore/project-name-description`
- New feature: `feat/project-name-description`
- Bug fix: `fix/project-name-description`
- Breaking / major change: `breaking/project-name-description`

---

INSTRUCTION: GIT COMMIT STYLE

Keep commit messages short and focused on WHY the change is needed, not what was done.
One-line subject is usually enough. No multi-line body unless truly necessary.

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
