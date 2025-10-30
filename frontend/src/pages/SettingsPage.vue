<template>
  <div class="settings-page">
    <header class="settings-page__header">
      <button class="settings-page__back" @click="goBack">返回</button>
      <h1 class="settings-page__title">全局配置</h1>
    </header>
    <section class="settings-page__body">
      <p class="settings-page__intro">
        在这里配置 LLM、SMTP、客户评级标准及搜索 API。当前页面为功能骨架，随开发推进将补齐完整表单与交互。
      </p>
      <div class="settings-preview" v-if="!loading">
        <article class="settings-preview__card">
          <h2>大模型设置</h2>
          <dl>
            <div>
              <dt>Base URL</dt>
              <dd>{{ sanitized(data.llm_base_url) }}</dd>
            </div>
            <div>
              <dt>模型名称</dt>
              <dd>{{ sanitized(data.llm_model) }}</dd>
            </div>
          </dl>
          <button class="link-button" @click="settingsStore.testLLM">测试 LLM 连通性</button>
        </article>
        <article class="settings-preview__card">
          <h2>邮件设置</h2>
          <dl>
            <div>
              <dt>SMTP 主机</dt>
              <dd>{{ sanitized(data.smtp_host) }}</dd>
            </div>
            <div>
              <dt>管理员邮箱</dt>
              <dd>{{ sanitized(data.admin_email) }}</dd>
            </div>
          </dl>
          <button class="link-button" @click="settingsStore.testSMTP">发送测试邮件</button>
        </article>
      </div>
      <div v-else class="settings-page__loading">正在加载配置...</div>
      <form class="search-config" @submit.prevent="saveSearchConfig">
        <h3 class="search-config__title">搜索 API</h3>
        <p class="search-config__remark">
          推荐使用 <span>SerpAPI</span>（https://serpapi.com/search-api），成功配置后系统可自动调用真实搜索结果。
        </p>
        <div class="search-config__fields">
          <label class="search-config__field">
            <span>搜索提供商</span>
            <select v-model="localSearch.search_provider">
              <option value="">未配置（直连模式）</option>
              <option value="serpapi">SerpAPI</option>
            </select>
          </label>
          <label class="search-config__field">
            <span>API Key</span>
            <input
              v-model="localSearch.search_api_key"
              type="text"
              placeholder="填入 SerpAPI Key"
              autocomplete="off"
            />
          </label>
        </div>
        <button class="search-config__submit" type="submit" :disabled="loading">
          保存搜索配置
        </button>
        <button class="link-button" type="button" @click="settingsStore.testSearch">
          测试搜索 API
        </button>
      </form>
    </section>
  </div>
</template>

<script setup>
import { onMounted, reactive, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import { useSettingsStore } from '../stores/settings'

const router = useRouter()
const settingsStore = useSettingsStore()
const { data, loading } = storeToRefs(settingsStore)

const localSearch = reactive({
  search_provider: '',
  search_api_key: '',
})

onMounted(() => {
  settingsStore.fetchSettings()
})

const sanitized = (value) => {
  if (!value) return '未配置'
  return value
}

watch(
  data,
  (value) => {
    if (!value) return
    localSearch.search_provider = value.search_provider || ''
    localSearch.search_api_key = value.search_api_key || ''
  },
  { immediate: true }
)

const saveSearchConfig = () => {
  settingsStore.saveSettings({
    search_provider: localSearch.search_provider,
    search_api_key: localSearch.search_api_key,
  })
}

const goBack = () => {
  router.push({ name: 'home' })
}
</script>

<style scoped>
.settings-page {
  min-height: 100vh;
  background: var(--surface-background);
  color: var(--text-primary);
  padding: 32px 48px;
}

.settings-page__header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.settings-page__back {
  border: 1px solid var(--border-subtle);
  padding: 8px 20px;
  border-radius: 999px;
  background: transparent;
  cursor: pointer;
}

.settings-page__title {
  font-size: 26px;
  font-weight: 600;
}

.settings-page__intro {
  max-width: 540px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.settings-preview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
  margin-top: 32px;
}

.settings-preview__card {
  background: var(--surface-elevated);
  border-radius: 16px;
  padding: 20px 24px;
  box-shadow: 0 14px 28px rgba(15, 23, 42, 0.04);
}

.settings-preview__card h2 {
  margin-top: 0;
  margin-bottom: 16px;
  font-size: 18px;
  font-weight: 600;
}

.settings-preview__card dl {
  display: grid;
  row-gap: 12px;
  column-gap: 12px;
  margin: 0;
}

.settings-preview__card dt {
  font-size: 13px;
  color: var(--text-tertiary);
}

.settings-preview__card dd {
  margin: 4px 0 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.settings-page__loading {
  margin-top: 32px;
  color: var(--text-tertiary);
}

.search-config {
  margin-top: 40px;
  background: var(--surface-elevated);
  border-radius: 18px;
  padding: 24px;
  box-shadow: 0 16px 32px rgba(15, 23, 42, 0.06);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.search-config__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.search-config__remark {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.search-config__remark span {
  color: var(--accent-color);
  font-weight: 600;
}

.search-config__fields {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}

.search-config__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
  color: var(--text-secondary);
}

.search-config__field select,
.search-config__field input {
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--border-subtle);
  background: var(--surface-muted);
  font-size: 14px;
}

.search-config__submit {
  align-self: flex-start;
  border: none;
  border-radius: 999px;
  padding: 12px 28px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.search-config__submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.link-button {
  margin-top: 12px;
  background: none;
  border: none;
  color: var(--accent-color);
  cursor: pointer;
  padding: 0;
  font-size: 14px;
  text-align: left;
}

.link-button:hover {
  text-decoration: underline;
}
</style>
