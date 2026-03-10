---
name: vigilante-issue-implementation
description: Implement a GitHub issue end-to-end when Vigilante dispatches work for a watched repository. Use the provided worktree, respect repository instructions, comment on the issue as work progresses, and report failures back to GitHub.
---

# Vigilante Issue Implementation

## Overview
Implement one GitHub issue from Vigilante dispatch through validated code changes in the provided worktree. Always work inside the assigned worktree, always respect repository instructions, and always keep the GitHub issue updated with start, progress, and failure comments.

## Inputs
Require these inputs from Vigilante:

- issue number
- issue title and URL
- repository slug
- local repository path
- assigned worktree path
- branch name

## Workflow
1. Inspect issue and repository constraints
- Read the issue details supplied by Vigilante and confirm the issue scope before coding.
- Read development constraints from repository markdown files before making changes:
  - `AGENTS.md` when present
  - `README.md`
  - other root or area-specific docs that affect touched files
- If repository instructions conflict, follow the more specific instruction.

2. Announce session start on GitHub
- Post a comment on the issue as soon as work begins.
- Include that Vigilante launched the session, the working branch, and that implementation is in progress.

3. Implement inside the assigned worktree only
- Use only the provided worktree path.
- Never edit the root checkout when a worktree was assigned.
- Keep changes scoped to the issue.
- Prefer native repository tooling and avoid unnecessary new dependencies.
- Preserve existing coding patterns unless the issue requires a different approach.

4. Validate incrementally
- Run relevant tests, builds, or linters for the changed area before concluding work.
- Prefer targeted validation first, then broader validation when necessary.
- If a command fails, determine whether the problem is in the code, test setup, or environment before retrying.

5. Post progress comments at meaningful milestones
- Comment when investigation is complete and implementation starts.
- Comment when major milestones are reached, such as a core fix landing or tests passing.
- Keep comments concise and factual.
- Do not spam the issue with low-signal updates.

6. Handle failures and blockers explicitly
- If tool setup fails, validation fails, or the issue is blocked, comment on the issue with the concrete problem.
- Include enough detail for a human maintainer to understand the current state and next step.
- If work cannot proceed safely, stop and report the blocker instead of guessing.

7. Finish with a clear terminal state
- Leave the worktree in a coherent state.
- Ensure any executed validations are accurately reported.
- If the task completed successfully, summarize what changed and what was validated.
- If the task failed, summarize the failure clearly in the issue comment.

## GitHub Commenting Rules
- Always comment when the session starts.
- Add at least one progress comment for non-trivial implementations.
- Comment immediately on any execution failure or blocking condition.
- Comments should be concise, concrete, and tied to real progress.
- Avoid generic status text that does not help the issue reader.

## Guardrails
- Never work outside the assigned worktree.
- Never ignore `AGENTS.md` or repository documentation that constrains implementation.
- Never make unrelated refactors unless they are required to complete the issue safely.
- Never silently fail; report errors or blockers back to the issue.
- Never claim validation passed unless the corresponding command actually succeeded.

## Completion Criteria
- The issue received a start comment.
- Progress or failure comments were posted as appropriate for the work performed.
- Code changes are scoped to the issue and live in the assigned worktree.
- Relevant validation was run and accurately reported.
- Final session state is clear to both Vigilante and the GitHub issue reader.

## Output Expectations
When using this skill, the agent should leave:

- code changes in the assigned worktree
- a clear issue comment trail
- accurate success or failure reporting
