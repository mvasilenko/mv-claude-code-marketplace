# mv-claude-code-marketplace

- On any plugin change, bump the patch version in that plugin's `.claude-plugin/plugin.json` (e.g. 1.1.0 -> 1.1.1). This is required for auto-update cache invalidation -- without a version bump, users won't receive the update.
- All changes must go through PRs from main. Never push directly to main.
