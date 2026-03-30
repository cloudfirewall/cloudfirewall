#!/bin/sh
set -eu

API_URL="${CLOUDFIREWALL_API_URL:-}"
ENROLLMENT_TOKEN="${CLOUDFIREWALL_ENROLLMENT_TOKEN:-}"
AGENT_NAME="${CLOUDFIREWALL_AGENT_NAME:-}"
AGENT_HOSTNAME="${CLOUDFIREWALL_AGENT_HOSTNAME:-}"
AGENT_VERSION="${CLOUDFIREWALL_AGENT_VERSION:-0.1.0}"
DRY_RUN="${CLOUDFIREWALL_DRY_RUN:-false}"
BINARY_URL="${CLOUDFIREWALL_AGENT_BINARY_URL:-}"
REPO_URL="${CLOUDFIREWALL_REPO_URL:-https://github.com/cloudfirewall/cloudfirewall}"
REPO_REF="${CLOUDFIREWALL_REPO_REF:-main}"
INSTALL_DIR="${CLOUDFIREWALL_AGENT_INSTALL_DIR:-/opt/cloudfirewall-agent}"
BINARY_PATH="${CLOUDFIREWALL_AGENT_BINARY_PATH:-/usr/local/bin/cloudfirewall-agent}"
ENV_DIR="${CLOUDFIREWALL_AGENT_ENV_DIR:-/etc/cloudfirewall}"
ENV_FILE="$ENV_DIR/agent.env"
SERVICE_PATH="${CLOUDFIREWALL_AGENT_SERVICE_PATH:-/etc/systemd/system/cloudfirewall-agent.service}"

usage() {
  cat <<'EOF'
Usage: install-agent.sh --api-url URL --enrollment-token TOKEN [options]

Options:
  --api-url URL
  --enrollment-token TOKEN
  --name NAME
  --hostname HOSTNAME
  --agent-version VERSION
  --dry-run true|false
  --binary-url URL
  --repo-url URL
  --repo-ref REF
  --install-dir DIR
  --binary-path PATH
EOF
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

normalize_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

install_from_binary_url() {
  need_cmd curl
  install -d "$(dirname "$BINARY_PATH")"
  tmp_binary="$(mktemp)"
  trap 'rm -f "$tmp_binary"' EXIT INT TERM
  curl -fsSL "$BINARY_URL" -o "$tmp_binary"
  install -m 0755 "$tmp_binary" "$BINARY_PATH"
  rm -f "$tmp_binary"
  trap - EXIT INT TERM
}

install_from_source() {
  need_cmd curl
  need_cmd tar
  need_cmd go

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT INT TERM
  archive="$tmp_dir/source.tar.gz"
  curl -fsSL "$REPO_URL/archive/refs/heads/$REPO_REF.tar.gz" -o "$archive"
  tar -xzf "$archive" -C "$tmp_dir"

  src_dir="$(find "$tmp_dir" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
  if [ -z "$src_dir" ]; then
    echo "failed to unpack source archive" >&2
    exit 1
  fi

  install -d "$(dirname "$BINARY_PATH")"
  (
    cd "$src_dir"
    CGO_ENABLED=0 GOOS=linux GOARCH="$(normalize_arch)" go build -o "$BINARY_PATH" ./apps/agent/cmd/agent
  )
  trap - EXIT INT TERM
  rm -rf "$tmp_dir"
}

write_env_file() {
  install -d "$ENV_DIR" "$INSTALL_DIR"
  cat >"$ENV_FILE" <<EOF
CLOUDFIREWALL_API_URL=$API_URL
CLOUDFIREWALL_ENROLLMENT_TOKEN=$ENROLLMENT_TOKEN
CLOUDFIREWALL_AGENT_NAME=$AGENT_NAME
CLOUDFIREWALL_AGENT_HOSTNAME=$AGENT_HOSTNAME
CLOUDFIREWALL_AGENT_VERSION=$AGENT_VERSION
CLOUDFIREWALL_DRY_RUN=$DRY_RUN
EOF
  chmod 0600 "$ENV_FILE"
}

write_service_file() {
  cat >"$SERVICE_PATH" <<EOF
[Unit]
Description=Cloudfirewall Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$BINARY_PATH
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
}

start_service() {
  if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable --now cloudfirewall-agent.service
    systemctl status --no-pager cloudfirewall-agent.service || true
    return
  fi

  echo "systemd not found; installed binary at $BINARY_PATH" >&2
  echo "run manually with: $BINARY_PATH" >&2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --api-url)
      API_URL="$2"
      shift 2
      ;;
    --enrollment-token)
      ENROLLMENT_TOKEN="$2"
      shift 2
      ;;
    --name)
      AGENT_NAME="$2"
      shift 2
      ;;
    --hostname)
      AGENT_HOSTNAME="$2"
      shift 2
      ;;
    --agent-version)
      AGENT_VERSION="$2"
      shift 2
      ;;
    --dry-run)
      DRY_RUN="$2"
      shift 2
      ;;
    --binary-url)
      BINARY_URL="$2"
      shift 2
      ;;
    --repo-url)
      REPO_URL="$2"
      shift 2
      ;;
    --repo-ref)
      REPO_REF="$2"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="$2"
      shift 2
      ;;
    --binary-path)
      BINARY_PATH="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  echo "run this installer as root (for example with sudo)" >&2
  exit 1
fi

if [ -z "$API_URL" ] || [ -z "$ENROLLMENT_TOKEN" ]; then
  usage >&2
  exit 1
fi

need_cmd install

if [ -n "$BINARY_URL" ]; then
  install_from_binary_url
else
  install_from_source
fi

write_env_file
write_service_file
start_service

echo "cloudfirewall agent installed"
echo "binary: $BINARY_PATH"
echo "env:    $ENV_FILE"
echo "unit:   $SERVICE_PATH"
