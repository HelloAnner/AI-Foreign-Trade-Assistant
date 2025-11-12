#!/bin/bash

# AI 外贸客户开发助手 - 打包脚本

echo "================================================"
echo "   AI 外贸客户开发助手 - 打包脚本"
echo "================================================"

# 设置变量
PROJECT_DIR="$(pwd)"
DIST_DIR="dist"
BACKEND_DIR="backend"
BIN_DIR="bin"
SCRIPTS_DIR="scripts"
PACKAGE_NAME="ai-foreign-trade-assistant"

# 清理旧的dist目录
rm -rf "$DIST_DIR"

# 创建dist目录
mkdir -p "$DIST_DIR"

echo "正在编译应用程序..."

# 编译Windows版本
echo "编译Windows版本..."
cd "$BACKEND_DIR"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags playwright -o "../$DIST_DIR/ai-trade-assistant.exe" main.go
if [ $? -ne 0 ]; then
    echo "错误: Windows版本编译失败"
    exit 1
fi

# 编译macOS版本
echo "编译macOS版本..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags playwright -o "../$DIST_DIR/ai-trade-assistant-macos" main.go
if [ $? -ne 0 ]; then
    echo "错误: macOS版本编译失败"
    exit 1
fi

# 编译Linux版本
echo "编译Linux版本..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags playwright -o "../$DIST_DIR/ai-trade-assistant-linux" main.go
if [ $? -ne 0 ]; then
    echo "错误: Linux版本编译失败"
    exit 1
fi

cd "../$DIST_DIR"

# 复制交付启动脚本
echo "复制交付启动脚本..."
cp "../$SCRIPTS_DIR/package/start.bat" ./
cp "../$SCRIPTS_DIR/package/start.sh" ./
chmod +x start.sh

# 复制Playwright驱动和浏览器
echo "复制Playwright驱动和浏览器..."
cp -r "../$BIN_DIR/playwright" ./

# 创建README文件
echo "创建使用说明..."
cat > README.txt << 'EOF'
AI 外贸客户开发助手 - 使用说明
================================

本交付包包含完整的AI外贸客户开发助手应用程序，支持Windows、macOS和Linux系统。

快速启动:
=========

Windows用户:
  双击运行 start.bat

macOS/Linux用户:
  在终端中运行: ./start.sh

文件结构:
========
- ai-trade-assistant.exe        Windows可执行文件
- ai-trade-assistant-macos      macOS可执行文件
- ai-trade-assistant-linux      Linux可执行文件
- start.bat                     Windows启动脚本
- start.sh                      macOS/Linux启动脚本
- playwright/                   Playwright驱动和浏览器

注意事项:
========
1. 请确保整个dist目录结构完整，不要移动或删除任何文件
2. 首次启动可能需要较长时间，因为需要初始化浏览器
3. 应用程序将在 http://localhost:8080 启动
4. 日志文件将保存在用户主目录的 .foreign_trade/logs 目录下

技术支持:
========
如遇问题，请检查:
1. 是否所有文件都在dist目录中
2. 系统是否满足内存和网络要求
3. 防火墙是否允许应用程序访问网络

EOF

echo "创建zip交付包..."
cd "../"
zip -r "$PACKAGE_NAME.zip" "$DIST_DIR" -x "*.DS_Store"

# 计算文件大小
FILE_SIZE=$(du -h "$PACKAGE_NAME.zip" | cut -f1)

echo "================================================"
echo "  打包完成！"
echo "================================================"
echo "交付包: $PACKAGE_NAME.zip ($FILE_SIZE)"
echo "输出目录: $DIST_DIR"
echo
echo "交付包内容:"
echo "- ai-trade-assistant.exe        Windows可执行文件"
echo "- ai-trade-assistant-macos      macOS可执行文件"
echo "- ai-trade-assistant-linux      Linux可执行文件"
echo "- start.bat                     Windows启动脚本"
echo "- start.sh                      macOS/Linux启动脚本"
echo "- playwright/                   Playwright驱动和浏览器"
echo "- README.txt                    使用说明"
echo
echo "客户使用说明:"
echo "1. 解压zip包到任意目录"
echo "2. Windows: 双击运行 dist/start.bat"
echo "3. macOS/Linux: 在终端中运行 dist/start.sh"
echo "4. 应用程序将在 http://localhost:8080 启动"
echo
echo "注意: 交付包已包含完整的Playwright驱动和浏览器环境，无需额外安装！"