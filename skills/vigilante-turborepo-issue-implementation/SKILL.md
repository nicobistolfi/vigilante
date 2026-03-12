---
name: vigilante-turborepo-issue-implementation
description: Implement a Vigilante-dispatched issue in a Turborepo monorepo using the shared service-launch contract when local dependencies are required.
---

# Vigilante Turborepo Issue Implementation

Use this skill when Vigilante routes an issue from a repository classified as `monorepo` with stack `turborepo`.

## Workflow
- Treat the GitHub issue as the source of truth and keep changes minimal.
- Work from the repo root unless the issue clearly scopes work to one workspace.
- Use Turbo filters and task pipelines for targeted build, test, and lint commands.
- If the prompt says local services are required, invoke the shared `docker-compose-launch` workflow exactly as described in the prompt contract and keep services scoped to the assigned worktree.
- Fall back to repo-native commands only when Turbo does not provide a narrower equivalent.

## Guardrails
- Do not invent new service-launch behavior outside the shared contract.
- Do not broaden validation beyond the affected workspace(s) unless targeted checks are insufficient.
