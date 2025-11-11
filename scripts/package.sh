#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"
STATIC_DIR="$BACKEND_DIR/static"
DIST_DIR="$ROOT_DIR/dist"
PLAYWRIGHT_SCRIPT="$ROOT_DIR/scripts/setup-playwright.sh"

echo "[1/7] 构建前端..."
npm --prefix "$FRONTEND_DIR" install >/dev/null
npm --prefix "$FRONTEND_DIR" run build

echo "[2/7] 同步前端静态资源..."
rm -rf "$STATIC_DIR"
mkdir -p "$STATIC_DIR"
cp -R "$FRONTEND_DIR/dist"/* "$STATIC_DIR"/

echo "[3/7] 清理 dist 目录..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "[4/7] 构建 Windows (amd64) 可执行文件..."
GOOS=windows GOARCH=amd64 go build -C "$BACKEND_DIR" -ldflags "-H=windowsgui" -o "$DIST_DIR/AI_Trade_Assistant_Windows.exe"

# 下载 Windows Playwright
echo "[5/7] 下载 Windows Playwright 环境..."
mkdir -p "$DIST_DIR/windows"
bash "$PLAYWRIGHT_SCRIPT" "$DIST_DIR/windows" "windows" "amd64"

echo "[6/7] 构建 macOS (arm64) 可执行文件..."
GOOS=darwin GOARCH=arm64 go build -C "$BACKEND_DIR" -o "$DIST_DIR/macos/AI_Trade_Assistant"

# 下载 macOS Playwright
echo "[7/7] 下载 macOS Playwright 环境..."
mkdir -p "$DIST_DIR/macos"
bash "$PLAYWRIGHT_SCRIPT" "$DIST_DIR/macos"

echo "✓ 打包完成，产物位于 $DIST_DIR"
echo ""
echo "目录结构："
echo "  windows/"
echo "    ├── AI_Trade_Assistant.exe"
echo "    └── playwright/"
echo "  macos/"
echo "    ├── AI_Trade_Assistant"
echo "    └── playwright/"
echo ""
echo "使用说明："
echo "  1. Windows: 双击 windows/AI_Trade_Assistant.exe"
echo "  2. macOS: ./macos/AI_Trade_Assistant"
echo "  程序会自动使用当前目录下的 playwright 环境"
