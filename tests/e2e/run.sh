#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/tests/e2e/docker-compose.yml"

cleanup() {
  docker compose -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true
}

trap cleanup EXIT

docker compose -f "$COMPOSE_FILE" up -d --build api enrollment-token agent probe

deadline=$((SECONDS + 60))
while (( SECONDS < deadline )); do
  status_json="$(curl -fsS -H 'X-API-Key: dev-api-key' http://localhost:8080/api/v1/agents || true)"
  if [[ -n "$status_json" ]] && grep -q '"online":true' <<<"$status_json"; then
    if docker compose -f "$COMPOSE_FILE" exec -T probe sh -ec \
      "curl --connect-timeout 5 -fsS http://agent:443 >/dev/null 2>&1 && ! nc -zvw3 agent 22 >/dev/null 2>&1"; then
      echo "E2E agent reported online and probe assertions passed"
      echo "$status_json"
      exit 0
    fi
  fi
  sleep 2
done

echo "E2E test timed out waiting for an online agent and successful probe assertions" >&2
docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
exit 1
