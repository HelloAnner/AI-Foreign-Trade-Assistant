#!/bin/bash

# AI 外贸客户开发助手 - 打包脚本
# 基于 start.sh 逻辑创建完整的打包方案
# 确保打包后的环境解压即用，包含完整的 Playwright 环境

set -e

echo "================================================"
echo "   AI 外贸客户开发助手 - 打包脚本"
echo "================================================"

# 设置当前目录为项目根目录
PROJECT_DIR="$(pwd)"
DIST_DIR="dist"
BACKEND_DIR="backend"
BIN_DIR="bin"
SCRIPTS_DIR="scripts"
PACKAGE_NAME="ai-foreign-trade-assistant"

# 清理并重新创建 dist 目录
echo "清理打包目录..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# 创建打包目录结构
echo "创建打包目录结构..."
PACKAGE_DIR="$DIST_DIR/$PACKAGE_NAME"
mkdir -p "$PACKAGE_DIR/bin"
mkdir -p "$PACKAGE_DIR/scripts"

# 检查并确保 Playwright 环境存在
if [ ! -d "bin/playwright" ] || [ ! -d "bin/playwright/playwright-driver" ]; then
    echo "检测到缺少 Playwright 环境，正在自动安装..."
    ./scripts/start.sh --install-only
fi

echo "正在编译应用程序..."

# 编译 Windows 版本
echo "编译 Windows 版本..."
cd "$BACKEND_DIR"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags playwright -o "../$PACKAGE_DIR/bin/ai-trade-assistant.exe" main.go
if [ $? -ne 0 ]; then
    echo "错误: Windows版本编译失败"
    exit 1
fi

# 编译 macOS 版本
echo "编译 macOS 版本..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags playwright -o "../$PACKAGE_DIR/bin/ai-trade-assistant-macos" main.go
if [ $? -ne 0 ]; then
    echo "错误: macOS版本编译失败"
    exit 1
fi

# 编译 Linux 版本
echo "编译 Linux 版本..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags playwright -o "../$PACKAGE_DIR/bin/ai-trade-assistant-linux" main.go
if [ $? -ne 0 ]; then
    echo "错误: Linux版本编译失败"
    exit 1
fi

cd "../$PACKAGE_DIR"

# 复制静态资源
echo "复制静态资源..."
cp -r "../../backend/static" "./"

# 复制 Playwright 环境
echo "复制 Playwright 环境..."
cp -r "../../bin/playwright" "./bin/"

# 创建启动脚本
echo "创建启动脚本..."

# Linux/macOS 启动脚本
cat > "start.sh" << 'EOF'
#!/bin/bash

# AI 外贸客户开发助手 - 启动脚本
# 解压即用版本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "================================================"
echo "   AI 外贸客户开发助手 - 启动中..."
echo "================================================"

# 设置 Playwright 环境变量 (使用绝对路径)
export PLAYWRIGHT_NODE_HOME="$SCRIPT_DIR/bin/playwright/node"
export PLAYWRIGHT_BROWSERS_PATH="$SCRIPT_DIR/bin/playwright/browsers"
export PLAYWRIGHT_DRIVER_PATH="$SCRIPT_DIR/bin/playwright/playwright-driver"

# 强制使用自定义驱动路径，禁用系统缓存
export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true
export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

# 添加 node 到 PATH
export PATH="$PLAYWRIGHT_NODE_HOME/bin:$PATH"

echo "Playwright 驱动路径: $PLAYWRIGHT_DRIVER_PATH"
echo "浏览器路径: $PLAYWRIGHT_BROWSERS_PATH"
echo

# 检测操作系统并启动对应的可执行文件
OS_NAME="$(uname -s)"
ARCH_NAME="$(uname -m)"

echo "检测到系统: $OS_NAME $ARCH_NAME"

case "$OS_NAME" in
    Darwin)
        # macOS
        if [ -f "./bin/ai-trade-assistant-macos" ]; then
            echo "正在启动 macOS 版本..."
            ./bin/ai-trade-assistant-macos "$@"
        else
            echo "错误: 找不到 macOS 可执行文件"
            exit 1
        fi
        ;;
    Linux)
        # Linux
        if [ -f "./bin/ai-trade-assistant-linux" ]; then
            echo "正在启动 Linux 版本..."
            ./bin/ai-trade-assistant-linux "$@"
        else
            echo "错误: 找不到 Linux 可执行文件"
            exit 1
        fi
        ;;
    *)
        echo "错误: 不支持的操作系统: $OS_NAME"
        echo "请使用 Windows 版本或联系技术支持"
        exit 1
        ;;
esac
EOF

chmod +x "start.sh"

# Windows 启动脚本
cat > "start.bat" << 'EOF'
@echo off
chcp 65001 >nul

echo ================================================
echo    AI 外贸客户开发助手 - 启动中...
echo ================================================

REM 设置 Playwright 环境变量
set PLAYWRIGHT_NODE_HOME=%~dp0bin\playwright\node
set PLAYWRIGHT_BROWSERS_PATH=%~dp0bin\playwright\browsers
set PLAYWRIGHT_DRIVER_PATH=%~dp0bin\playwright\playwright-driver

REM 强制使用自定义驱动路径，禁用系统缓存
set PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true
set PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

REM 添加 node 到 PATH
set PATH=%PLAYWRIGHT_NODE_HOME%\bin;%PATH%

