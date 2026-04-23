INSTRUCTION: MANDATORY SKILL ACTIVATION SEQUENCE

Step 1 - EVALUATE (do this in your response):
For each skill in <available_skills>, state: [skill-name] - YES/NO - [reason]
- cost-optimization: YES if task involves spawning subagents, NO otherwise
- programming-skills:golang-dev-guidelines: YES if task involves writing, reviewing, or refactoring Go code, NO otherwise

Step 2 - ACTIVATE (do this immediately after Step 1):
IF any skills are YES - Use Skill(skill-name) tool for EACH relevant skill NOW
IF no skills are YES - State "No skills needed" and proceed

Step 3 - MODEL SELECTION:
Consider whether the task warrants Opus. If yes, tell the user and suggest `/model opus` before proceeding.
Use Opus when the task involves:
- Architectural decisions spanning multiple systems
- Deep code reviews on large or complex PRs
- Complex multi-file refactoring across many components
Otherwise, Sonnet is sufficient.

Step 4 - IMPLEMENT:
Only after Steps 2-3 are complete, proceed with implementation.

CRITICAL: You MUST call Skill() tool in Step 2. Do NOT skip to implementation.
The evaluation (Step 1) is WORTHLESS unless you ACTIVATE (Step 2) the skills.

Example of correct sequence:
- research: NO - not a research task
- svelte5-runes: YES - need reactive state
- sveltekit-structure: YES - creating routes

[Then IMMEDIATELY use Skill() tool:]
> Skill(svelte5-runes)
> Skill(sveltekit-structure)

[THEN and ONLY THEN start implementation]
