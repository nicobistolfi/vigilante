# Monorepo Service-Launch Contract

Vigilante exposes a shared `docker-compose-launch` handoff for stack-specific monorepo implementation skills.

## Contract

- Invoke `docker-compose-launch` only when local services are required to boot or test the target codebase.
- Scope launched services to the assigned issue worktree.
- The request may include any combination of `mysql`, `mariadb`, `postgres`, and `mongodb`.
- The launcher should return enough connection details for implementation and test commands to use the started services.
- The contract is limited to local implementation and test dependencies. It is not for shared or remote environments.

## Prompt Fields

Issue prompts pass the following fields to monorepo implementation skills:

- repository shape
- monorepo stack
- workspace hints
- process hints
- whether local services are required
- requested local service types
- the `docker-compose-launch` scope and purpose
