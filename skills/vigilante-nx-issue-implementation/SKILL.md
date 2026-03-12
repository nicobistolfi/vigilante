---
name: vigilante-nx-issue-implementation
description: Implement a Vigilante-dispatched issue in an Nx monorepo using the shared service-launch contract when local dependencies are required.
---

# Vigilante Nx Issue Implementation

Use this skill when Vigilante routes an issue from a repository classified as `monorepo` with stack `nx`.

## Workflow
- Treat the GitHub issue as the source of truth and keep changes minimal.
- Work from the repo root and prefer Nx project names and targets for scoped commands.
- Use the narrowest Nx task or package-manager wrapper that covers the affected project.
- If the prompt says local services are required, invoke the shared `docker-compose-launch` workflow exactly as described in the prompt contract and keep services scoped to the assigned worktree.

## Guardrails
- Do not hard-code service startup logic into the skill.
- Do not widen validation beyond the affected projects unless targeted checks are insufficient.
