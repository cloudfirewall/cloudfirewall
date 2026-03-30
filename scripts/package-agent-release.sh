#!/bin/sh
set -eu

VERSION="${VERSION:-}"
OUT_DIR="${OUT_DIR:-dist/releases}"
APP_NAME="cloudfirewall-agent"
TARGETS="${TARGETS:-amd64 arm64}"

usage() {
  cat <<'EOF'
Usage:
  VERSION=v0.1.0 scripts/package-agent-release.sh

Environment:
  VERSION   Release version tag, for example v0.1.0
  OUT_DIR   Output directory for packaged archives
  TARGETS   Space-separated GOARCH list, default: "amd64 arm64"
EOF
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

if [ -z "$VERSION" ]; then
  usage >&2
  exit 1
fi

need_cmd go
need_cmd tar
need_cmd mktemp
need_cmd install

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR_ABS="$ROOT_DIR/$OUT_DIR"
mkdir -p "$OUT_DIR_ABS"

for arch in $TARGETS; do
  case "$arch" in
    amd64|arm64) ;;
    *)
      echo "unsupported target architecture: $arch" >&2
      exit 1
      ;;
  esac

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT INT TERM

  binary_name="$APP_NAME"
  archive_name="${APP_NAME}_${VERSION}_linux_${arch}.tar.gz"

  (
    cd "$ROOT_DIR"
    CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -o "$tmp_dir/$binary_name" ./apps/agent/cmd/agent
  )

  tar -C "$tmp_dir" -czf "$OUT_DIR_ABS/$archive_name" "$binary_name"
  rm -rf "$tmp_dir"
  trap - EXIT INT TERM

  echo "wrote $OUT_DIR/$archive_name"
done
