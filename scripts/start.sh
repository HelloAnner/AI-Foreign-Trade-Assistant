#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"
STATIC_DIR="$BACKEND_DIR/static"
BIN_DIR="$ROOT_DIR/bin"

echo "[1/5] 构建前端..."
npm --prefix "$FRONTEND_DIR" install >/dev/null
npm --prefix "$FRONTEND_DIR" run build

echo "[2/5] 同步前端静态资源..."
rm -rf "$STATIC_DIR"
mkdir -p "$STATIC_DIR"
cp -R "$FRONTEND_DIR/dist"/* "$STATIC_DIR"/

echo "[3/5] 构建后端可执行文件..."
mkdir -p "$BIN_DIR"
go build -C "$BACKEND_DIR" -o "$BIN_DIR/ai-trade-assistant"

echo "[4/5] 启动本地服务..."
"$BIN_DIR/ai-trade-assistant" &
SERVER_PID=$!

cleanup() {
  echo "\n[5/5] 停止服务 (PID $SERVER_PID)"
  kill "$SERVER_PID" >/dev/null 2>&1 || true
}

trap cleanup EXIT

wait "$SERVER_PID"
