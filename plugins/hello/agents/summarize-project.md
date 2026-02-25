---
name: summarize-project
description: Summarizes the current project structure and purpose
tools: Read, Glob, Grep
model: haiku
permissionMode: default
---

# Summarize Project

Analyze the current working directory and provide a brief summary:

1. Use Glob to find key files (README.md, package.json, go.mod, etc.)
2. Read the main documentation file
3. List the top-level directory structure
4. Output a 3-5 sentence summary of what this project does
