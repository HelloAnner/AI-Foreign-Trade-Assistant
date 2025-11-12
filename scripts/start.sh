#!/bin/bash

# AI 外贸客户开发助手 - 开发环境启动脚本
# 用于在开发期间验证驱动加载，模拟真实启动环境

echo "================================================"
echo "   AI 外贸客户开发助手 - 开发环境启动脚本"
echo "================================================"

# 设置当前目录为项目根目录
cd "$(dirname "$0")/.."
PROJECT_ROOT="$(pwd)"

# 检查并安装 Playwright 环境
if [ ! -d "bin/playwright" ] || [ ! -d "bin/playwright/playwright-driver" ]; then
    echo "检测到缺少 Playwright 环境，正在自动安装..."

    # 创建目录结构
    mkdir -p bin/playwright
    cd bin/playwright

    # 下载并安装 Node.js
    echo "安装 Node.js..."
    if [[ "$(uname)" == "Darwin" ]]; then
        if [[ "$(uname -m)" == "arm64" ]]; then
            curl -L https://nodejs.org/dist/v20.11.0/node-v20.11.0-darwin-arm64.tar.gz | tar xz
            mv node-v20.11.0-darwin-arm64 node
        else
            curl -L https://nodejs.org/dist/v20.11.0/node-v20.11.0-darwin-x64.tar.gz | tar xz
            mv node-v20.11.0-darwin-x64 node
        fi
    else
        echo "错误: 不支持的操作系统 $(uname)"
        exit 1
    fi

    # 安装 Playwright NPM 包
    echo "安装 Playwright NPM 包..."
    cat > package.json << 'EOF'
{
  "name": "playwright",
  "version": "1.0.0",
  "description": "",
  "main": "index.js",
  "scripts": {
    "test": "echo \"Error: no test specified\" && exit 1"
  },
  "keywords": [],
  "author": "",
  "license": "ISC"
}
EOF

    ./node/bin/npm install playwright@1.49.1

    # 安装浏览器二进制文件
    echo "安装浏览器二进制文件..."
    ./node/bin/npx playwright install

    cd "$PROJECT_ROOT"
    echo "Playwright 环境安装完成！"
fi

# 使用 playwright-go 内置安装机制来确保驱动版本兼容性
if [ ! -d "bin/playwright/playwright-driver/package" ] || [ ! -f "bin/playwright/playwright-driver/package/package.json" ] || [ ! -f "bin/playwright/playwright-driver/package/index.js" ]; then
    echo "使用 playwright-go 内置机制安装驱动..."
    cd backend
    go run -tags playwright ../scripts/install-playwright-driver.go
    cd "$PROJECT_ROOT"
    echo "Playwright 驱动安装完成！"
fi

# 创建 playwright-go 缓存目录的符号链接，确保 playwright-go 能找到驱动
PLAYWRIGHT_GO_CACHE_DIR="$HOME/Library/Caches/ms-playwright-go/1.49.1"
if [ -d "$PLAYWRIGHT_GO_CACHE_DIR" ]; then
    echo "创建 playwright-go 缓存符号链接..."
    rm -rf "$PLAYWRIGHT_GO_CACHE_DIR/package"
    ln -s "$PROJECT_ROOT/bin/playwright/playwright-driver" "$PLAYWRIGHT_GO_CACHE_DIR/package"
    echo "符号链接创建完成: $PLAYWRIGHT_GO_CACHE_DIR/package -> $PROJECT_ROOT/bin/playwright/playwright-driver"
fi

# 设置 Playwright 环境变量 (使用绝对路径)
export PLAYWRIGHT_NODE_HOME="$PROJECT_ROOT/bin/playwright/node"
export PLAYWRIGHT_BROWSERS_PATH="$PROJECT_ROOT/bin/playwright/browsers"
export PLAYWRIGHT_DRIVER_PATH="$PROJECT_ROOT/bin/playwright/playwright-driver"

# 强制使用自定义驱动路径，禁用系统缓存
export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true
export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

# 添加 node 到 PATH
export PATH="$PLAYWRIGHT_NODE_HOME/bin:$PATH"

echo "正在启动 AI 外贸客户开发助手..."
echo "Playwright 驱动路径: $PLAYWRIGHT_DRIVER_PATH"
echo "浏览器路径: $PLAYWRIGHT_BROWSERS_PATH"
echo

# 编译并运行Go程序
echo "编译并启动应用程序..."
cd backend
PLAYWRIGHT_DRIVER_PATH="$PLAYWRIGHT_DRIVER_PATH" \
PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_BROWSERS_PATH" \
PLAYWRIGHT_NODE_HOME="$PLAYWRIGHT_NODE_HOME" \
PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true \
PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 \
go run -tags playwright main.go "$@"

# 如果程序退出，显示退出代码
if [ $? -ne 0 ]; then
    echo
    echo "程序异常退出，请检查日志文件"
fi