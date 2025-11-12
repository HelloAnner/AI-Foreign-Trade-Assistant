@echo off
chcp 65001 >nul

echo ================================================
echo    AI 外贸客户开发助手 - Windows 启动脚本
echo ================================================

REM 设置当前目录为脚本所在目录
cd /d "%~dp0"

REM 设置 Playwright 环境变量
set "PLAYWRIGHT_NODE_HOME=bin\playwright\node"
set "PLAYWRIGHT_BROWSERS_PATH=bin\playwright\browsers"
set "PLAYWRIGHT_DRIVER_PATH=bin\playwright\playwright-driver"

REM 添加 node 到 PATH
set "PATH=%PLAYWRIGHT_NODE_HOME%\bin;%PATH%"

echo 正在启动 AI 外贸客户开发助手...
echo Playwright 驱动路径: %PLAYWRIGHT_DRIVER_PATH%
echo 浏览器路径: %PLAYWRIGHT_BROWSERS_PATH%
echo.

REM 检查可执行文件是否存在
if not exist "ai-trade-assistant.exe" (
    echo 错误: 未找到可执行文件 ai-trade-assistant.exe
    echo 请确保在 dist 目录中运行此脚本
    pause
    exit /b 1
)

REM 启动应用程序
ai-trade-assistant.exe

REM 如果程序退出，暂停以便查看输出
if %errorlevel% neq 0 (
    echo.
    echo 程序异常退出，错误代码: %errorlevel%
    pause
)