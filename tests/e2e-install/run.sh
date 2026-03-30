#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/tests/e2e-install/docker-compose.yml"
VERSION="${VERSION:-v0.1.0-test}"
API_HOST_PORT="${API_HOST_PORT:-18080}"

cleanup() {
  docker compose -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
}

trap cleanup EXIT

mkdir -p "$ROOT_DIR/dist/releases/$VERSION"
env GOCACHE=/tmp/go-build-cache VERSION="$VERSION" OUT_DIR="dist/releases/$VERSION" sh "$ROOT_DIR/scripts/package-agent-release.sh"

api_get_agents() {
  curl -fsS -H 'X-API-Key: dev-api-key' "http://localhost:${API_HOST_PORT}/api/v1/agents" || true
}

wait_for_agents() {
  local expected_count="$1"
  local expected_version="$2"
  local deadline=$((SECONDS + 90))

  while (( SECONDS < deadline )); do
    local status_json
    status_json="$(api_get_agents)"
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

assert_installed_agent() {
  local service_name="$1"
  docker compose -f "$COMPOSE_FILE" exec -T "$service_name" sh -ec \
    'test -x /usr/local/bin/cloudfirewall-agent && test -f /etc/cloudfirewall/agent.env && test -f /etc/systemd/system/cloudfirewall-agent.service'
}

run_single_scenario() {
  local scenario_name="$1"
  local api_config_path="$2"
  local expected_version="$3"
  local probe_script="$4"

  echo "==> Running installer-driven e2e scenario: $scenario_name"
  cleanup

  API_CONFIG_PATH="$api_config_path" \
  AGENT_INSTALL_VERSION="$VERSION" \
  API_PUBLISHED_PORT="$API_HOST_PORT" \
  docker compose -f "$COMPOSE_FILE" up -d --build api enrollment-token agent probe

  local status_json
  if ! status_json="$(wait_for_agents 1 "$expected_version")"; then
    echo "Installer-driven e2e scenario failed: $scenario_name" >&2
    docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
    return 1
  fi

  assert_installed_agent agent
  docker compose -f "$COMPOSE_FILE" exec -T probe sh -ec "$probe_script"

  echo "Scenario passed: installer-$scenario_name"
  echo "$status_json"
}

run_rollout_scenario() {
  echo "==> Running installer-driven e2e scenario: rollout-two-agents"
  cleanup

  API_CONFIG_PATH="/app/compiled/public-web-server.nft.golden" \
  AGENT_INSTALL_VERSION="$VERSION" \
  API_PUBLISHED_PORT="$API_HOST_PORT" \
  docker compose -f "$COMPOSE_FILE" up -d --build \
    api enrollment-token enrollment-token-b agent agent-b probe

  local initial_version="sha256-03299926edb21be9"
  local rollout_version="rollout-risky-public-ssh"

  local initial_status
  if ! initial_status="$(wait_for_agents 2 "$initial_version")"; then
    echo "Initial installer rollout convergence failed" >&2
    docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
    return 1
  fi

  assert_installed_agent agent
  assert_installed_agent agent-b

  if ! docker compose -f "$COMPOSE_FILE" exec -T probe sh -ec \
    'nc -zvw3 agent 443 >/dev/null 2>&1 && nc -zvw3 agent-b 443 >/dev/null 2>&1 && ! nc -zvw3 agent 22 >/dev/null 2>&1 && ! nc -zvw3 agent-b 22 >/dev/null 2>&1'; then
    echo "Initial installer rollout probe assertions failed" >&2
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

  curl -fsS -X POST "http://localhost:${API_HOST_PORT}/api/v1/firewall-config" \
    -H 'X-API-Key: dev-api-key' \
    -H 'Content-Type: application/json' \
    -d "$rollout_config" >/dev/null

  local rollout_status
  if ! rollout_status="$(wait_for_agents 2 "$rollout_version")"; then
    echo "Installer rollout convergence failed" >&2
    docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
    return 1
  fi

  if ! docker compose -f "$COMPOSE_FILE" exec -T probe sh -ec \
    'nc -zvw3 agent 22 >/dev/null 2>&1 && nc -zvw3 agent-b 22 >/dev/null 2>&1 && ! nc -zvw3 agent 443 >/dev/null 2>&1 && ! nc -zvw3 agent-b 443 >/dev/null 2>&1'; then
    echo "Installer rolled out probe assertions failed" >&2
    docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
    return 1
  fi

  echo "Scenario passed: installer-rollout-two-agents"
  echo "$rollout_status"
}

run_single_scenario \
  "public-web-server" \
  "/app/compiled/public-web-server.nft.golden" \
  "sha256-03299926edb21be9" \
  'nc -zvw3 agent 443 >/dev/null 2>&1 && ! nc -zvw3 agent 22 >/dev/null 2>&1'

run_rollout_scenario

echo "All installer-driven e2e scenarios passed"
