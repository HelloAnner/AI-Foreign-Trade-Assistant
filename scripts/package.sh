#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"
STATIC_DIR="$BACKEND_DIR/static"
DIST_DIR="$ROOT_DIR/dist"

echo "[1/5] 构建前端..."
npm --prefix "$FRONTEND_DIR" install >/dev/null
npm --prefix "$FRONTEND_DIR" run build

echo "[2/5] 同步前端静态资源..."
rm -rf "$STATIC_DIR"
mkdir -p "$STATIC_DIR"
cp -R "$FRONTEND_DIR/dist"/* "$STATIC_DIR"/

echo "[3/5] 清理 dist 目录..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "[4/5] 构建 Windows (amd64) 可执行文件..."
GOOS=windows GOARCH=amd64 go build -C "$BACKEND_DIR" -ldflags "-H=windowsgui" -o "$DIST_DIR/AI_Trade_Assistant_Windows.exe"

echo "[5/5] 构建 macOS (arm64) 可执行文件..."
GOOS=darwin GOARCH=arm64 go build -C "$BACKEND_DIR" -o "$DIST_DIR/AI_Trade_Assistant_macOS_arm64"

echo "打包完成，产物位于 $DIST_DIR"
