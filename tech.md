以下为“AI 外贸客户开发助手”的最小化可用技术文档（MVP），覆盖从客户识别到自动跟进的全流程。后端采用 Go，前端采用 Vue，数据仅存储于本地 SQLite，运行形态为本地 exe 启动轻量级 Web 服务，严格按 ui 文件夹提供的样式实现界面。

一、总体概览与运行形态
- 目标：实现客户识别 -> 智能评级 -> 深度分析 -> 个性化触达 -> 自动化跟进的单页线性流程，并确保本地私有化、可离线（除使用外部 API 时）。
- 运行：用户双击 exe 启动内置 HTTP 服务器（默认 127.0.0.1:7860），自动打开默认浏览器。
- 数据：所有数据与配置均保存在用户目录下 ./foreign_trade 目录。
- 智能：通过用户配置的 LLM API（可兼容 OpenAI 格式）完成搜索解析、评估与内容生成。
- 邮件：通过用户配置 SMTP 发送自动跟进邮件。
- 搜索：调用用户配置的搜索引擎 API（建议 Bing/SerpAPI/DDG API）。若未配置，提供“仅网址直连模式”。

二、技术栈与关键依赖
- 后端（Go 1.22+）
  - Web 框架：net/http + chi 或 gorilla/mux（二选一，MVP 推荐 chi）
  - 静态资源：Go embed
  - 数据库：SQLite + mattn/go-sqlite3
  - HTML 解析：PuerkitoBio/goquery
  - 抓取：net/http（可选 colly，MVP 先用 http+goquery）
  - 任务调度：robfig/cron v3（或简化为内循环轮询）
  - SMTP：gomail（gopkg.in/gomail.v2）
  - JSON：encoding/json
- 前端（Vue 3 + Vite）
  - 状态管理：Pinia
  - 路由：Vue Router（两页：主流程、全局配置）
  - UI：严格复刻 ui 文件夹样式与组件规范（仅用原生 CSS/SCSS，避免引入重 UI 库）
- 构建与发布
  - 前端打包到 dist，嵌入后端二进制
  - Windows 打包：go build -ldflags "-H=windowsgui" 生成无控制台窗口 exe

三、目录结构
- 根目录
  - /backend
    - main.go（启动、路由、嵌入静态资源、首次运行初始化）
    - api（REST 接口）
    - services（LLM、搜索、抓取、评估、邮件、调度）
    - store（SQLite 连接与DAO）
    - domain（实体与DTO）
    - prompts（模板）
    - static（内置兜底静态资源，如 favicon）
  - /frontend
    - src（Vue 代码：pages、components、stores、assets）
    - ui（提供的样式资源与切图，保持原命名）
    - vite.config.ts
  - /scripts（打包、发布辅助）
- 运行时（首次启动自动创建）
  - ~/.foreign_trade（首次运行会从旧目录 ~/foreign_trade 迁移，如存在）
    - app.db（SQLite）
    - config.json（全局配置）
    - logs/app.log（按天切分，历史日志自动 gzip 压缩）
    - cache/（搜索结果与网页快取）
    - exports/（导出报告或邮件备份）

五、日志
- 位置：默认写入用户目录下 `~/.foreign_trade/logs`，当前文件 `app.log`，每日生成 `app.YYYY-MM-DD.log` 并在切分后自动 gzip 压缩历史日志；同时输出到终端（stdout）。
- 等级：默认 Info；可通过环境变量 `LOG_LEVEL` 调整（支持 `debug`/`info`/`warn`/`error`/`off`）。
- 后端内部所有 stdlib `log.Printf` 输出会写入文件；同时提供结构化 `slog` 日志（见 `backend/logging`）。

四、数据库模型（SQLite）
通用字段：id INTEGER PK AUTOINCREMENT；created_at TEXT；updated_at TEXT（ISO8601）

1) settings（仅一行）
- llm_base_url TEXT
- llm_api_key TEXT
- llm_model TEXT
- my_company_name TEXT
- my_product_profile TEXT
- smtp_host TEXT
- smtp_port INTEGER
- smtp_username TEXT
- smtp_password TEXT
- admin_email TEXT
- rating_guideline TEXT
- search_provider TEXT（如 bing、serpapi、ddg）
- search_api_key TEXT

