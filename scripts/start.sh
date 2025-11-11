#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"
STATIC_DIR="$BACKEND_DIR/static"
BIN_DIR="$ROOT_DIR/bin"
PLAYWRIGHT_DIR="$BIN_DIR/playwright"

echo "[0/5] 检查 Playwright 环境..."

# 检查 Playwright 驱动目录是否存在
if [ ! -d "$PLAYWRIGHT_DIR/playwright-driver" ]; then
    echo "⚠ Playwright 驱动未找到，请先运行 setup-playwright.sh"
    exit 1
fi

# 检查关键文件
if [ ! -f "$PLAYWRIGHT_DIR/playwright-driver/package.json" ]; then
    echo "⚠ Playwright 驱动不完整，请重新运行 setup-playwright.sh"
    exit 1
fi

echo "✓ Playwright 驱动已存在"

echo ""
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

# 设置 Playwright 环境变量
export PLAYWRIGHT_NODE_HOME="$PLAYWRIGHT_DIR/node"
export PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_DIR/browsers"
export PLAYWRIGHT_DRIVER_PATH="$PLAYWRIGHT_DIR/playwright-driver"
export PATH="$PLAYWRIGHT_DIR/node/bin:$PATH"

# 启动服务（环境变量会被子进程继承）
"$BIN_DIR/ai-trade-assistant" &
SERVER_PID=$!

cleanup() {
  echo "\n[5/5] 停止服务 (PID $SERVER_PID)"
  kill "$SERVER_PID" >/dev/null 2>&1 || true
}

trap cleanup EXIT

wait "$SERVER_PID"
