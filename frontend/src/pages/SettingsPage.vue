<template>
  <div class="settings-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="avatar"></div>
        <div>
          <p class="brand-title">AI外贸助手</p>
          <p class="brand-sub">专业版</p>
        </div>
      </div>
      <nav class="menu">
        <button class="menu-item" type="button" @click="goHome">
          <span class="material">search</span>
          客户开发
        </button>
        <button class="menu-item">
          <span class="material">group</span>
          客户管理
        </button>
        <button class="menu-item">
          <span class="material">mail</span>
          邮件营销
        </button>
        <button class="menu-item">
          <span class="material">bar_chart</span>
          数据分析
        </button>
        <button class="menu-item menu-item--active">
          <span class="material">settings</span>
          全局配置
        </button>
      </nav>
    </aside>

    <main class="main">
      <header class="page-head">
        <div>
          <h1>全局配置</h1>
          <p>配置软件核心参数以确保 AI 助手正常运行。</p>
        </div>
        <button class="ghost" type="button" @click="goBack">
          <span class="material">arrow_back</span>
          返回
        </button>
      </header>

      <form class="form" @submit.prevent="handleSave">
        <section class="card">
          <header>
            <div>
              <h2>大模型 (LLM) 设置</h2>
              <p>配置 API 地址与密钥，用于智能分析与内容生成。</p>
            </div>
            <button type="button" class="chip" @click="handleTestLLM">
              <span class="material">bolt</span>
              测试连接
            </button>
          </header>
          <div class="grid">
            <label>
              <span>Base URL</span>
              <input v-model="local.llm_base_url" type="text" placeholder="https://api.example.com/v1" />
            </label>
            <label>
              <span>API Key</span>
              <input v-model="local.llm_api_key" type="text" placeholder="sk-..." />
            </label>
            <label>
              <span>模型名称</span>
              <input v-model="local.llm_model" type="text" placeholder="gpt-4o" />
            </label>
          </div>
        </section>

        <section class="card">
          <header>
            <div>
              <h2>我的信息</h2>
              <p>这些信息将用于生成开发信签名与公司介绍。</p>
            </div>
          </header>
          <div class="grid">
            <label>
              <span>我的公司名称</span>
              <input v-model="local.my_company_name" type="text" placeholder="例如：环球贸易有限公司" />
            </label>
          </div>
          <label>
            <span>我的产品 / 服务简介</span>
            <textarea
              v-model="local.my_product_profile"
              rows="4"
              placeholder="详细描述您的核心产品、优势和主要目标市场。"
            ></textarea>
          </label>
        </section>

        <section class="card">
          <header>
            <div>
              <h2>邮件发送 (SMTP) 设置</h2>
              <p>配置用于自动发送跟进邮件的 SMTP 服务器。</p>
            </div>
            <button type="button" class="chip" @click="handleTestSMTP">
              <span class="material">send</span>
              发送测试邮件
            </button>
          </header>
          <div class="grid">
            <label>
              <span>SMTP 服务器</span>
              <input v-model="local.smtp_host" type="text" placeholder="smtp.example.com" />
            </label>
            <label>
              <span>端口</span>
              <input v-model.number="local.smtp_port" type="number" placeholder="465" />
            </label>
            <label>
              <span>邮箱账号</span>
              <input v-model="local.smtp_username" type="text" placeholder="your@email.com" />
            </label>
            <label>
              <span>邮箱密码 / 授权码</span>
              <input v-model="local.smtp_password" type="password" placeholder="授权码" />
            </label>
          </div>
        </section>

        <section class="grid-two">
          <div class="card">
            <header>
              <div>
                <h2>系统设置</h2>
                <p>全局通知与默认管理员设置。</p>
              </div>
            </header>
            <label>
              <span>管理员邮箱</span>
              <input v-model="local.admin_email" type="email" placeholder="用于接收系统通知" />
            </label>
          </div>

          <div class="card card--stretch">
            <header>
              <div>
                <h2>客户评级标准</h2>
                <p>以自然语言描述您的 A/B/C 客户定义。</p>
              </div>
            </header>
            <textarea
              v-model="local.rating_guideline"
              rows="8"
              placeholder="例如：
