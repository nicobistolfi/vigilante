---
name: vigilante-bazel-issue-implementation
description: Implement a Vigilante-dispatched issue in a Bazel monorepo using the shared service-launch contract when local dependencies are required.
---

# Vigilante Bazel Issue Implementation

Use this skill when Vigilante routes an issue from a repository classified as `monorepo` with stack `bazel`.

## Workflow
- Treat the GitHub issue as the source of truth and keep changes minimal.
- Prefer Bazel targets scoped to the affected packages or applications.
- Use the smallest `bazel test` or `bazel run` target set that proves the change.
- If the prompt says local services are required, invoke the shared `docker-compose-launch` workflow exactly as described in the prompt contract and keep services scoped to the assigned worktree.

## Guardrails
- Do not embed service startup steps directly in Bazel commands.
- Do not expand validation to unrelated targets unless targeted coverage is insufficient.
