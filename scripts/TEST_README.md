# AI 外贸助手 - 单元测试使用说明

## 环境变量说明

在运行测试之前，需要配置以下环境变量。所有测试模块都会从环境变量读取配置，而不是从 config 文件读取。

### 必须配置的环境变量

#### 1. 搜索模块配置
测试搜索功能时需要配置以下任一变量：

```bash
export SerpApi="your_serpapi_key_here"
# 或
export SERPAPI_API_KEY="your_serpapi_key_here"
```

**如何获取：** 前往 https://serpapi.com/ 注册并获取 API Key

#### 2. LLM 大模型配置
测试 LLM 功能时必填：

```bash
export LLM_API_KEY="your_llm_api_key_here"
export LLM_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"
export LLM_MODEL_NAME="deepseek-v3-250324"
```

**说明：** 本配置适用于火山引擎的 DeepSeek API，也可用于其他 OpenAI 兼容的 API

### 可选配置的环境变量

#### 3. SMTP 邮件配置
测试邮件功能时需要：

```bash
export SMTP_HOST="smtp.example.com"
export SMTP_PORT="465"
export SMTP_USERNAME="your_email@example.com"
export SMTP_PASSWORD="your_smtp_password"
export ADMIN_EMAIL="admin@example.com"
```

#### 4. 公司信息配置
影响邮件模板和其他业务功能：

```bash
export COMPANY_NAME="Your Company Name"
export COMPANY_PRODUCT="Your product description"
```

## 快速配置方法

### 方法一：直接导出到环境

在终端中直接执行：

```bash
export SerpApi="your_key_here"
export LLM_API_KEY="your_llm_key"
export LLM_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"
export LLM_MODEL_NAME="deepseek-v3-250324"

# 然后运行测试
```

### 方法二：使用 .env 文件

1. 复制示例文件：
```bash
cp scripts/.env.example scripts/.env
```

2. 编辑 scripts/.env 文件，填入真实值

3. 加载环境变量并运行测试：
```bash
# 加载 .env 文件
source scripts/.env

# 或在运行测试时加载（如果支持）
```

## 运行测试

### 使用脚本运行测试

推荐使用提供的交互式脚本：

```bash
cd /Users/anner/code/AI-Foreign-Trade-Assistant/backend
bash ../scripts/run_tests.sh
```

脚本功能：
- 自动检查环境变量配置状态
- 显示 ✅ 或 ❌ 标注配置是否完整
- 提供菜单式选择，支持运行不同模块的测试
- 必须先配置环境变量，否则提示重新配置

### 脚本菜单选项

```
1) 搜索模块测试 (Search)
2) LLM 模块测试 (LLM)
3) 存储模块测试 (Store)
4) 邮件模块测试 (Mail)
5) 准确率测试 (Accuracy)           - 需要 SerpApi 和 LLM
6) 外部依赖检查 (Dependency Check) - 检查 API 连通性
7) 全部测试 (All Tests)
0) 退出
```

### 手动运行测试

如果不想使用脚本，也可以手动运行：

```bash
cd /Users/anner/code/AI-Foreign-Trade-Assistant/backend

# 运行特定模块的测试
go test -v ./services -run TestSearchAccuracy     # 搜索准确率测试
go test -v ./services -run TestExternalDependencies # 外部依赖检查
go test -v ./services -run TestSearch              # 搜索模块测试
go test -v ./services -run TestLLM                 # LLM 模块测试
go test -v ./store                                 # 存储模块测试
go test -v ./services -run TestMail                # 邮件模块测试

# 运行所有测试
go test -v ./...
```

## 测试执行流程

### 1. 外部依赖检查

当选择选项 5 或 6 时，会自动先执行 `TestExternalDependencies`：

- 检查 SerpApi 配置是否存在
- 检查 LLM 配置是否完整
- 测试 LLM API 连通性（发送测试请求）
- 测试搜索 API 连通性（搜索 "Apple Inc."）

如果不通过，准确率测试不会开始。

