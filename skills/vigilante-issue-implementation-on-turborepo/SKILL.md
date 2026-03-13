---
name: vigilante-issue-implementation-on-turborepo
description: Implement a GitHub issue end-to-end when Vigilante dispatches work for a watched Turborepo monorepo. Use the provided worktree, respect repository instructions, comment on the issue as work progresses, and report failures back to GitHub.
---

# Vigilante Turborepo Issue Implementation

## Overview
Implement one GitHub issue from Vigilante dispatch through validated code changes, a pushed branch, and an opened pull request from the provided worktree. Always work inside the assigned worktree, respect repository instructions, and keep the GitHub issue updated with start, plan, progress, PR, and failure comments.

## Turborepo Focus
- Read the repo/process context supplied in the prompt before changing code.
- Limit edits to the apps, packages, and shared modules required for the issue.
- Prefer targeted `turbo`, package, or workspace validation for the touched area before broader monorepo checks.
- If the repo needs local databases or dependent services, use the shared `docker-compose-launch` contract supplied in the prompt instead of creating ad hoc compose logic.

## Workflow
1. Inspect issue and repository constraints
- Read the issue details supplied by Vigilante and confirm the issue scope before coding.
- Read development constraints from repository markdown files before making changes:
  - `AGENTS.md` when present
  - `README.md`
  - other root or area-specific docs that affect touched files
- If repository instructions conflict, follow the more specific instruction.

2. Announce session start on GitHub
- Post a comment on the issue as soon as work begins using `gh issue comment`.
- Include that Vigilante launched the session, the working branch, and that implementation is in progress.

3. Post an implementation plan early
- After inspecting the issue and repository constraints, post a concise implementation plan to the issue using `gh issue comment`.
- The plan comment should describe the intended development steps before substantial coding work begins.

4. Implement inside the assigned worktree only
- Use only the provided worktree path.
- Never edit the root checkout when a worktree was assigned.
- Keep changes scoped to the issue.
- Prefer native repository tooling and avoid unnecessary new dependencies.

5. Validate incrementally
- Run the most relevant app/package/workspace checks first, then expand only if needed.
- If validation fails, determine whether the problem is in the code, test setup, or environment before retrying.

6. Commit, push, and open a pull request
- Commit only issue-relevant changes in the assigned branch.
- Push the assigned branch to the remote.
- Open a pull request targeting the repository default branch unless repository instructions say otherwise.

7. Report progress and failures clearly
- Use `gh issue comment` for progress updates, milestone updates, PR creation, and execution failures.
- Keep comments concise, factual, and tied to real progress.
