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

wait_for_agents() {
  local expected_count="$1"
  local expected_version="$2"
  local deadline=$((SECONDS + 60))

  while (( SECONDS < deadline )); do
    local status_json
    status_json="$(curl -fsS -H 'X-API-Key: dev-api-key' http://localhost:8080/api/v1/agents || true)"
    if [[ -n "$status_json" ]]; then
      local online_count
      online_count="$(grep -o '"online":true' <<<"$status_json" | wc -l | tr -d ' ')"
      local version_count
      version_count="$(grep -o "\"firewallVersion\":\"$expected_version\"" <<<"$status_json" | wc -l | tr -d ' ')"
      if [[ "$online_count" == "$expected_count" && "$version_count" == "$expected_count" ]]; then
        echo "$status_json"
        return 0
      fi
    fi
    sleep 2
  done

  return 1
}

run_rollout_scenario() {
  echo "==> Running e2e scenario: rollout-two-agents"
  cleanup

  API_CONFIG_PATH="/app/compiled/public-web-server.nft.golden" docker compose -f "$COMPOSE_FILE" up -d --build \
    api enrollment-token enrollment-token-b agent agent-b probe

  local initial_version="sha256-03299926edb21be9"
  local rollout_version="rollout-risky-public-ssh"

  local initial_status
  if ! initial_status="$(wait_for_agents 2 "$initial_version")"; then
    echo "Initial two-agent convergence failed" >&2
    docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
    return 1
  fi
  echo "$initial_status"

  if ! docker compose -f "$COMPOSE_FILE" exec -T probe sh -ec \
    "nc -zvw3 agent 443 >/dev/null 2>&1 && nc -zvw3 agent-b 443 >/dev/null 2>&1 && ! nc -zvw3 agent 22 >/dev/null 2>&1 && ! nc -zvw3 agent-b 22 >/dev/null 2>&1"; then
    echo "Initial rollout probe assertions failed" >&2
    docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
    return 1
  fi

  local rollout_config
  rollout_config="$(python3 - <<'PY'
import json
from pathlib import Path
content = Path("apps/engine/testdata/compiled/risky-public-ssh.nft.golden").read_text()
print(json.dumps({"version": "rollout-risky-public-ssh", "nftablesConfig": content}))
PY
)"

  curl -fsS -X POST http://localhost:8080/api/v1/firewall-config \
    -H 'X-API-Key: dev-api-key' \
    -H 'Content-Type: application/json' \
    -d "$rollout_config" >/dev/null

  local rollout_status
  if ! rollout_status="$(wait_for_agents 2 "$rollout_version")"; then
    echo "Rollout convergence failed" >&2
    docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
    return 1
  fi

  if ! docker compose -f "$COMPOSE_FILE" exec -T probe sh -ec \
    "nc -zvw3 agent 22 >/dev/null 2>&1 && nc -zvw3 agent-b 22 >/dev/null 2>&1 && ! nc -zvw3 agent 443 >/dev/null 2>&1 && ! nc -zvw3 agent-b 443 >/dev/null 2>&1"; then
    echo "Rolled out probe assertions failed" >&2
    docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
    return 1
  fi

  echo "Scenario passed: rollout-two-agents"
  echo "$rollout_status"
}

run_scenario \
  "public-web-server" \
  "/app/compiled/public-web-server.nft.golden" \
  "nc -zvw3 agent 443 >/dev/null 2>&1 && ! nc -zvw3 agent 22 >/dev/null 2>&1"

run_scenario \
  "risky-public-ssh" \
  "/app/compiled/risky-public-ssh.nft.golden" \
  "nc -zvw3 agent 22 >/dev/null 2>&1 && ! nc -zvw3 agent 443 >/dev/null 2>&1"

run_rollout_scenario

echo "All e2e scenarios passed"
