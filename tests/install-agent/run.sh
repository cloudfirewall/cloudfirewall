#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
IMAGE_NAME="cloudfirewall-install-agent-test"
VERSION="${VERSION:-v0.1.0-test}"

mkdir -p "$ROOT_DIR/dist/releases/$VERSION"
env GOCACHE=/tmp/go-build-cache VERSION="$VERSION" OUT_DIR="dist/releases/$VERSION" sh "$ROOT_DIR/scripts/package-agent-release.sh"

docker build -f "$ROOT_DIR/tests/install-agent/Dockerfile" -t "$IMAGE_NAME" "$ROOT_DIR"
docker run --rm \
  -e ROOT_DIR=/workspace \
  -e VERSION="$VERSION" \
  -v "$ROOT_DIR:/workspace" \
  "$IMAGE_NAME" \
  sh /workspace/tests/install-agent/assert.sh
