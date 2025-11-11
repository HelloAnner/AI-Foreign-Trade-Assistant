#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"
STATIC_DIR="$BACKEND_DIR/static"
BIN_DIR="$ROOT_DIR/bin"
PLAYWRIGHT_DIR="$BIN_DIR/playwright"

echo "[0/5] 检查 Playwright 环境..."

# 检查 Playwright 是否已安装且完整
is_playwright_ready() {
    local dir="$1"
    # 检查关键文件和目录是否存在且有实际内容
    if [ -d "$dir/node/bin" ] && [ -f "$dir/node/bin/node" ] && [ -s "$dir/node/bin/node" ] &&
       [ -f "$dir/package.json" ] && [ -s "$dir/package.json" ] &&
       [ -d "$dir/browsers" ] && [ "$(ls -A "$dir/browsers" 2>/dev/null | wc -l)" -gt 0 ] &&
       [ -d "$dir/node_modules" ] && [ "$(ls -A "$dir/node_modules" 2>/dev/null | wc -l)" -gt 0 ]; then
        return 0  # 已就绪
    else
        return 1  # 不完整或不存在
    fi
}

if ! is_playwright_ready "$PLAYWRIGHT_DIR"; then
    if [ -d "$PLAYWRIGHT_DIR" ]; then
        echo "⚠ Playwright 环境不完整，删除后重新下载..."
        rm -rf "$PLAYWRIGHT_DIR"
    else
        echo "⚠ Playwright 环境未找到，开始自动下载..."
    fi
    bash "$ROOT_DIR/scripts/setup-playwright.sh" "$PLAYWRIGHT_DIR"
    if [ $? -ne 0 ]; then
        echo "❌ Playwright 下载失败"
        exit 1
    fi
else
    echo "✓ Playwright 环境已存在且完整"
fi

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
