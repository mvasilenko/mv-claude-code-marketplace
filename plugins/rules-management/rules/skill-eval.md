INSTRUCTION: MANDATORY SKILL ACTIVATION SEQUENCE

Step 0 - CAVEMAN ALWAYS (no evaluation, no exceptions):
caveman:caveman is ALWAYS active, for EVERY task, regardless of which other
skills apply or whether any apply at all. It is not part of the YES/NO
evaluation - it is unconditional. Never skip it, never trade it off against
another skill, never drop it because the task is short, conversational, or
non-coding. Other skills stack ON TOP of caveman; they never replace it.
Only the user saying "stop caveman" / "normal mode" turns it off.

Step 1 - EVALUATE (do this in your response):
For each skill in <available_skills>, state: [skill-name] - YES/NO - [reason]
- cost-optimization: YES if task involves spawning subagents, NO otherwise
- programming-skills:golang-dev-guidelines: YES if task involves writing, reviewing, or refactoring Go code, NO otherwise
- karpathy-guidelines: YES always - apply these coding guidelines to every task
(caveman:caveman is NOT evaluated here - see Step 0, it is always on)

NEVER call Skill() for a skill whose SKILL.md sets `disable-model-invocation: true`
(currently: i-have-adhd:i-have-adhd). Those are user-invoked only - the tool call
always fails. If one is wanted, tell the user to run `/<skill-name>` themselves.
Do not paste the skill's content inline as a substitute. i-have-adhd is already
always-on via its own SessionStart hook (`~/.claude/.i-have-adhd-always`, created
by setup.sh), so its ruleset arrives as session context - obey it, never Skill() it.

Step 2 - ACTIVATE (do this immediately after Step 1):
IF any skills are YES - Use Skill(skill-name) tool for EACH relevant skill NOW
IF no skills are YES - State "No skills needed" and proceed

Step 2b - LEAN REVIEW (mandatory after any code is written or changed):
Run /ponytail:ponytail-review on the diff and APPLY its findings - delete the
reinvented stdlib, drop the speculative abstractions, cut the unneeded deps.
Report what was cut in one line. Skip only if no code changed this turn.

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
- karpathy-guidelines: YES always - apply these coding guidelines
- caveman:caveman: YES always - terse output style

[Then IMMEDIATELY use Skill() tool:]
> Skill(svelte5-runes)
> Skill(sveltekit-structure)
> Skill(karpathy-guidelines)
> Skill(caveman:caveman)

[THEN implement, THEN run /ponytail:ponytail-review and apply the cuts]
