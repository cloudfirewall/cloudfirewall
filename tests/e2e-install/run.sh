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

echo "==> Running installer-driven e2e scenario: public-web-server"
cleanup

API_CONFIG_PATH="/app/compiled/public-web-server.nft.golden" \
AGENT_INSTALL_VERSION="$VERSION" \
API_PUBLISHED_PORT="$API_HOST_PORT" \
docker compose -f "$COMPOSE_FILE" up -d --build api enrollment-token agent probe

deadline=$((SECONDS + 90))
while (( SECONDS < deadline )); do
  status_json="$(curl -fsS -H 'X-API-Key: dev-api-key' "http://localhost:${API_HOST_PORT}/api/v1/agents" || true)"
  if [[ -n "$status_json" ]] && grep -q '"online":true' <<<"$status_json"; then
    if docker compose -f "$COMPOSE_FILE" exec -T agent sh -ec \
      'test -x /usr/local/bin/cloudfirewall-agent && test -f /etc/cloudfirewall/agent.env && test -f /etc/systemd/system/cloudfirewall-agent.service'; then
      if docker compose -f "$COMPOSE_FILE" exec -T probe sh -ec \
        'nc -zvw3 agent 443 >/dev/null 2>&1 && ! nc -zvw3 agent 22 >/dev/null 2>&1'; then
        echo "Scenario passed: installer-public-web-server"
        echo "$status_json"
        exit 0
      fi
    fi
  fi
  sleep 2
done

echo "Installer-driven e2e scenario failed" >&2
docker compose -f "$COMPOSE_FILE" logs --no-color >&2 || true
exit 1
