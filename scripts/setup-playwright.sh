#!/bin/bash

# AI 外贸助手 - Playwright 安装脚本
# 使用方法：bash scripts/setup-playwright.sh [output-dir]

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PLAYWRIGHT_VERSION="1.40.0"  # 固定版本，确保稳定性

# 输出目录（默认为 bin/playwright）
OUTPUT_DIR="${1:-$ROOT_DIR/bin/playwright}"
NODE_DIR="$OUTPUT_DIR/node"
BROWSER_DIR="$OUTPUT_DIR/browsers"

echo "📦 开始安装 Playwright..."
echo "输出目录：$OUTPUT_DIR"

# 确保目录存在
mkdir -p "$OUTPUT_DIR"
mkdir -p "$NODE_DIR"
mkdir -p "$BROWSER_DIR"

# 1. 安装 Node.js 运行时（如果不存在）
if [ ! -f "$NODE_DIR/bin/node" ]; then
    echo "[1/4] 下载 Node.js..."

    # 检测平台
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux)
            NODE_OS="linux"
            ;;
        darwin)
            NODE_OS="darwin"
            ;;
        msys*|mingw*|cygwin*)
            NODE_OS="win"
            ;;
        *)
            echo "❌ 不支持的操作系统: $OS"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64)
            NODE_ARCH="x64"
            ;;
        arm64|aarch64)
            NODE_ARCH="arm64"
            ;;
        *)
            echo "❌ 不支持的架构: $ARCH"
            exit 1
            ;;
    esac

    NODE_VERSION="20.11.0"
    NODE_FILENAME="node-v${NODE_VERSION}-${NODE_OS}-${NODE_ARCH}"

    if [ "$NODE_OS" = "win" ]; then
        NODE_FILENAME="${NODE_FILENAME}.zip"
    else
        NODE_FILENAME="${NODE_FILENAME}.tar.gz"
    fi

    # 下载 Node.js
    cd /tmp
    curl -L -o "$NODE_FILENAME" "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_FILENAME}"

    # 解压
    if [ "$NODE_OS" = "win" ]; then
        unzip -q "$NODE_FILENAME"
        mv "node-v${NODE_VERSION}-${NODE_OS}-${NODE_ARCH}" node-tmp
    else
        tar -xzf "$NODE_FILENAME"
        mv "node-v${NODE_VERSION}-${NODE_OS}-${NODE_ARCH}" node-tmp
    fi

    # 移动到目标位置
    mv node-tmp/* "$NODE_DIR/"
    rm -rf node-tmp
    rm "$NODE_FILENAME"

    echo "✓ Node.js 安装完成"
else
    echo "✓ Node.js 已存在，跳过下载"
fi

# 2. 安装 Playwright NPM 包
echo "[2/4] 安装 Playwright NPM 包..."

cd "$OUTPUT_DIR"

# 初始化 package.json（如果不存在）
if [ ! -f "package.json" ]; then
    "$NODE_DIR/bin/npm" init -y
fi

# 安装 Playwright CLI
"$NODE_DIR/bin/npm" install --save-dev @playwright/test@"$PLAYWRIGHT_VERSION"

# 3. 安装浏览器
echo "[3/4] 安装 Playwright 浏览器..."

# 设置浏览器安装路径
export PLAYWRIGHT_BROWSERS_PATH="$BROWSER_DIR"

# 安装 Chromium
"$NODE_DIR/bin/npx" playwright install chromium

echo "✓ 浏览器安装完成"

# 4. 创建启动脚本
echo "[4/4] 创建启动脚本..."

cat > "$OUTPUT_DIR/playwright-path.sh" << 'EOF'
#!/bin/bash
# 设置 Playwright 环境变量
DIR="$(cd "$(dirname "$0")" && pwd)"
export PLAYWRIGHT_NODE_HOME="$DIR/node"
export PLAYWRIGHT_BROWSERS_PATH="$DIR/browsers"
export PATH="$DIR/node/bin:$PATH"
EOF

chmod +x "$OUTPUT_DIR/playwright-path.sh"

echo ""
echo "🎉 Playwright 安装完成！"
echo ""
echo "使用说明："
echo "1. 使用前执行：source $OUTPUT_DIR/playwright-path.sh"
echo "2. 验证安装：npx playwright --version"
echo "3. 浏览器位置：$BROWSER_DIR"
echo ""
