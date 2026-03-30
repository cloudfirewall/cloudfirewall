#!/bin/sh
set -eu

export CLOUDFIREWALL_ENROLLMENT_TOKEN="$(cat /shared/enrollment.token)"

http_listener() {
  while true; do
    printf 'HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok' | nc -l -p 443 -q 1
  done
}

ssh_listener() {
  while true; do
    printf 'SSH-2.0-cloudfirewall\r\n' | nc -l -p 22 -q 1
  done
}

http_listener &
ssh_listener &

sh /workspace/scripts/install-agent.sh \
  --api-url "${CLOUDFIREWALL_API_URL}" \
  --enrollment-token "${CLOUDFIREWALL_ENROLLMENT_TOKEN}" \
  --name "${CLOUDFIREWALL_AGENT_NAME}" \
  --hostname "${CLOUDFIREWALL_AGENT_HOSTNAME}" \
  --agent-version "${CLOUDFIREWALL_AGENT_VERSION}" \
  --dry-run "${CLOUDFIREWALL_DRY_RUN}" \
  --release-version "${CLOUDFIREWALL_AGENT_RELEASE_VERSION}" \
  --release-base-url "${CLOUDFIREWALL_AGENT_RELEASE_BASE_URL}"

set -a
. /etc/cloudfirewall/agent.env
set +a

exec /usr/local/bin/cloudfirewall-agent
