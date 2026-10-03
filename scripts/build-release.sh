#!/usr/bin/env bash
# Reproducible release build for supported platforms. Embed version metadata.
set -euo pipefail
VERSION="${VERSION:-0.1.0-dev}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"
DATE="$(date -u +%Y-%m-%d)"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${DATE}"
mkdir -p dist
for t in linux/amd64 linux/arm64 darwin/arm64 darwin/amd64 windows/amd64; do
  os=${t%/*}; arch=${t#*/}
  ext=""; [ "$os" = "windows" ] && ext=".exe"
  echo "building $os/$arch"
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -ldflags "$LDFLAGS" -o "dist/sy-${os}-${arch}${ext}" ./cmd/sy
done
echo "dist:"
ls -la dist/
