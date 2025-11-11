## **B2B AI 外贸客户开发助手**


### **二、 阶段一：客户信息解析 (Step 1) - 优化**


1.  **搜索算法优化 (Query Expansion & Targeting)**:
    *   **关键词扩展**：除了拼写“联系人”，应扩展为更具体、价值更高的角色，如 `(owner OR founder OR CEO OR "purchasing manager" OR "sourcing director")`。
    *   **分层搜索**：第一轮进行泛搜确定官网；第二轮在确定官网后，使用 `site:{{customer.website}}` 语法进行站内深度搜索，查找“About Us”、“Team”、“Contact”等页面，信息更精准。
    *   **社交媒体补充**：增加一步可选的 LinkedIn 搜索，特别是针对关键联系人，以获取其职位和动态。

2.  **解析算法优化 (Smarter Crawling & Validation)**:
    *   **智能内容抓取**：`WebFetcher.Fetch` 不应只抓取前 2000 字。应优先解析 `<body>` 内的文本，并可以尝试基于 HTML 结构（如 `<h1>`, `<p>`, `<footer>`）或关键词（如 "About", "Services", "Products"）来提取更有价值的信息块。
    *   **信息交叉验证**：LLM 解析出的公司名、业务摘要，应与搜索结果中的多个 `snippet` 进行比对，增加一个置信度评分。如果 LLM 提取的官网内容与用户最初的查询关联度低，应在 `candidates` 中降低其 `rank` 并注明原因。

#### **优化后的提示词 (Prompt)**

- **System Prompt**:
  ```
  You are a senior B2B market intelligence analyst. Your mission is to synthesize web search results and website content into a structured company profile for a sales team. Prioritize accuracy and identify key decision-makers.
  ```
- **User Prompt 模板**:
  ```text
  My Goal: To build a profile for a potential B2B customer.
  Initial Query: {{query}}

  ### Search Engine Results:
  {{#each search_results}}
  - **{{this.title}}**
    URL: {{this.url}}
    Snippet: {{this.snippet}}
  {{/each}}

  ### Official Website Content Analysis:
  Crawled URL: {{page.url}}
  Key Text Sections:
  {{page.text}}
  Discovered Emails: {{emails}}

  ### Instructions:
  Based on all the material provided, please perform the following steps:
  1.  **Identify the most credible official website.**
  2.  **Determine the company's country of operation.**
  3.  **Extract key contact persons, especially those in leadership or procurement roles.**
  4.  **Write a concise summary focusing on their business model and target market.**
  5.  **Output a clean JSON object.**

  ### JSON Output Format:
  {
    "website": "The normalized official URL (https://...)",
    "website_confidence": 0.95, // Your confidence score from 0.0 to 1.0
    "country": "...",
    "contacts": [
      {
        "name": "...",
        "title": "...", // e.g., CEO, Purchasing Manager
        "email": "...",
        "is_key_decision_maker": true, // boolean
        "source": "e.g., Website 'About Us' page, Search Snippet 1"
      }
    ],
    "summary": "A 100-150 word summary in Chinese, focusing on their core business, scale, and primary customer base.",
    "candidates": [ // List of other potential websites with reasoning
      {
        "url": "...",
        "title": "...",
        "rank": 2,
        "reason": "e.g., Appears to be a regional distributor site, not corporate HQ."
      }
    ]
  }
  ```

---

### **三、 阶段二：客户评级 (Step 2) - 优化**

#### **当前设计的优点**
- 逻辑简单，依赖用户定义的标准。

#### **优化建议**

1.  **算法优化 (Few-Shot Learning & Structured Reasoning)**:
    *   **引入范例**：在提示词中提供 1-2 个“黄金标准”的评级案例（一个 A 级，一个 C 级）。这能极大提升 LLM 对评级标准理解的准确性，即 Few-Shot Learning。
    *   **结构化推理**：要求 LLM 不仅给出结果，还要列出支持该评级的“积极信号”和“消极信号”。这使得 `reason` 字段不再是模糊的描述，而是可供分析的结构化数据。

#### **优化后的提示词 (Prompt)**

- **System Prompt**:
  ```
  You are a B2B sales strategist specializing in customer segmentation. Your task is to provide a precise customer rating (A, B, or C) based on the user's guideline. You must justify your rating by listing clear positive and negative signals.
  ```
