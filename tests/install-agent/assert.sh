#!/bin/sh
set -eu

ROOT_DIR="${ROOT_DIR:-/workspace}"
VERSION="${VERSION:-v0.1.0-test}"
RELEASE_DIR="$ROOT_DIR/dist/releases/$VERSION"
ARCHIVE_PATH="$RELEASE_DIR/cloudfirewall-agent_${VERSION}_linux_amd64.tar.gz"
TMP_ROOT="/tmp/cloudfirewall-agent-test"
API_URL="http://api.example.internal:8080"
ENROLLMENT_TOKEN="test-enrollment-token"

if [ ! -f "$ARCHIVE_PATH" ]; then
  echo "missing archive: $ARCHIVE_PATH" >&2
  exit 1
fi

rm -rf "$TMP_ROOT"
mkdir -p "$TMP_ROOT"

sh "$ROOT_DIR/scripts/install-agent.sh" \
  --api-url "$API_URL" \
  --enrollment-token "$ENROLLMENT_TOKEN" \
  --name edge-01 \
  --hostname edge-01.local \
  --agent-version "$VERSION" \
  --dry-run true \
  --release-version "$VERSION" \
  --release-base-url "file://$ROOT_DIR/dist/releases" \
  --install-dir "$TMP_ROOT/opt/cloudfirewall-agent" \
  --binary-path "$TMP_ROOT/usr/local/bin/cloudfirewall-agent"

BINARY_PATH="$TMP_ROOT/usr/local/bin/cloudfirewall-agent"
ENV_FILE="/etc/cloudfirewall/agent.env"
SERVICE_FILE="/etc/systemd/system/cloudfirewall-agent.service"

test -x "$BINARY_PATH"
test -f "$ENV_FILE"
test -f "$SERVICE_FILE"

"$BINARY_PATH" -h >/tmp/cloudfirewall-agent-help.txt 2>&1 || true
grep -q "base URL of the cloudfirewall API" /tmp/cloudfirewall-agent-help.txt
grep -q "CLOUDFIREWALL_API_URL=$API_URL" "$ENV_FILE"
grep -q "CLOUDFIREWALL_ENROLLMENT_TOKEN=$ENROLLMENT_TOKEN" "$ENV_FILE"
grep -q "CLOUDFIREWALL_AGENT_NAME=edge-01" "$ENV_FILE"
grep -q "CLOUDFIREWALL_AGENT_HOSTNAME=edge-01.local" "$ENV_FILE"
grep -q "CLOUDFIREWALL_AGENT_VERSION=$VERSION" "$ENV_FILE"
grep -q "CLOUDFIREWALL_DRY_RUN=true" "$ENV_FILE"
grep -q "ExecStart=$BINARY_PATH" "$SERVICE_FILE"
grep -q "EnvironmentFile=$ENV_FILE" "$SERVICE_FILE"

echo "agent installer assertions passed"
