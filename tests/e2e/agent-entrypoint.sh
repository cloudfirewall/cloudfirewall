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

exec /usr/local/bin/agent \
  -api-url "${CLOUDFIREWALL_API_URL}" \
  -hostname "${CLOUDFIREWALL_AGENT_HOSTNAME}" \
  -name "${CLOUDFIREWALL_AGENT_NAME}" \
  -agent-version "${CLOUDFIREWALL_AGENT_VERSION}" \
  -enrollment-token "${CLOUDFIREWALL_ENROLLMENT_TOKEN}" \
  -dry-run="${CLOUDFIREWALL_DRY_RUN}"