2) customers
- name TEXT
- website TEXT
- country TEXT
- grade TEXT（A/B/C/unknown）
- grade_reason TEXT
- source_json TEXT（搜索与解析原始数据，用于追溯）

3) contacts
- customer_id INTEGER
- name TEXT
- title TEXT
- email TEXT
- phone TEXT
- source TEXT（抓取来源页面）
- is_key BOOLEAN

4) analyses（A 级客户的切入点报告）
- customer_id INTEGER
- core_business TEXT
- pain_points TEXT
- my_entry_points TEXT
- full_report TEXT

5) emails（邮件草稿与发送记录）
- customer_id INTEGER
- type TEXT（initial/followup）
- subject TEXT
- body TEXT
- status TEXT（draft/sent/failed）
- sent_at TEXT
- smtp_message_id TEXT

6) followups（首次保存的跟进记录）
- customer_id INTEGER
- initial_email_id INTEGER
- notes TEXT

7) scheduled_tasks（自动跟进任务）
- customer_id INTEGER
- due_at TEXT（计划触发时间）
- status TEXT（scheduled/running/sent/failed/cancelled）
- last_error TEXT
- context_email_id INTEGER（引用首次邮件）
- generated_email_id INTEGER（生成的跟进邮件ID）

8) logs（可选，或仅文件日志）
- level TEXT
- message TEXT
- meta_json TEXT

五、后端 API 设计（REST）
约定：所有接口返回 {ok: bool, data: any, error: string}

全局配置
- GET /api/settings 获取配置
- PUT /api/settings 保存配置
- POST /api/settings/test-llm 测试 LLM 连通性
- POST /api/settings/test-smtp 发送测试邮件到 admin_email
- POST /api/settings/test-search 测试搜索 API 可用性

主流程 Step 1：智能信息获取与聚合
- POST /api/companies/resolve
  - 入参：{query: string}（公司全名或官网地址）
  - 出参：{
      website: string, country: string, contacts: [{email,name,title,source,is_key}],
      candidates: [{url,title,rank,reason}], // 供前端展示可编辑
      summary: string // LLM 汇总描述
    }
- POST /api/companies 保存用户校准后的信息，返回 customer_id

Step 2：AI 辅助客户价值评级
- POST /api/companies/{id}/grade/suggest
  - 入参：空（服务端读取 settings.rating_guideline 与公司官网内容）
  - 出参：{suggested_grade: 'A'|'B'|'C', reason: string}
- POST /api/companies/{id}/grade/confirm
  - 入参：{grade: 'A'|'B'|'C', reason?: string}
  - 出参：保存后对象
  - 若 B/C：直接归档，流程结束；若 A：进入 Step 3

Step 3：生成产品切入点分析（仅 A 级）
- POST /api/companies/{id}/analysis
  - 出参：{
      core_business:string, pain_points:string, my_entry_points:string, full_report:string
    }

Step 4：生成个性化开发信并保存记录
- POST /api/companies/{id}/email-draft
  - 出参：{subject:string, body:string, email_id:number(status=draft)}
- POST /api/companies/{id}/followup/first-save
  - 入参：{email_id:number}（将草稿标记为首次跟进记录，不发送）
  - 出参：{followup_id:number}

Step 5：设置自动化邮件跟进
- POST /api/followups/schedule
  - 入参：{customer_id:number, context_email_id:number, delay_days:number}
  - 出参：{task_id:number, due_at:string}
- GET /api/scheduled-tasks?status=scheduled
- POST /api/scheduled-tasks/{id}/run-now（调试用）
- 任务到期时后台自动：
  - 读取 context_email_id 对应的首次邮件，调用 LLM 生成简短且提供新价值的跟进邮件；
  - 调用 SMTP 发送，写入 emails 并更新 scheduled_tasks 状态。

