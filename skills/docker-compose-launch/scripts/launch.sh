#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: launch.sh --worktree <path> --services <csv> [--dry-run]
EOF
}

fail() {
  printf '%s\n' "$*" >&2
  exit 1
}

json_escape() {
  local value
  value=${1//\\/\\\\}
  value=${value//\"/\\\"}
  value=${value//$'\n'/\\n}
  printf '%s' "$value"
}

sanitize_name() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr ' _/' '-' | tr -cd 'a-z0-9-'
}

checksum_mod_200() {
  printf '%s' "$1" | cksum | awk '{print $1 % 200}'
}

project_name_for_worktree() {
  local worktree base sum
  worktree=$1
  base=$(basename "$worktree")
  base=$(sanitize_name "$base")
  if [ -z "$base" ]; then
    base=worktree
  fi
  sum=$(printf '%s' "$worktree" | cksum | awk '{printf "%08x", $1}')
  printf '%s-%s' "$base" "${sum:0:8}"
}

compose_command_string() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    printf 'docker compose'
    return 0
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    printf 'docker-compose'
    return 0
  fi
  return 1
}

known_compose_files() {
  find "$1" -type f \( -name 'compose.yaml' -o -name 'compose.yml' -o -name 'docker-compose.yaml' -o -name 'docker-compose.yml' \) 2>/dev/null | sort
}

compose_file_supports_services() {
  local file service
  file=$1
  shift
  for service in "$@"; do
    if ! grep -Eq "^[[:space:]]{2}${service}:" "$file"; then
      return 1
    fi
  done
  return 0
}

check_port_available() {
  local port
  port=$1
  if command -v lsof >/dev/null 2>&1 && lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "selected host port ${port} is already in use"
  fi
}

append_connection_json() {
  local json=$1
  if [ -n "${connections_json:-}" ]; then
    connections_json="${connections_json},${json}"
  else
    connections_json=$json
  fi
}

append_service_json() {
  local name=$1
  if [ -n "${services_json:-}" ]; then
    services_json="${services_json},\"${name}\""
  else
    services_json="\"${name}\""
  fi
}

append_yaml() {
  yaml_lines="${yaml_lines}$1"$'\n'
}

build_generated_service() {
  local service worktree offset port
  service=$1
  worktree=$2
  offset=$(checksum_mod_200 "${worktree}-${service}")

  case "$service" in
    mysql)
      port=$((23306 + offset))
      check_port_available "$port"
      append_service_json "mysql"
      append_connection_json "{\"service\":\"mysql\",\"host\":\"127.0.0.1\",\"port\":${port},\"database\":\"app\",\"username\":\"app\",\"password\":\"app\",\"connection_uri\":\"mysql://app:app@127.0.0.1:${port}/app\"}"
      append_yaml "  mysql:"
      append_yaml "    image: mysql:8.4"
      append_yaml "    environment:"
      append_yaml "      MYSQL_DATABASE: app"
      append_yaml "      MYSQL_USER: app"
      append_yaml "      MYSQL_PASSWORD: app"
      append_yaml "      MYSQL_ROOT_PASSWORD: root"
      append_yaml "    ports:"
      append_yaml "      - \"${port}:3306\""
      ;;
    mariadb)
      port=$((23307 + offset))
      check_port_available "$port"
      append_service_json "mariadb"
      append_connection_json "{\"service\":\"mariadb\",\"host\":\"127.0.0.1\",\"port\":${port},\"database\":\"app\",\"username\":\"app\",\"password\":\"app\",\"connection_uri\":\"mysql://app:app@127.0.0.1:${port}/app\"}"
      append_yaml "  mariadb:"
      append_yaml "    image: mariadb:11"
      append_yaml "    environment:"
      append_yaml "      MARIADB_DATABASE: app"
      append_yaml "      MARIADB_USER: app"
      append_yaml "      MARIADB_PASSWORD: app"
      append_yaml "      MARIADB_ROOT_PASSWORD: root"
      append_yaml "    ports:"
      append_yaml "      - \"${port}:3306\""
      ;;
    postgres)
      port=$((25432 + offset))
      check_port_available "$port"
      append_service_json "postgres"
      append_connection_json "{\"service\":\"postgres\",\"host\":\"127.0.0.1\",\"port\":${port},\"database\":\"app\",\"username\":\"app\",\"password\":\"app\",\"connection_uri\":\"postgres://app:app@127.0.0.1:${port}/app?sslmode=disable\"}"
      append_yaml "  postgres:"
      append_yaml "    image: postgres:16"
      append_yaml "    environment:"
      append_yaml "      POSTGRES_DB: app"
      append_yaml "      POSTGRES_USER: app"
      append_yaml "      POSTGRES_PASSWORD: app"
      append_yaml "    ports:"
      append_yaml "      - \"${port}:5432\""
      ;;
    mongodb)
      port=$((27018 + offset))
      check_port_available "$port"
      append_service_json "mongodb"
      append_connection_json "{\"service\":\"mongodb\",\"host\":\"127.0.0.1\",\"port\":${port},\"database\":\"app\",\"username\":\"\",\"password\":\"\",\"connection_uri\":\"mongodb://127.0.0.1:${port}/app\"}"
      append_yaml "  mongodb:"
      append_yaml "    image: mongo:7"
      append_yaml "    ports:"
      append_yaml "      - \"${port}:27017\""
      ;;
    *)
      fail "unsupported database service: ${service}"
      ;;
  esac
}

