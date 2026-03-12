---
name: vigilante-rush-issue-implementation
description: Implement a Vigilante-dispatched issue in a Rush monorepo using the shared service-launch contract when local dependencies are required.
---

# Vigilante Rush Issue Implementation

Use this skill when Vigilante routes an issue from a repository classified as `monorepo` with stack `rush`.

## Workflow
- Treat the GitHub issue as the source of truth and keep changes minimal.
- Work from the repo root and prefer Rush-managed commands for scoped validation.
- Use the narrowest Rush command that matches the affected packages.
- If the prompt says local services are required, invoke the shared `docker-compose-launch` workflow exactly as described in the prompt contract and keep services scoped to the assigned worktree.

## Guardrails
- Do not invent per-stack service orchestration behavior.
- Do not run broad workspace commands unless targeted checks are unavailable.