- **User Prompt 模板**:
  ```text
  ### Customer Profile:
  - Name: {{customer.name}}
  - Website: {{customer.website}}
  - Country: {{customer.country}}
  - Summary: {{customer.summary}}

  ### Rating Guideline:
  {{settings.rating_guideline}}

  ### Rating Examples (for calibration):
  - **Example of an 'A' Grade Customer**: A large manufacturer in a target industry with over 200 employees and clear demand for our type of product.
  - **Example of a 'C' Grade Customer**: A small trading company with a generic website, unclear business focus, and located in a high-risk region.

  ### Instructions:
  Analyze the customer profile against the guideline and examples. Output a JSON object with your suggested grade and a structured reasoning.

  ### JSON Output Format:
  {
    "suggested_grade": "A|B|C",
    "confidence_score": 0.9, // Your confidence in this rating
    "reasoning": {
      "positive_signals": [ // List of points supporting the grade
        "e.g., Company operates in a key target industry.",
        "e.g., Website showcases high-value products relevant to our offerings."
      ],
      "negative_signals": [ // List of points against a higher grade
        "e.g., Company size appears to be small.",
        "e.g., No key contact information was found."
      ]
    }
  }
  ```

---

### **四、 阶段三：产品切入分析 (Step 3) - 优化**

#### **当前设计的优点**
- 明确结合了客户信息和我方产品。

#### **优化建议**

1.  **分析算法优化 (Deeper & More Actionable Insights)**:
    *   **从“痛点”到“需求与机遇”**：“痛点”一词有时过于笼统。优化为“客户的潜在需求与增长机遇”，引导 LLM 从更积极和商业的角度思考。
    *   **解决方案映射 (Solution Mapping)**：要求 LLM 不仅是给出“切入点”，而是将“我方产品的具体特性”与“客户的具体需求”进行一对一或一对多的映射。这是最有价值的优化，直接为销售邮件提供了核心论据。
    *   **增加风险识别**：增加一个“潜在顾虑与应对策略”字段，提前思考客户可能会提出的反对意见（如价格、集成复杂性），并构思应对策略。

#### **优化后的提示词 (Prompt)**

- **System Prompt**:
  ```
  You are a top-tier B2B product marketing consultant. Your expertise is in mapping our product's strengths to a prospective customer's specific business needs. Generate a deep and actionable analysis.
  ```
- **User Prompt 模板**:
  ```text
  ### Customer Profile:
  - Name: {{customer.name}}
  - Website: {{customer.website}}
  - Country: {{customer.country}}
  - Summary: {{customer.summary}}

  ### My Product/Service Profile:
  - **Product Name**: {{settings.my_product_name}}
  - **Key Features**:
    {{#each settings.my_product_features}}
    - {{this}}
    {{/each}}
  - **Unique Selling Proposition (USP)**: {{settings.my_product_usp}}

  ### Instructions:
  Conduct a strategic analysis to identify the best entry points. Think step-by-step:
  1.  What are the customer's primary business goals and operational needs based on their profile?
  2.  How can our specific product features help them achieve these goals or solve their problems?
  3.  What are the most compelling arguments we can make?
  4.  What objections might they have?

  ### JSON Output Format (in professional Chinese):
  {
    "customer_needs_assessment": "深入分析客户的核心业务模式，并推断其当前最关心的 2-3 个业务目标或运营挑战。",
    "solution_mapping": [
      {
        "customer_need": "客户的一个具体需求或挑战",
        "our_solution_feature": "我方能满足此需求的具体产品特性",
        "value_proposition": "这个解决方案能为客户带来的核心价值（如：降本15%、效率提升20%、打开新市场）"
      }
    ],
    "potential_objections_and_responses": [
      {
        "objection": "客户可能提出的顾虑（例如：价格太高、部署复杂）",
        "response_strategy": "应对该顾虑的初步沟通策略"
      }
    ],
    "executive_summary": "一段 200 字左右的中文综述，整合以上分析，作为内部沟通的核心摘要。"
  }
  ```

---

### **五、 阶段四：个性化开发信与跟进 (Step 4) - 优化**

#### **当前设计的优点**
- 流程清晰，区分了初次邮件和跟进邮件。

#### **优化建议**