A级：明确表达采购意向，有具体需求和预算；
B级：对产品感兴趣，正在评估供应商；
C级：暂无明确需求，仅收集资料。"
            ></textarea>
          </div>
        </section>

        <section class="card">
          <header>
            <div>
              <h2>搜索 API 设置</h2>
              <p>配置用于外部情报搜索的 API。</p>
            </div>
            <button type="button" class="chip" @click="handleTestSearch">
              <span class="material">travel_explore</span>
              测试搜索
            </button>
          </header>
          <div class="grid">
            <label>
              <span>搜索提供商</span>
              <select v-model="local.search_provider">
                <option value="">未配置（直连模式）</option>
                <option value="serpapi">SerpAPI</option>
              </select>
            </label>
            <label>
              <span>API Key</span>
              <input v-model="local.search_api_key" type="text" placeholder="SerpAPI Key" />
            </label>
          </div>
        </section>

        <footer class="form-footer">
          <button class="primary" type="submit" :disabled="settingsStore.loading">保存配置</button>
        </footer>
      </form>
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import { useSettingsStore } from '../stores/settings'

const router = useRouter()
const settingsStore = useSettingsStore()
const { data } = storeToRefs(settingsStore)

const local = reactive({
  llm_base_url: '',
  llm_api_key: '',
  llm_model: '',
  my_company_name: '',
  my_product_profile: '',
  smtp_host: '',
  smtp_port: 465,
  smtp_username: '',
  smtp_password: '',
  admin_email: '',
  rating_guideline: '',
  search_provider: '',
  search_api_key: '',
})

const fieldKeys = Object.keys(local)

onMounted(() => {
  settingsStore.fetchSettings()
})

watch(
  data,
  (value) => {
    if (!value) return
    Object.assign(local, value)
  },
  { immediate: true }
)

const isDirty = computed(() => {
  if (!data.value) {
    return fieldKeys.some((key) => {
      const value = local[key]
      return value !== '' && value !== null && value !== undefined
    })
  }
  return fieldKeys.some((key) => local[key] !== data.value[key])
})

const ensureSaved = async () => {
  if (!isDirty.value) {
    return true
  }
  await settingsStore.saveSettings({ ...local })
  return !isDirty.value
}

const handleSave = async () => {
  await settingsStore.saveSettings({ ...local })
}

const handleTestLLM = async () => {
  if (!(await ensureSaved())) {
    return
  }
  await settingsStore.testLLM()
}

const handleTestSMTP = async () => {
  if (!(await ensureSaved())) {
    return
  }
  await settingsStore.testSMTP()
}

const handleTestSearch = async () => {
  if (!(await ensureSaved())) {
    return
  }
  await settingsStore.testSearch()
}

const goHome = () => {
  router.push({ name: 'home' })
}

const goBack = () => {
  goHome()
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined');

.settings-shell {
  min-height: 100vh;
  display: flex;
  background: var(--surface-background);
  color: var(--text-primary);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.sidebar {
  width: 240px;
  background: #ffffff;
  border-right: 1px solid var(--border-subtle);
  padding: 32px 24px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.brand {
  display: flex;
  gap: 12px;
  align-items: center;
}

.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
}

.brand-title {
  margin: 0;
  font-weight: 600;
}

.brand-sub {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.menu {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.menu-item {
  border: none;
  border-radius: 12px;
  padding: 10px 14px;
  background: transparent;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.menu-item--active {
  background: rgba(37, 99, 235, 0.12);
  color: var(--accent-color);
}

.material {
  font-family: 'Material Symbols Outlined';
  font-size: 20px;
  display: inline-flex;
}

.main {
  flex: 1;
  padding: 48px 64px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.page-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-head h1 {
  margin: 0;
  font-size: 32px;
  font-weight: 800;
}

.page-head p {
  margin: 8px 0 0;
  color: var(--text-secondary);
}

.ghost {
  border: 1px solid var(--border-subtle);
  background: transparent;
  border-radius: 999px;
  padding: 8px 18px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.card {
  background: #ffffff;
  border-radius: 24px;
  padding: 28px 32px;
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.08);
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.card h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}

.card p {
  margin: 6px 0 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}

.grid-two {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 24px;
}

.card--stretch {
  display: flex;
  flex-direction: column;
}

label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
}

input,
textarea,
select {
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-subtle);
  background: var(--surface-muted);
  font-size: 14px;
}

textarea {
  resize: vertical;
}

select {
  appearance: none;
}

.chip {
  border: none;
  border-radius: 999px;
  padding: 10px 18px;
  background: rgba(37, 99, 235, 0.12);
  color: var(--accent-color);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.form-footer {
  display: flex;
  justify-content: flex-end;
}

button.primary {
  border: none;
  border-radius: 12px;
  padding: 12px 28px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

button.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
