# Monorepo Service Launch Contract

`docker-compose-launch` is the shared integration point for monorepo issue-implementation skills when local services are required during implementation or testing.

## Contract

- Invoke `docker-compose-launch` only when the assigned worktree needs local dependencies to boot or validate the target code.
- Scope all launched services to the assigned issue worktree. Do not reuse shared or remote environments.
- Keep the contract generic so stack-specific implementation skills can request services without embedding compose logic in Vigilante core.

## Supported Service Types

- `mysql`
- `mariadb`
- `postgres`
- `mongodb`

## Expected Response Shape

The launch workflow should return enough connection data for the implementation skill to run app and test commands:

- `compose_project`
- `services[].name`
- `services[].type`
- `services[].host`
- `services[].port`
- `services[].database`
- `services[].username`
- `services[].password_env`
- `services[].connection_url_env`