六、核心服务与最小实现策略
1) 搜索与官网识别（services/search, services/scrape）
- 搜索流程（可配置 search_provider）：
  - 若 query 为 URL：直接进入抓取。
  - 否则调用搜索 API，取前 10 个结果；过滤 linkedin, facebook, crunchbase, 1688, alibaba, job boards 等域名；保留独立域候选。
  - 评分规则（MVP）：域名与公司名相似度、是否为品牌页、是否有 About/Contact、是否存在 schema.org Organization。
  - 选择最高分作为官网，并记录候选列表。
- 抓取流程：
  - 抓取首页、/about、/contact（并跟随站内 1 层深度限制）
  - 提取：
    - 邮箱：正则 mailto 与文本匹配
    - 职位/人名：基于常见头衔词（CEO, Founder, Purchasing, Buyer, Sourcing, Sales）邻近邮箱的行文本
    - 国家：优先读取 schema.org PostalAddress；其次页面 footer 地址；再次 TLD 辅助；最后交给 LLM 从页面文本推断
  - 将抓取页面摘要提供给 LLM，生成 summary 与联系人关键信息的补全与去噪。
- 失败兜底：若无搜索 API，且非 URL，提示用户手动输入官网；仍允许继续流程。

2) LLM 适配（services/llm）
- 通用 ChatCompletion 适配（OpenAI 兼容接口：/v1/chat/completions）
- 统一超时与重试；提示词模板化（prompts/）
- 需要的最小提示模板：
  - 官网判定与信息聚合（输入：搜索结果片段与抓取片段；输出：官网、国家、联系人、summary）
  - 评级建议（输入：官网文本摘要与 rating_guideline；输出：A/B/C 与理由）
  - 切入点分析（输入：客户信息、官网摘要、my_product_profile；输出：核心业务/痛点/切入点）
  - 开发信初稿（输入：切入点分析与我的公司信息；输出：标题与正文）
  - 自动跟进信（输入：首次邮件正文与近期新增价值点的要求；输出：简短、友好、附加价值的正文）

3) 邮件服务（services/mail）
- 使用 TLS 连接 SMTP；From 使用 smtp_username；To 用户手动维护的联系人邮箱（首次记录保存不发送）
- 发送成功写入 emails 表并返回 message-id；失败记录 last_error

4) 调度与后台任务（services/scheduler）
- 程序启动时加载定时器：每分钟扫描 scheduled_tasks where status='scheduled' and due_at<=now
- 对到期任务：
  - 标记 running
  - 生成跟进邮件草稿 -> 调用 SMTP 发送 -> 更新任务为 sent 并写入 generated_email_id
  - 错误即 status=failed 并写 last_error
- 注意：应用关闭则不触发。MVP 中在设置页提示“需保持程序运行以发送自动邮件”，未来可接入系统计划任务。

七、前端信息架构与交互（Vue 3 单页）
路由
- / 主流程页（单页面线性步骤）
- /settings 全局配置页

状态管理（Pinia）
- settingsStore：LLM、SMTP、搜索配置、加载状态
- flowStore：当前客户对象、步骤状态与结果、当前草稿与任务
- uiStore：全局提示、错误、自动滚动

主流程页组件
- Header：标题“AI 外贸客户开发助手”、右上角设置按钮
- Step1Card：
  - 输入框“输入客户公司全名或官网地址”
  - “开始分析”按钮 -> 调用 /api/companies/resolve
  - 展示：官网、国家、联系人（可编辑）、候选域名列表与置信度
  - “保存并继续” -> POST /api/companies 返回 id
- Step2Rating：
  - 自动触发“AI 正在进行价值评估…”
  - 展示建议等级与理由
  - 三个按钮：确认 A（继续）、调整 B（归档）、调整 C（忽略）
- Step3Analysis（仅 A）：
  - “正在生成切入点分析…”
  - 展示核心业务/痛点/我方切入点（可编辑）
  - “保存并继续”
- Step4Email：
  - “生成开发信” -> 展示标题与正文（可编辑）
  - “保存为首次跟进记录” -> 固化草稿为 initial 邮件与 followup 记录
- Step5Schedule：
  - 快捷按钮：3/7/14 天
  - 点击后提示“已设置任务：X 天后将自动发送跟进邮件”
  - 按钮：“完成并开始下一个” -> 重置 flowStore

