#!/bin/bash

# AI 外贸客户开发助手 - 交付环境启动脚本
# 客户使用此脚本启动应用程序

echo "================================================"
echo "   AI 外贸客户开发助手 - 交付环境启动脚本"
echo "================================================"

# 设置当前目录为脚本所在目录
cd "$(dirname "$0")"

# 设置 Playwright 环境变量
export PLAYWRIGHT_NODE_HOME="bin/playwright/node"
export PLAYWRIGHT_BROWSERS_PATH="bin/playwright/browsers"
export PLAYWRIGHT_DRIVER_PATH="bin/playwright/playwright-driver"

# 添加 node 到 PATH
export PATH="$PLAYWRIGHT_NODE_HOME/bin:$PATH"

echo "正在启动 AI 外贸客户开发助手..."
echo "Playwright 驱动路径: $PLAYWRIGHT_DRIVER_PATH"
echo "浏览器路径: $PLAYWRIGHT_BROWSERS_PATH"
echo

# 根据平台选择可执行文件
if [[ "$OSTYPE" == "darwin"* ]]; then
    EXECUTABLE="ai-trade-assistant-macos"
    echo "检测到 macOS 系统，使用: $EXECUTABLE"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    EXECUTABLE="ai-trade-assistant-linux"
    echo "检测到 Linux 系统，使用: $EXECUTABLE"
else
    echo "错误: 不支持的操作系统: $OSTYPE"
    exit 1
fi

# 检查可执行文件是否存在
if [ ! -f "$EXECUTABLE" ]; then
    echo "错误: 未找到可执行文件 $EXECUTABLE"
    echo "请确保在 dist 目录中运行此脚本"
    exit 1
fi

# 给可执行文件添加执行权限（如果需要）
chmod +x "$EXECUTABLE" 2>/dev/null

# 启动应用程序
./"$EXECUTABLE" "$@"

# 如果程序退出，显示退出代码
if [ $? -ne 0 ]; then
    echo
    echo "程序异常退出，请检查日志文件"
fi