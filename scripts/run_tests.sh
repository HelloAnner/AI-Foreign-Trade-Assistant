#!/bin/bash

# AI 外贸助手 - 单元测试选择器
# 使用方法：
#   1. 交互模式：cd backend && bash ../scripts/run_tests.sh
#   2. 执行所有测试：cd backend && bash ../scripts/run_tests.sh all

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# 项目根目录
ROOT_DIR="$(dirname "$0")/.."
# 后端项目根目录
BACKEND_DIR="$ROOT_DIR/backend"

# 显示环境变量说明（表格形式）
print_env_requirements() {
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}                        📋 环境变量配置检查表${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${MAGENTA}🌐 搜索模块配置（必需）${NC}"
    echo -e "${CYAN}┌──────────────┬──────────────┬─────────────────────────────────────────┐${NC}"
    echo -e "${CYAN}│  变量名      │  当前状态    │  说明                                   │${NC}"
    echo -e "${CYAN}├──────────────┼──────────────┼─────────────────────────────────────────┤${NC}"
    if [ -n "${SerpApi}" ]; then
        echo -e "${CYAN}│  SerpApi     │${GREEN} ✓ 已设置     ${CYAN}│  SerpApi 的 API Key                    │${NC}"
    else
        echo -e "${CYAN}│  SerpApi     │${RED} ✗ 未设置     ${CYAN}│  SerpApi 的 API Key                    │${NC}"
    fi
    if [ -n "${SERPAPI_API_KEY}" ]; then
        echo -e "${CYAN}│  SERPAPI_... │${GREEN} ✓ 已设置     ${CYAN}│  SerpApi 的 API Key（替代变量名）      │${NC}"
    else
        echo -e "${CYAN}│  SERPAPI_... │${RED} ✗ 未设置     ${CYAN}│  SerpApi 的 API Key（替代变量名）      │${NC}"
    fi
    echo -e "${CYAN}│              │              │  获取地址: https://serpapi.com/        │${NC}"
    echo -e "${CYAN}└──────────────┴──────────────┴─────────────────────────────────────────┘${NC}"
    echo ""

    echo -e "${MAGENTA}🤖 LLM 大模型配置（必需）${NC}"
    echo -e "${CYAN}┌──────────────┬──────────────┬─────────────────────────────────────────┐${NC}"
    echo -e "${CYAN}│  变量名      │  当前状态    │  说明                                   │${NC}"
    echo -e "${CYAN}├──────────────┼──────────────┼─────────────────────────────────────────┤${NC}"
    if [ -n "${LLM_API_KEY}" ]; then
        echo -e "${CYAN}│  LLM_API_KEY │${GREEN} ✓ 已设置     ${CYAN}│  LLM API 访问密钥                      │${NC}"
    else
        echo -e "${CYAN}│  LLM_API_KEY │${RED} ✗ 未设置     ${CYAN}│  LLM API 访问密钥                      │${NC}"
    fi
    if [ -n "${LLM_BASE_URL}" ]; then
        echo -e "${CYAN}│  LLM_BASE_URL│${GREEN} ✓ 已设置     ${CYAN}│  API 基础地址                          │${NC}"
    else
        echo -e "${CYAN}│  LLM_BASE_URL│${RED} ✗ 未设置     ${CYAN}│  API 基础地址                          │${NC}"
    fi
    if [ -n "${LLM_MODEL_NAME}" ]; then
        echo -e "${CYAN}│  LLM_MODEL...│${GREEN} ✓ 已设置     ${CYAN}│  模型名称                              │${NC}"
    else
        echo -e "${CYAN}│  LLM_MODEL...│${RED} ✗ 未设置     ${CYAN}│  模型名称                              │${NC}"
    fi
    echo -e "${CYAN}│              │              │  示例地址:                             │${CYAN}"
    echo -e "${CYAN}│              │              │  https://ark.cn-beijing.volces.com/... │${CYAN}"
    echo -e "${CYAN}│              │              │  示例模型: deepseek-v3-250324          │${CYAN}"
    echo -e "${CYAN}└──────────────┴──────────────┴─────────────────────────────────────────┘${NC}"
    echo ""

    echo -e "${MAGENTA}📧 SMTP 邮件配置（可选）${NC}"
    echo -e "${CYAN}┌──────────────┬──────────────┬─────────────────────────────────────────┐${NC}"
    echo -e "${CYAN}│  变量名      │  当前状态    │  说明                                   │${NC}"
    echo -e "${CYAN}├──────────────┼──────────────┼─────────────────────────────────────────┤${NC}"
    if [ -n "${SMTP_HOST}" ]; then
        echo -e "${CYAN}│  SMTP_HOST   │${GREEN} ✓ 已设置     ${CYAN}│  SMTP 服务器地址                       │${NC}"
    else
        echo -e "${CYAN}│  SMTP_HOST   │${YELLOW} ⚠ 可选       ${CYAN}│  SMTP 服务器地址                       │${NC}"
    fi
    if [ -n "${SMTP_PORT}" ]; then
        echo -e "${CYAN}│  SMTP_PORT   │${GREEN} ✓ 已设置     ${CYAN}│  SMTP 端口号                           │${NC}"
    else
        echo -e "${CYAN}│  SMTP_PORT   │${YELLOW} ⚠ 可选       ${CYAN}│  SMTP 端口号                           │${NC}"
    fi
    if [ -n "${SMTP_USERNAME}" ]; then
        echo -e "${CYAN}│  SMTP_USERN..│${GREEN} ✓ 已设置     ${CYAN}│  SMTP 用户名                           │${NC}"
    else
        echo -e "${CYAN}│  SMTP_USERN..│${YELLOW} ⚠ 可选       ${CYAN}│  SMTP 用户名                           │${NC}"
    fi
    if [ -n "${SMTP_PASSWORD}" ]; then
        echo -e "${CYAN}│  SMTP_PASSW..│${GREEN} ✓ 已设置     ${CYAN}│  SMTP 授权码                           │${NC}"
    else
        echo -e "${CYAN}│  SMTP_PASSW..│${YELLOW} ⚠ 可选       ${CYAN}│  SMTP 授权码                           │${NC}"
    fi
    echo -e "${CYAN}└──────────────┴──────────────┴─────────────────────────────────────────┘${NC}"
    echo ""

    echo -e "${MAGENTA}💡 提示${NC}"
    echo "  1. 可以将环境变量配置写入 .env 文件，然后执行 source .env"
    echo "  2. 示例文件位于: scripts/.env.example"
    echo ""
}