echo Playwright 驱动路径: %PLAYWRIGHT_DRIVER_PATH%
echo 浏览器路径: %PLAYWRIGHT_BROWSERS_PATH%
echo.

REM 启动应用程序
echo 正在启动 AI 外贸客户开发助手...
%~dp0bin\ai-trade-assistant.exe %*
EOF

# 创建使用说明
echo "创建使用说明..."
cat > "README.md" << 'EOF'
# AI 外贸客户开发助手

## 简介
AI 外贸客户开发助手是一个基于人工智能的外贸客户开发工具，支持自动化搜索、分析和邮件营销功能。

## 系统要求
- Linux/macOS/Windows 系统
- 至少 2GB 可用内存
- 网络连接

## 快速开始

### Linux/macOS
```bash
./start.sh
```

### Windows
```bash
start.bat
```

## 启动参数
- `--port <端口号>`: 指定服务端口 (默认: 8080)
- `--help`: 显示帮助信息

## 数据存储
应用数据存储在用户主目录的 `.foreign_trade` 目录下：
- 配置文件: `~/.foreign_trade/config.json`
- 数据库: `~/.foreign_trade/app.db`
- 日志文件: `~/.foreign_trade/logs/`

## 技术支持
如遇到问题，请检查日志文件或联系技术支持。
EOF

# 创建安装脚本
echo "创建安装脚本..."
cat > "scripts/install.sh" << 'EOF'
#!/bin/bash

# AI 外贸客户开发助手 - 安装脚本

set -e

echo "================================================"
echo "   AI 外贸客户开发助手 - 系统安装"
echo "================================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/usr/local/bin/ai-trade-assistant"

# 检查权限
if [ "$EUID" -ne 0 ]; then
    echo "请使用 sudo 运行此脚本: sudo ./scripts/install.sh"
    exit 1
fi

# 创建符号链接
echo "创建系统符号链接..."
OS_NAME="$(uname -s)"
case "$OS_NAME" in
    Darwin)
        ln -sf "$SCRIPT_DIR/../bin/ai-trade-assistant-macos" "$INSTALL_DIR"
        ;;
    Linux)
        ln -sf "$SCRIPT_DIR/../bin/ai-trade-assistant-linux" "$INSTALL_DIR"
        ;;
    *)
        echo "错误: 不支持的操作系统: $OS_NAME"
        exit 1
        ;;
esac

# 设置权限
chmod +x "$INSTALL_DIR"

echo "安装完成！"
echo "现在可以通过 'ai-trade-assistant' 命令启动应用程序"
echo "或者使用 './start.sh' 在当前目录启动"
EOF

chmod +x "scripts/install.sh"

# 创建压缩包
echo "创建压缩包..."
cd "../"

# 检测操作系统并创建相应的压缩包
case "$(uname -s)" in
    Darwin)
        # macOS
        tar -czf "${PACKAGE_NAME}-macos.tar.gz" "$PACKAGE_NAME"
        echo "已创建: ${PACKAGE_NAME}-macos.tar.gz"
        ;;
    Linux)
        # Linux
        if [ "$(uname -m)" = "aarch64" ]; then
            tar -czf "${PACKAGE_NAME}-linux-arm64.tar.gz" "$PACKAGE_NAME"
            echo "已创建: ${PACKAGE_NAME}-linux-arm64.tar.gz"
        else
            tar -czf "${PACKAGE_NAME}-linux-amd64.tar.gz" "$PACKAGE_NAME"
            echo "已创建: ${PACKAGE_NAME}-linux-amd64.tar.gz"
        fi
        ;;
    CYGWIN*|MINGW32*|MINGW64*|MSYS*)
        # Windows
        zip -r "${PACKAGE_NAME}-windows.zip" "$PACKAGE_NAME"
        echo "已创建: ${PACKAGE_NAME}-windows.zip"
        ;;
    *)
        echo "未知操作系统: $(uname -s)"
        tar -czf "${PACKAGE_NAME}-unknown.tar.gz" "$PACKAGE_NAME"
        echo "已创建: ${PACKAGE_NAME}-unknown.tar.gz"
        ;;
esac

# 显示打包结果
echo
echo "================================================"
echo "   打包完成！"
echo "================================================"
echo "打包目录: $PACKAGE_DIR"
echo "压缩包位置: $DIST_DIR/"
echo
echo "包含内容:"
echo "- Go 可执行文件: bin/ai-trade-assistant-* (多平台)"
echo "- Playwright 环境: bin/playwright/"
echo "- 前端静态资源: static/"
echo "- 启动脚本: start.sh (Linux/macOS)"
echo "- 启动脚本: start.bat (Windows)"
echo "- 使用说明: README.md"
echo "- 安装脚本: scripts/install.sh"
echo
echo "使用方法:"
echo "1. 解压压缩包"
echo "2. 运行 ./start.sh (Linux/macOS) 或 start.bat (Windows)"
echo "3. 访问 http://localhost:8080"
echo

# 显示压缩包大小
cd "$DIST_DIR"
for file in *.tar.gz *.zip; do
    if [ -f "$file" ]; then
        size=$(du -h "$file" | cut -f1)
        echo "压缩包大小: $file - $size"
    fi
done

cd "$PROJECT_DIR"

echo
echo "注意: 交付包已包含完整的 Playwright 驱动和浏览器环境，无需额外安装！"
echo "基于 start.sh 逻辑，确保解压即用，驱动路径配置正确。"