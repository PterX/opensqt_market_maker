#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="opensqt_market_maker"

cd "$ROOT_DIR"

extract_version() {
  grep -E '^var Version = "[^"]+"' main.go | sed -E 's/^var Version = "([^"]+)"$/\1/'
}

VERSION="${VERSION:-$(extract_version)}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
DIST_DIR="$ROOT_DIR/dist"
PACKAGE_BASENAME="${APP_NAME}_${VERSION}_${GOOS}_${GOARCH}"
STAGE_DIR="$DIST_DIR/$PACKAGE_BASENAME"
ARCHIVE_PATH="$DIST_DIR/${PACKAGE_BASENAME}.tar.gz"
CHECKSUM_PATH="$ARCHIVE_PATH.sha256"
BIN_PATH="$STAGE_DIR/$APP_NAME"

if [[ -z "$VERSION" ]]; then
  echo "无法从 main.go 提取版本号" >&2
  exit 1
fi

rm -rf "$STAGE_DIR" "$ARCHIVE_PATH" "$CHECKSUM_PATH"
mkdir -p "$STAGE_DIR"

echo "==> 构建 $APP_NAME $VERSION ($GOOS/$GOARCH)"
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$BIN_PATH" .

echo "==> 准备发布包内容"
cp README.md "$STAGE_DIR/README.md"
cp ARCHITECTURE.md "$STAGE_DIR/ARCHITECTURE.md"
cp 部署教程.pdf "$STAGE_DIR/部署教程.pdf"
cp config.example.yaml "$STAGE_DIR/config.example.yaml"
cp config.example.yaml "$STAGE_DIR/config.yaml"
cp -R live_server "$STAGE_DIR/live_server"

echo "==> 打包归档"
tar -C "$DIST_DIR" -czf "$ARCHIVE_PATH" "$PACKAGE_BASENAME"
sha256sum "$ARCHIVE_PATH" > "$CHECKSUM_PATH"

echo "==> 发布包已生成"
echo "$ARCHIVE_PATH"
echo "$CHECKSUM_PATH"