# 执行测试模块
run_test() {
    local test_type=$1

    # 显示环境变量检查表（在测试运行前）
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}                         📝 测试运行前配置检查${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    print_env_requirements

    # 设置 Playwright 环境变量
    PLAYWRIGHT_DIR="$(dirname "$0")/../bin/playwright"

    # 使用 is_playwright_ready 函数检测
    if ! is_playwright_ready "$PLAYWRIGHT_DIR"; then
        echo ""
        echo -e "${RED}❌ Playwright 环境不完整或未找到${NC}"
        echo "  请先运行: bash scripts/setup-playwright.sh"
        exit 1
    fi

    export PLAYWRIGHT_NODE_HOME="$PLAYWRIGHT_DIR/node"
    export PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_DIR/browsers"
    export PATH="$PLAYWRIGHT_DIR/node/bin:$PATH"
    echo ""
    echo -e "${GREEN}✓ Playwright 环境已配置${NC}"
    echo "  节点路径: $PLAYWRIGHT_NODE_HOME"
    echo "  浏览器路径: $PLAYWRIGHT_BROWSERS_PATH"
    echo ""

    # 使用 -count=1 禁用测试缓存，确保每次都真实运行
    local test_flags="-v -count=1"

    case $test_type in
        "search")
            echo -e "${YELLOW}正在运行搜索模块测试...${NC}"
            (cd "$BACKEND_DIR" && go test $test_flags ./services -run "TestSearch")
            ;;
        "llm")
            echo -e "${YELLOW}正在运行 LLM 模块测试...${NC}"
            (cd "$BACKEND_DIR" && go test $test_flags ./services -run "TestLLM")
            ;;
        "store")
            echo -e "${YELLOW}正在运行存储模块测试...${NC}"
            (cd "$BACKEND_DIR" && go test $test_flags ./store)
            ;;
        "mail")
            echo -e "${YELLOW}正在运行邮件模块测试...${NC}"
            (cd "$BACKEND_DIR" && go test $test_flags ./services -run "TestMail")
            ;;
        "accuracy")
            echo -e "${YELLOW}正在运行搜索准确率测试...${NC}"
            (cd "$BACKEND_DIR" && go test $test_flags ./services -run "TestExternalDependencies")
            (cd "$BACKEND_DIR" && go test $test_flags ./services -run "TestSearchAccuracy")
            ;;
        "deps")
            echo -e "${YELLOW}正在运行外部依赖检查...${NC}"
            (cd "$BACKEND_DIR" && go test $test_flags ./services -run "TestExternalDependencies")
            ;;
        "all")
            echo -e "${YELLOW}正在运行所有测试...${NC}"
            (cd "$BACKEND_DIR" && go test $test_flags ./...)
            ;;
        *)
            echo -e "${RED}无效的测试类型: $test_type${NC}"
            return 1
            ;;
    esac

    echo ""
    echo -e "${GREEN}✓ 测试完成${NC}"
    echo ""
}

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}                       🤖 AI 外贸助手 - 单元测试选择器${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

### 命令行帮助 ###
show_usage() {
    echo "使用方法："
    echo "  $(basename $0) [选项]"
    echo ""
    echo "示例："
    echo "  $(basename $0)              # 进入交互模式"
    echo "  $(basename $0) all          # 运行所有测试"
    echo "  $(basename $0) accuracy     # 运行搜索准确率测试"
    echo ""
    echo "选项说明："
    echo "  all       运行所有测试（./...）"
    echo "  search    搜索模块测试（搜索查询构建、结果处理等）"
    echo "  llm       LLM 模块测试（LLM 连接和对话功能）"
    echo "  store     存储模块测试（数据库操作和持久化）"
    echo "  mail      邮件模块测试（SMTP 配置和邮件模板）"
    echo "  accuracy  搜索准确率测试（需要 SerpApi 和 LLM）"
    echo "  deps      外部依赖检查（检查 API 连通性）"
    echo ""
}

### 检查命令行参数 ###
if [ $# -gt 0 ]; then
    case $1 in
        "all"|"search"|"llm"|"store"|"mail"|"accuracy"|"deps")
            # 命令行模式下，在测试运行前显示环境变量检查表
            echo ""
            echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo -e "${CYAN}                         📝 测试运行前配置检查${NC}"
            echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo ""
            print_env_requirements
            run_test $1
            exit $?
            ;;
        "-h"|"--help"|"help")
            show_usage
            exit 0
            ;;
        *)
            echo -e "${RED}错误：未知的选项 '$1'${NC}"
            show_usage
            exit 1
            ;;
    esac
