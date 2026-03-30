#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/tests/e2e/docker-compose.yml"

cleanup() {
  docker compose -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
}

trap cleanup EXIT

run_scenario() {
  local scenario_name="$1"
  local api_config_path="$2"
  local probe_script="$3"

  echo "==> Running e2e scenario: $scenario_name"
  cleanup

  API_CONFIG_PATH="$api_config_path" docker compose -f "$COMPOSE_FILE" up -d --build api enrollment-token agent probe

  local deadline=$((SECONDS + 60))
  while (( SECONDS < deadline )); do
    local status_json
    status_json="$(curl -fsS -H 'X-API-Key: dev-api-key' http://localhost:8080/api/v1/agents || true)"
    if [[ -n "$status_json" ]] && grep -q '"online":true' <<<"$status_json"; then
      if docker compose -f "$COMPOSE_FILE" exec -T probe sh -ec "$probe_script"; then
        echo "Scenario passed: $scenario_name"
        echo "$status_json"
        return 0
      fi
    fi
    sleep 2
  done

  echo "Scenario failed: $scenario_name" >&2
  docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
  return 1
}

run_scenario \
  "public-web-server" \
  "/app/compiled/public-web-server.nft.golden" \
  "curl --connect-timeout 5 -fsS http://agent:443 >/dev/null 2>&1 && ! nc -zvw3 agent 22 >/dev/null 2>&1"

run_scenario \
  "risky-public-ssh" \
  "/app/compiled/risky-public-ssh.nft.golden" \
  "nc -zvw3 agent 22 >/dev/null 2>&1 && ! curl --connect-timeout 5 -fsS http://agent:443 >/dev/null 2>&1"

echo "All e2e scenarios passed"
