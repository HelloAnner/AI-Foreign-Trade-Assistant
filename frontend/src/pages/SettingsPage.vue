<template>
  <FlowLayout :step="1" :total="1" title="全局配置" subtitle="配置软件核心参数，保障 AI 助手稳定运行。">
    <form id="settings-form" class="settings-form" @submit.prevent="handleSave">
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
            <input v-model="local.llm_api_key" type="text" placeholder="sk-…" />
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
        <label class="full">
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

        <div class="card">
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

    </form>
    <template #footer>
      <button class="ghost" type="button" :disabled="!isDirty" @click="handleReset">取消</button>
      <button class="primary" type="button" :disabled="settingsStore.loading" @click="handleSave">保存更改</button>
    </template>
  </FlowLayout>
</template>

<script setup>
import { computed, onMounted, reactive, watch } from 'vue'
import { storeToRefs } from 'pinia'
import FlowLayout from '../components/flow/FlowLayout.vue'
import { useSettingsStore } from '../stores/settings'

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
  if (!(await ensureSaved())) return
  await settingsStore.testLLM()
}

const handleReset = () => {
  if (data.value) {
    Object.assign(local, data.value)
  }
}

const handleTestSMTP = async () => {
  if (!(await ensureSaved())) return
  await settingsStore.testSMTP()
}

const handleTestSearch = async () => {
  if (!(await ensureSaved())) return
  await settingsStore.testSearch()
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined');

.settings-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.card {
  background: var(--surface-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
  padding: 28px 32px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.card h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}

.card p {
  margin: 6px 0 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 18px;
}

.grid-two {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 24px;
}

label,
textarea,
select {
  font-size: 14px;
}

label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--text-secondary);
}

label.full {
  display: flex;
}

input,
textarea,
select {
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-default);
  background: #fff;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

input:focus,
textarea:focus,
select:focus {
  outline: none;
  border-color: var(--primary-500);
  box-shadow: 0 0 0 3px rgba(19, 73, 236, 0.14);
}

textarea {
  resize: vertical;
}

select {
  appearance: none;
}

.chip {
  border: none;
  border-radius: var(--radius-full);
  padding: 10px 18px;
  background: rgba(19, 73, 236, 0.12);
  color: var(--primary-500);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.chip .material {
  font-size: 18px;
}

.primary {
  border: none;
  border-radius: var(--radius-full);
  padding: 12px 28px;
  background: var(--primary-500);
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s ease;
}

.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.primary:hover:not(:disabled) {
  background: var(--primary-600);
}

@media (max-width: 768px) {
  .card {
    padding: 24px;
  }
}
</style>