fi

### 进入交互模式 ###
# 启动时显示初始环境变量检查表（表格形式）
echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}                         📝 环境变量配置检查${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 检查 Playwright 环境
PLAYWRIGHT_DIR="$ROOT_DIR/bin/playwright"

# 检查 Playwright 是否已安装且完整
is_playwright_ready() {
    local dir="$1"
    # 检查关键文件和目录是否存在且有实际内容
    if [ -d "$dir/node/bin" ] && [ -f "$dir/node/bin/node" ] && [ -s "$dir/node/bin/node" ] &&
       [ -f "$dir/package.json" ] && [ -s "$dir/package.json" ] &&
       [ -d "$dir/browsers" ] && [ "$(ls -A "$dir/browsers" 2>/dev/null | wc -l)" -gt 0 ] &&
       [ -d "$dir/node_modules" ] && [ "$(ls -A "$dir/node_modules" 2>/dev/null | wc -l)" -gt 0 ]; then
        return 0
    else
        return 1
    fi
}

if ! is_playwright_ready "$PLAYWRIGHT_DIR"; then
    if [ -d "$PLAYWRIGHT_DIR" ]; then
        echo -e "${YELLOW}⚠ Playwright 环境不完整或为空，删除后重新下载...${NC}"
        rm -rf "$PLAYWRIGHT_DIR"
    else
        echo -e "${YELLOW}⚠ Playwright 环境未找到，开始自动下载...${NC}"
    fi

    bash "$ROOT_DIR/scripts/setup-playwright.sh" "$PLAYWRIGHT_DIR"
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Playwright 下载完成${NC}"
    else
        echo -e "${RED}❌ Playwright 下载失败${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ Playwright 环境已就绪${NC}"
fi

echo ""
print_env_requirements

while true; do
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo "请选择要运行的测试模块："
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${YELLOW}🚀 快速测试${NC}"
    echo "  1) 搜索模块测试 (Search)          - 测试搜索查询构建、结果处理等"
    echo "  2) LLM 模块测试 (LLM)             - 测试 LLM 连接和对话功能"
    echo "  3) 存储模块测试 (Store)           - 测试数据库操作和持久化"
    echo "  4) 邮件模块测试 (Mail)            - 测试 SMTP 配置和邮件模板"
    echo ""
    echo -e "${YELLOW}🎯 完整测试${NC}"
    echo "  5) 搜索准确率测试 (Accuracy)      - 完整搜索准确率评估，测试30家公司"
    echo "  6) 外部依赖检查 (Dependency)      - 检查 SerpApi 和 LLM API 连通性"
    echo "  7) 全部测试 (All Tests)           - 运行项目所有测试套件"
    echo ""
    echo -e "${YELLOW}🎛️  其他${NC}"
    echo "  8) 重新显示环境变量配置表"
    echo "  0) 退出"
    echo ""
    echo -n "请输入选项 [0-8]: "
    read -r option
    echo ""

    case $option in
        1) run_test "search" ;;
        2) run_test "llm" ;;
        3) run_test "store" ;;
        4) run_test "mail" ;;
        5) run_test "accuracy" ;;
        6) run_test "deps" ;;
        7) run_test "all" ;;
        8)
            echo ""
            echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo -e "${BLUE}                       📋 环境变量配置检查表${NC}"
            echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo ""
            print_env_requirements
            ;;
        0)
            echo -e "${GREEN}感谢使用，再见！${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}无效的选项: ${option}${NC}"
            echo "请重新输入"
            ;;
    esac