worktree=
services_csv=
dry_run=0

while [ $# -gt 0 ]; do
  case "$1" in
    --worktree)
      worktree=${2-}
      shift 2
      ;;
    --services)
      services_csv=${2-}
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      fail "unknown argument: $1"
      ;;
  esac
done

[ -n "$worktree" ] || fail "missing required --worktree"
[ -d "$worktree" ] || fail "worktree does not exist: $worktree"
[ -n "$services_csv" ] || fail "missing required --services"

IFS=',' read -r -a requested_services <<<"$services_csv"
[ ${#requested_services[@]} -gt 0 ] || fail "at least one service must be requested"

normalized_services=()
for service in "${requested_services[@]}"; do
  normalized=$(sanitize_name "$service")
  [ -n "$normalized" ] || fail "service names must not be empty"
  normalized_services+=("$normalized")
done

compose_cmd=$(compose_command_string) || fail "docker compose unavailable: neither docker compose nor docker-compose is available"
project_name=$(project_name_for_worktree "$worktree")
compose_workdir=$worktree
compose_file=
launch_command=
cleanup_command=
services_json=
connections_json=

while IFS= read -r candidate; do
  [ -n "$candidate" ] || continue
  if compose_file_supports_services "$candidate" "${normalized_services[@]}"; then
    compose_file=$candidate
    compose_workdir=$(dirname "$candidate")
    break
  fi
done < <(known_compose_files "$worktree")

if [ -n "$compose_file" ]; then
  for service in "${normalized_services[@]}"; do
    append_service_json "$service"
    append_connection_json "{\"service\":\"${service}\",\"host\":\"127.0.0.1\",\"port\":0,\"database\":\"\",\"username\":\"\",\"password\":\"\",\"connection_uri\":\"repository-compose-managed\"}"
  done
else
  compose_file="$worktree/.vigilante/docker-compose.launch.yml"
  mkdir -p "$(dirname "$compose_file")"
  yaml_lines="services"$'\n'
  for service in "${normalized_services[@]}"; do
    build_generated_service "$service" "$worktree"
  done
  printf '%s' "$yaml_lines" >"$compose_file"
fi

launch_command="${compose_cmd} -f ${compose_file} -p ${project_name} up -d ${normalized_services[*]}"
cleanup_command="${compose_cmd} -f ${compose_file} -p ${project_name} down -v"

if [ "$dry_run" -ne 1 ]; then
  (
    cd "$compose_workdir"
    if [ "$compose_cmd" = "docker compose" ]; then
      docker compose -f "$compose_file" -p "$project_name" up -d "${normalized_services[@]}"
    else
      docker-compose -f "$compose_file" -p "$project_name" up -d "${normalized_services[@]}"
    fi
  )
fi

cat <<EOF
{
  "compose_command": "$(json_escape "$compose_cmd")",
  "compose_working_directory": "$(json_escape "$compose_workdir")",
  "compose_file_path": "$(json_escape "$compose_file")",
  "project_name": "$(json_escape "$project_name")",
  "launched_services": [${services_json}],
  "connections": [${connections_json}],
  "launch_command": "$(json_escape "$launch_command")",
  "cleanup_command": "$(json_escape "$cleanup_command")"
}
EOF
