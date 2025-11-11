#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"
STATIC_DIR="$BACKEND_DIR/static"
DIST_DIR="$ROOT_DIR/dist"
PLAYWRIGHT_SCRIPT="$ROOT_DIR/scripts/setup-playwright.sh"

echo "[1/8] 构建前端..."
npm --prefix "$FRONTEND_DIR" install >/dev/null
npm --prefix "$FRONTEND_DIR" run build

echo "[2/8] 同步前端静态资源..."
rm -rf "$STATIC_DIR"
mkdir -p "$STATIC_DIR"
cp -R "$FRONTEND_DIR/dist"/* "$STATIC_DIR"/

echo "[3/8] 清理 dist 目录..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "[4/8] 构建 Windows (amd64) 可执行文件..."
GOOS=windows GOARCH=amd64 go build -C "$BACKEND_DIR" -ldflags "-H=windowsgui" -o "$DIST_DIR/windows/AI_Trade_Assistant.exe"

# 下载 Windows Playwright
echo "[5/8] 下载 Windows Playwright 环境..."
bash "$PLAYWRIGHT_SCRIPT" "$DIST_DIR/windows" "windows" "amd64"

echo "[6/8] 构建 macOS (arm64) 可执行文件..."
GOOS=darwin GOARCH=arm64 go build -C "$BACKEND_DIR" -o "$DIST_DIR/macos/AI_Trade_Assistant"

# 下载 macOS Playwright
echo "[7/8] 下载 macOS Playwright 环境..."
mkdir -p "$DIST_DIR/macos"
bash "$PLAYWRIGHT_SCRIPT" "$DIST_DIR/macos"

# 8. 验证打包完整性
echo "[8/8] 验证打包完整性..."

verify_package() {
    local platform=$1
    local exe_path=$2
    local playwright_dir=$3

    echo ""
    echo "验证 $platform 打包:"

    # 检查可执行文件
    if [ -f "$exe_path" ]; then
        size=$(ls -lh "$exe_path" | awk '{print $5}')
        echo "  ✓ 可执行文件: $(basename $exe_path) ($size)"
    else
        echo "  ✗ 缺少可执行文件: $exe_path"
        return 1
    fi

    # 检查 Playwright 目录
    if [ ! -d "$playwright_dir" ]; then
        echo "  ✗ 缺少 playwright 目录: $playwright_dir"
        return 1
    fi

    # 检查驱动
    if [ -f "$playwright_dir/playwright-driver/package.json" ]; then
        echo "  ✓ Go 驱动已包含"
    else
        echo "  ✗ 缺少 Go 驱动"
        return 1
    fi

    # 检查浏览器
    if [ -d "$playwright_dir/browsers" ]; then
        browser_count=$(ls -1 "$playwright_dir/browsers" | wc -l)
        echo "  ✓ 浏览器: $browser_count 个"
    else
        echo "  ⚠ 浏览器目录可能不完整"
    fi

    # 检查 Node
    if [ -f "$playwright_dir/node/bin/node" ]; then
        echo "  ✓ Node.js runtime"
    else
        echo "  ✗ 缺少 Node.js"
        return 1
    fi

    return 0
}

WINDOWS_EXE="$DIST_DIR/windows/AI_Trade_Assistant.exe"
WINDOWS_PLAYWRIGHT="$DIST_DIR/windows"

MACOS_EXE="$DIST_DIR/macos/AI_Trade_Assistant"
MACOS_PLAYWRIGHT="$DIST_DIR/macos"

# 验证 Windows 包
if [ -f "$WINDOWS_EXE" ]; then
    verify_package "Windows" "$WINDOWS_EXE" "$WINDOWS_PLAYWRIGHT"
    WINDOWS_OK=$?
else
    echo "⚠ Windows 可执行文件不存在，跳过验证"
    WINDOWS_OK=1
fi

# 验证 macOS 包
if [ -f "$MACOS_EXE" ]; then
    verify_package "macOS" "$MACOS_EXE" "$MACOS_PLAYWRIGHT"
    MACOS_OK=$?
else
    echo "⚠ macOS 可执行文件不存在，跳过验证"
    MACOS_OK=0
fi

echo ""
if [ $WINDOWS_OK -eq 0 ] && [ $MACOS_OK -eq 0 ]; then
    echo "✅ 打包完成！"
else
    echo "❌ 打包验证发现问题，请检查输出"
    exit 1
fi

echo ""
echo "==================== 构建总结 ===================="
echo ""
echo "✅ 所有平台打包完成！"
echo ""
echo "📦 打包输出:"
echo ""

# 显示 Windows 构建状态
if [ -f "$WINDOWS_EXE" ]; then
    size=$(ls -lh "$WINDOWS_EXE" | awk '{print $5}')
    echo "   Windows 可执行文件: $size"
else
    echo "   ❌ Windows 可执行文件: 构建失败"
fi

# 检查 Windows Playwright
if [ -d "$WINDOWS_PLAYWRIGHT/playwright-driver" ]; then
    browser_count=$(ls -1 "$WINDOWS_PLAYWRIGHT/browsers" 2>/dev/null | wc -l)
    echo "   Windows Playwright: ✓ 驱动 + $browser_count 个浏览器"
else
    echo "   ❌ Windows Playwright: 安装失败"
fi
echo ""

# 显示 macOS 构建状态
if [ -f "$MACOS_EXE" ]; then
    size=$(ls -lh "$MACOS_EXE" | awk '{print $5}')
    echo "   macOS 可执行文件: $size"
else
    echo "   ❌ macOS 可执行文件: 构建失败"
fi

# 检查 macOS Playwright
if [ -d "$MACOS_PLAYWRIGHT/playwright-driver" ]; then
    browser_count=$(ls -1 "$MACOS_PLAYWRIGHT/browsers" 2>/dev/null | wc -l)
    echo "   macOS Playwright: ✓ 驱动 + $browser_count 个浏览器"
else
    echo "   ❌ macOS Playwright: 安装失败"
fi
echo ""
echo "=================================================="
echo ""
echo "📂 输出目录:"
echo "   $DIST_DIR"
echo ""
echo "🚀 使用说明:"
echo ""
echo "   Windows 版本:"
echo "     $DIST_DIR/windows/AI_Trade_Assistant.exe"
echo ""
echo "   macOS 版本:"
echo "     $DIST_DIR/macos/AI_Trade_Assistant"
echo ""
echo "   两个版本都包含完整的 Playwright 环境"
echo "   可以直接在对应平台上运行"
echo ""