全局配置页组件
- LLM 设置：base_url、api_key、model、[测试连接]
- 我的信息：公司名、产品/服务简介
- 邮件设置：SMTP、端口、账号、密码/授权码、[发送测试邮件]
- 系统设置：管理员邮箱
- 客户评级标准：自然语言规则文本域
- 搜索 API（可选）：提供商、API Key、[测试搜索]
- 保存按钮：保存后给出成功提示

样式实现
- 完全按 ui 文件夹样式表与切图，保证字体、颜色、间距与交互动效一致
- 响应式要求：桌面优先，≥1280 宽度最佳显示

八、首次启动与初始化
- 若不存在 ./foreign_trade：
  - 创建目录、config.json（默认空配置）、app.db（建表）、logs 目录
- 监听 127.0.0.1:7860，嵌入静态资源路由到前端 index.html
- 自动打开默认浏览器到主页面
- 前端检测配置完整性（LLM、SMTP、评级标准若缺失则红点提示）

九、最小提示词模板（要点级，MVP 可直接内嵌）
1) 官网与信息聚合
- 角色：专业 B2B 研究助理
- 输入：搜索 TopN 摘要、抓取到的页面片段、已过滤平台域名清单
- 任务：确定官网 URL、推断国家（给出依据信息）、列出潜在联系人与邮箱、生成 80 字摘要
- 输出 JSON：{website,country,contacts:[{name,title,email,is_key}],summary}

2) 评级
- 输入：settings.rating_guideline + 官网摘要 + 行业关键词
- 输出：{grade:'A'|'B'|'C',reason}

3) 切入点分析
- 输入：my_product_profile + 客户官网摘要与联系人信息
- 输出：{core_business,pain_points,my_entry_points,full_report}

4) 开发信
- 输入：公司名、联系人（若无则通用称呼）、切入点
- 约束：主题 1 行；正文 150-220 词、明确痛点与解决点、含软性 CTA
- 输出：{subject,body}

5) 自动跟进
- 输入：首次邮件正文、要求提供新价值（如案例/参数/白皮书链接占位符）
- 输出：简短友好、≤120 词、单一行动点

十、安全与隐私（MVP 约束）
- 数据与日志仅本地存储，无遥测
- 配置中的 key 明文保存在 config.json（MVP 简化）。若需加密：
  - Windows 可用 DPAPI；跨平台可用 AES-256（密钥保存在本机受限路径）。非本次 MVP 必需
- LLM 调用与搜索 API 超时与重试，避免泄露超出必要上下文

十一、错误处理与可观测性
- 统一错误响应结构，前端 toast 显示
- logs/app.log 记录：INFO/ERROR，含任务 id、公司 id
- 任务失败自动重试策略：最多 3 次，指数退避；仍失败标记 failed 并在设置页提示

十二、构建与发布流程
- 前端
  - 安装：cd frontend && npm ci
  - 构建：npm run build（产物位于 frontend/dist）
- 后端
  - 将 dist 通过 //go:embed 嵌入
  - 构建：cd backend && go build -ldflags "-H=windowsgui" -o AI_Trade_Assistant.exe
- 分发：仅分发 exe；用户双击运行

十三、验收用例（MVP）
- 配置页
  - 成功保存 LLM/SMTP/评级标准；测试 LLM 返回模型名；测试 SMTP 发到管理员邮箱成功
- Step 1
  - 输入公司名，能返回官网、国家与 1-3 个邮箱；可手动修正后保存
- Step 2
  - 自动给出 A/B/C 建议及理由；确认 B/C 后归档并结束；确认 A 自动进入 Step 3
- Step 3
  - 返回三段要点与完整报告；可编辑保存
- Step 4
  - 一键生成开发信；保存为首次跟进记录（不发送）；数据库有 emails draft 与 followups 记录
- Step 5
  - 选择 7 天后；任务入库；手动触发 run-now 可发送一封跟进邮件；发送成功写入 emails 并更新任务为 sent


单元测试

使用 config 下的配置文件信息完成单元测试, 要求LLM 都是真实调用的, 搜索 API 也需要真实调用; 单元测试禁止mock 数据
