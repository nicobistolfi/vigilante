---
name: vigilante-gradle-issue-implementation
description: Implement a Vigilante-dispatched issue in a Gradle multi-project monorepo using the shared service-launch contract when local dependencies are required.
---

# Vigilante Gradle Issue Implementation

Use this skill when Vigilante routes an issue from a repository classified as `monorepo` with stack `gradle`.

## Workflow
- Treat the GitHub issue as the source of truth and keep changes minimal.
- Prefer Gradle project paths and targeted root tasks from the repo root.
- Use the narrowest `./gradlew` command that validates the affected module set.
- If the prompt says local services are required, invoke the shared `docker-compose-launch` workflow exactly as described in the prompt contract and keep services scoped to the assigned worktree.

## Guardrails
- Do not duplicate the shared service-launch contract inside project-specific scripts.
- Do not expand validation to unrelated modules unless targeted checks are insufficient.