done

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}                       🤖 AI 外贸助手 - 单元测试选择器${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

while true; do
    echo "请选择要运行的测试模块："
    echo ""
    echo "1) 搜索模块测试 (Search)"
    echo "2) LLM 模块测试 (LLM)"
    echo "3) 存储模块测试 (Store)"
    echo "4) 邮件模块测试 (Mail)"
    echo "5) 准确率测试 (Accuracy)"
    echo "6) 外部依赖检查 (Dependency Check)"
    echo "7) 全部测试 (All Tests)"
    echo "0) 退出"
    echo ""
    echo -n "请输入选项 [0-7]: "
    read -r option
    echo ""

    case $option in
        1)
            echo -e "${YELLOW}正在运行搜索模块测试...${NC}"
            cd "$BACKEND_DIR"
            go test -v ./services -run "TestSearch"
            ;;
        2)
            echo -e "${YELLOW}正在运行 LLM 模块测试...${NC}"
            cd "$BACKEND_DIR"
            go test -v ./services -run "TestLLM"
            ;;
        3)
            echo -e "${YELLOW}正在运行存储模块测试...${NC}"
            cd "$BACKEND_DIR"
            go test -v ./store
            ;;
        4)
            echo -e "${YELLOW}正在运行邮件模块测试...${NC}"
            cd "$BACKEND_DIR"
            go test -v ./services -run "TestMail"
            ;;
        5)
            echo -e "${YELLOW}正在运行搜索准确率测试...${NC}"
            cd "$BACKEND_DIR"
            go test -v ./services -run "TestExternalDependencies"
            go test -v ./services -run "TestSearchAccuracy"
            ;;
        6)
            echo -e "${YELLOW}正在运行外部依赖检查...${NC}"
            cd "$BACKEND_DIR"
            go test -v ./services -run "TestExternalDependencies"
            ;;
        7)
            echo -e "${YELLOW}正在运行所有测试...${NC}"
            cd "$BACKEND_DIR"
            go test -v ./...
            ;;
        0)
            echo -e "${GREEN}感谢使用，再见！${NC}"
            exit 0
            ;;
        *)
            echo -e "无效的选项: ${option}"
            echo "请重新输入"
            ;;
    esac

    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo ""
done