### 2. 搜索准确率测试

测试 30 家知名公司的官网搜索准确性：

- Apple Inc., Tesla, Nike, Coca-Cola, Nestlé, P&G, L'Oréal 等
- 验证搜索结果是否包含正确的官方网站
- 计算并输出准确率统计

### 3. 其他模块测试

- **搜索模块**：测试查询构建、结果去重、提供商回退等
- **LLM 模块**：测试配置验证、连接测试等
- **存储模块**：测试设置持久化、数据库操作等
- **邮件模块**：测试 SMTP 配置、邮件模板生成等

## 测试结果输出

### 成功示例

```
==========================================
环境变量配置说明
==========================================

✓ 搜索模块配置 (OK)
✓ LLM 大模型配置 (OK)
⚠ SMTP 邮件配置 (可选)

测试执行结果示例：
--- PASS: TestSearchAccuracy (125.67s)
    search_test.go:171: ========================================
    search_test.go:172: Search Accuracy Test Summary:
    search_test.go:173: Total companies tested: 30
    search_test.go:174: Correct results: 28
    search_test.go:175: Accuracy: 93.33%
    search_test.go:176: ========================================
```

### 配置缺失示例

```
==========================================
环境变量配置说明
==========================================

❌ 搜索模块配置 (必須)
   SerpApi 或 SERPAPI_API_KEY: 用于搜索功能测试

❌ LLM 大模型配置 (必須)
   LLM_API_KEY:    LLM API Key
   LLM_BASE_URL:   API 基础地址
   LLM_MODEL_NAME: 模型名称

⚠ SMTP 邮件配置 (可选)

需要配置完成后才能继续测试。
```

## 常见问题

### Q1: 如何快速配置所有环境变量？

A: 创建 .env 文件并一次性加载：

```bash
cat > scripts/.env << 'EOF'
# 搜索配置
export SerpApi="your_key_here"

# LLM 配置
export LLM_API_KEY="your_key_here"
export LLM_BASE_URL="https://ark.cn-beijing.volces.com/api/v3"
export LLM_MODEL_NAME="deepseek-v3-250324"
EOF

# 加载配置
source scripts/.env
```

### Q2: 没有 SerpApi Key 可以测试吗？

A: 可以，但搜索准确率和外部依赖检查会被跳过。其他模块的测试（如存储、查询构建等）仍可正常运行。

### Q3: 测试会消耗 API 额度吗？

A: 是的：
- 搜索准确率测试：约 30 次搜索请求
- LLM 连通性测试：1 次对话请求
- 其他测试使用内存数据库，不消耗外部 API

### Q4: 如何在 CI/CD 中运行测试？

A: 在 CI/CD 环境中配置环境变量：

```bash
# GitHub Actions 示例
env:
  SerpApi: ${{ secrets.SERPAPI_KEY }}
  LLM_API_KEY: ${{ secrets.LLM_API_KEY }}
  LLM_BASE_URL: ${{ secrets.LLM_BASE_URL }}
  LLM_MODEL_NAME: ${{ secrets.LLM_MODEL_NAME }}

# 然后运行
go test -v ./services -run "TestExternalDependencies"
go test -v ./services -run "TestSearchAccuracy"
```

### Q5: 为什么测试都从环境变量读取？

A: 为了：
- 不依赖外部配置文件
- 避免将敏感信息提交到版本控制
- 支持在不同的环境（开发、测试、生产）中使用不同的配置
- 便于在 CI/CD 中集成

## 相关文件

- 测试文件目录：`/Users/anner/code/AI-Foreign-Trade-Assistant/backend/services/`
- 测试脚本：`/Users/anner/code/AI-Foreign-Trade-Assistant/scripts/run_tests.sh`
- 环境变量示例：`/Users/anner/code/AI-Foreign-Trade-Assistant/scripts/.env.example`
- 本说明文件：`/Users/anner/code/AI-Foreign-Trade-Assistant/scripts/TEST_README.md`
