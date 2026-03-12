---
name: docker-compose-launch
description: Launch worktree-local database dependencies with repository compose reuse when available, or generate a minimal Docker Compose stack for common databases when needed.
---

# Docker Compose Launch

Use this skill from issue-implementation skills when the assigned repository needs local database services before app boot, migrations, or integration tests can run.

## Inputs
- assigned worktree path
- required database services from this set:
  - MySQL
  - MariaDB
  - Postgres
  - MongoDB

## Workflow
1. Inspect the repository first
- Read the repository docs for local development and test setup before launching anything.
- Prefer repository-owned compose files or documented startup commands when they clearly cover the required database services.

2. Choose the compose CLI deterministically
- Prefer `docker compose` when available.
- Fall back to `docker-compose` only when the plugin form is unavailable.
- If neither command works, stop and report the failure clearly to the parent implementation flow.

3. Reuse repository compose assets when suitable
- Look for repository-owned compose files in the assigned worktree.
- Reuse them when they obviously define the required database service images or ports.
- Keep the compose project name namespaced to the assigned worktree so concurrent issue sessions remain isolated.

4. Generate a worktree-local compose file only when necessary
- If the repository does not provide suitable compose assets, create a minimal compose file inside the assigned worktree under a tool-owned folder such as `.vigilante/docker-compose/`.
- Support these default local services:
  - MySQL
  - MariaDB
  - Postgres
  - MongoDB
- Keep generated names, networks, and volumes namespaced to the assigned worktree or session.

5. Launch and report outputs
- Start the compose stack in detached mode.
- Return structured runtime details the caller can continue with:
  - launched service names
  - compose command form used
  - compose file path
  - working directory
  - compose project name
  - host ports and connection strings when known
  - cleanup expectation, typically the matching `down -v` command

## Guardrails
- Scope all generated files to the assigned worktree.
- Do not modify repository-owned compose files.
- Keep credentials local and non-secret; use disposable development defaults only.
- Surface failures clearly, including missing Docker, unsupported services, and port-allocation errors.

## Notes
- This skill exists for reuse by implementation skills; it is not a direct end-user entrypoint.
- Keep the behavior minimal and predictable so stack-specific skills can build on it.