1.  **邮件结构与算法优化 (AIDA Model & Multi-option Generation)**:
    *   **引入经典营销模型**：明确要求 LLM 遵循 **AIDA** (Attention, Interest, Desire, Action) 或 **PAS** (Problem, Agitate, Solution) 等成熟的邮件撰写框架，使邮件逻辑更具说服力。
    *   **提供备选项**：不要只生成一个版本。要求 LLM 提供 2-3 个不同风格的邮件标题（如一个直接型，一个悬念型）和不同的 CTA (Call-to-Action) 选项（一个硬性 CTA 如“预约会议”，一个软性 CTA 如“是否有兴趣看一份案例”）。
    *   **跟进策略序列化**：为自动跟进建立一个策略序列。第一次跟进可以“提供价值”（如发送一份行业白皮书），第二次可以是“展示案例”，第三次是“最后尝试/破冰”。在调用 LLM 时传入跟进的类型（`followup_type`），使其生成高度相关的邮件内容。

#### **优化后的提示词 (Prompt)**

##### **初次开发信 (Initial Email)**

- **System Prompt**:
  ```
  You are an expert B2B copywriter specializing in high-conversion cold emails. Write in a concise, professional, and native English tone. Your goal is to secure a response. Use the AIDA (Attention, Interest, Desire, Action) model.
  ```
- **User Prompt 模板**:
  ```text
  ### Context for the Email:
  - **Customer Name**: {{customer.name}}
  - **My Company Name**: {{settings.my_company_name}}
  - **Target Contact**: {{contactLine}} // "John Doe" or "Hi there"
  - **Strategic Analysis (Key Arguments to Use)**:
    {{#each analysis.solution_mapping}}
    - **Customer Need**: {{this.customer_need}}
      **Our Solution**: Our '{{this.our_solution_feature}}' can help you achieve '{{this.value_proposition}}'.
    {{/each}}

  ### Instructions:
  Draft a compelling cold email based on the context. The body should be between 120-180 words.
  1.  **Attention**: Start with a personalized hook related to their company.
  2.  **Interest & Desire**: Clearly connect their potential need with our solution's value proposition.
  3.  **Action**: End with a clear, low-friction call-to-action.

  ### JSON Output Format:
  {
    "subject_options": [
      "A question about {{customer.name}}'s operations",
      "Idea for improving {{a specific customer pain point}}",
      "Partnership Opportunity with {{settings.my_company_name}}"
    ],
    "body": "A 120-180 word email body, written in English. Ensure it's personalized and value-driven.",
    "cta_options": [
      "Are you available for a brief 15-minute call next week to explore this?",
      "Would you be open to seeing a one-page case study on how we helped a similar company?",
      "Is this a priority for you at the moment?"
    ]
  }
  ```

##### **自动跟进信 (Follow-up Email)**

- **System Prompt**:
  ```
  You are a professional Customer Success Manager. Your task is to write a brief, friendly, and value-added follow-up email in English. Avoid sounding desperate or pushy.
  ```
- **User Prompt 模板**:
  ```text
  ### Context:
  - **Previous Email Body**:
    """
    {{context_email.body}}
    """
  - **My Company Name**: {{settings.my_company_name}}
  - **Follow-up Type**: {{followup_type}} // e.g., "Value Add", "Case Study", "Break Up"

  ### Instructions:
  Based on the previous email and the follow-up type, write a concise follow-up email (under 100 words).
  - If type is "Value Add", offer a useful resource (e.g., an industry report link placeholder).
  - If type is "Case Study", briefly mention a success story (use placeholders).
  - If type is "Break Up", politely close the loop and leave the door open for the future.

  ### JSON Output Format:
  {
    "subject": "e.g., Re: Previous email | A quick follow-up",
    "body": "The follow-up email body, written in English."
  }
  ```

---

### **六、 自动化流程串联优化**

- **动态跟进调度**：`Scheduler.Schedule` 不应只依赖固定的 `AutomationFollowupDays`。可以根据客户评级动态调整：
    - **A 级客户**：跟进周期缩短（如 2, 4, 7 天）。
    - **B 级客户**：标准周期（如 3, 7, 14 天）。
- **引入“冷却”机制**：如果一个自动化序列（如 3 封邮件）完成后客户仍未回复，应自动将该客户标记为“冷却”，在 3-6 个月内不再自动触达，避免骚扰。
- **失败与重试逻辑**：在任何 LLM 调用失败或返回不合规 JSON 时，应有自动重试机制（最多 2 次），并在重试时可以略微调整提示词（例如增加 "You must output a valid JSON object." 的指令）。若持续失败，则将任务移入“待人工处理”队列。

通过以上优化，整个系统将变得更加智能、精准和强大，能够显著提升 B2B 客户开发流程的效率和效果。