<template>
  <FlowLayout
    :step="1"
    :total="5"
    title="Step 1: 智能信息获取与聚合"
    subtitle="请核对并校准以下 AI 获取的信息，确保其准确性。"
    :progress="20"
  >
    <section class="cards">
      <div class="query">
        <input
          v-model="queryText"
          :disabled="flowStore.resolving"
          type="text"
          placeholder="输入客户公司全名或官网地址"
        />
        <button :class="hasResult && !flowStore.resolving ? 'success' : 'primary'" :disabled="actionDisabled" @click="handleAction">
          {{ buttonLabel }}
        </button>
      </div>

      <transition name="fade">
        <div v-if="flowStore.resolveResult" key="result" class="result">
          <section class="card">
            <header>
              <h2>公司信息</h2>
              <p>AI 自动抓取的公司基础信息。</p>
            </header>
            <div class="grid">
              <label>
                <span>公司名称</span>
                <input v-model="companyForm.name" type="text" placeholder="如：环球贸易有限公司" />
              </label>
              <label>
                <span>公司官网</span>
                <input v-model="companyForm.website" type="text" placeholder="https://example.com" />
              </label>
              <label>
                <span>国家 / 地区</span>
                <input v-model="companyForm.country" type="text" placeholder="United States" />
              </label>
            </div>
            <label class="full">
              <span>AI 摘要</span>
              <textarea v-model="companyForm.summary" rows="4" placeholder="公司业务概览"></textarea>
            </label>
            <div class="candidates" v-if="candidateList.length">
              <h3>候选官网列表</h3>
              <ul>
                <li v-for="item in candidateList" :key="item.url">
                  <button type="button" class="link" @click="applyCandidate(item.url)">{{ item.url }}</button>
                  <small>{{ item.reason }}</small>
                </li>
              </ul>
            </div>
          </section>

          <section class="card">
            <header>
              <h2>潜在联系人</h2>
              <p>AI 识别到的潜在业务联系人，可继续补充。</p>
            </header>
            <div class="contacts-header">
              <button class="ghost" type="button" @click="addContact">新增联系人</button>
            </div>
            <div v-if="contactsLocal.length" class="contacts">
              <div v-for="(contact, index) in contactsLocal" :key="index" class="contact-row">
                <input v-model="contact.name" type="text" placeholder="联系人姓名" />
                <input v-model="contact.title" type="text" placeholder="职位" />
                <input v-model="contact.email" type="email" placeholder="邮箱地址" />
                <input v-model="contact.source" type="text" placeholder="社交媒体 / 来源" />
                <label class="key">
                  <input type="checkbox" v-model="contact.is_key" /> 重点联系人
                </label>
                <button class="ghost" type="button" @click="removeContact(index)">删除</button>
              </div>
            </div>
            <div v-else class="placeholder">暂无联系人，试着手动添加一个关键联系人。</div>
          </section>
        </div>
      </transition>
    </section>

    <template #footer>
      <div class="footer-actions">
        <button class="ghost" type="button" :disabled="true">返回上一步</button>
        <button class="primary" type="button" @click="handleSave" :disabled="!companyForm.name">
          保存并继续
        </button>
      </div>
    </template>
  </FlowLayout>
</template>

<script setup>
import { inject, ref, watch, computed } from 'vue'
import FlowLayout from './FlowLayout.vue'
import { useFlowStore } from '../../stores/flow'

const flowStore = useFlowStore()
const nav = inject('flowNav', { goNext: () => {} })

const queryText = ref(flowStore.query)
const companyForm = ref({
  name: '',
  website: '',
  country: '',
  summary: '',
})
const contactsLocal = ref([])

const candidateList = computed(() => flowStore.resolveResult?.candidates || [])
const hasResult = computed(() => Boolean(flowStore.resolveResult))
const queryChanged = computed(() => {
  if (!hasResult.value) return false
  const current = (queryText.value || '').trim()
  const original = (flowStore.query || '').trim()
  return current !== original
})

const buttonLabel = computed(() => {
  if (flowStore.resolving) return '分析中…'
  if (queryChanged.value) return '开始分析'
  return hasResult.value ? '下一步' : '开始分析'
})

const actionDisabled = computed(() => {
  if (flowStore.resolving) return true
  if (queryChanged.value) {
    return !queryText.value
  }
  if (hasResult.value) {
    return !companyForm.value.name
  }
  return !queryText.value
})

watch(
  () => flowStore.resolveResult,
  (value) => {
    if (!value) {
      companyForm.value = {
        name: flowStore.query || '',
        website: '',
        country: '',
        summary: '',
      }
      contactsLocal.value = []
      return
    }
    companyForm.value = {
      name: value.name || flowStore.query || '',
      website: value.website || '',
      country: value.country || '',
      summary: value.summary || '',
    }
    contactsLocal.value = (value.contacts || []).map((contact) => ({ ...contact }))
  },
  { immediate: true }
)

watch(
  () => flowStore.contacts,
  (value) => {
    if (!value || !value.length) return
    contactsLocal.value = value.map((item) => ({ ...item }))
  }
)

const handleResolve = async () => {
  if (!queryText.value || flowStore.resolving) return
  await flowStore.startResolve(queryText.value)
}

const handleSave = async () => {
  if (!companyForm.value.name) return
  flowStore.contacts = contactsLocal.value.map((item) => ({ ...item }))
  await flowStore.saveCompany({ ...companyForm.value })
  nav.goNext?.()
}

const handleAction = async () => {
  if (flowStore.resolving) return
  if (queryChanged.value || !hasResult.value) {
    await handleResolve()
  } else {
    await handleSave()
  }
}

const syncContacts = () => {
  flowStore.updateContacts(contactsLocal.value.map((item) => ({ ...item })))
}

const addContact = () => {
  contactsLocal.value.push({
    name: '',
    title: '',
    email: '',
    source: '',
    is_key: contactsLocal.value.length === 0,
  })
}

const removeContact = (index) => {
  contactsLocal.value.splice(index, 1)
}

const applyCandidate = (url) => {
  if (!url) return
  companyForm.value.website = url
}
</script>

<style scoped>
.cards {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.query {
  display: flex;
  gap: 16px;
}

.query input {
  flex: 1;
}

.card {
  background: #ffffff;
  border-radius: 24px;
  padding: 32px;
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.08);
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.card header {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.card h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}

.card p {
  margin: 0;
  color: var(--text-secondary);
}

input,
textarea {
  width: 100%;
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-subtle);
  background: var(--surface-muted);
  font-size: 14px;
}

textarea {
  resize: vertical;
}

button {
  border: none;
  border-radius: 12px;
  padding: 12px 24px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

button.primary {
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
}

button.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

button.success {
  background: linear-gradient(135deg, #22c55e, #16a34a);
  color: #fff;
}

button.success:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

button.ghost {
  background: rgba(148, 163, 184, 0.16);
  color: var(--text-secondary);
}

button.link {
  border: none;
  background: none;
  padding: 0;
  color: var(--accent-color);
  cursor: pointer;
}

.result {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}

.full {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.candidates h3 {
  margin: 0;
  font-size: 16px;
}

.candidates ul {
  padding-left: 18px;
  margin: 8px 0 0;
}

.candidates li {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 8px;
}

.contacts-header {
  display: flex;
  justify-content: flex-end;
}

.contacts {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.contact-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(160px, 1fr)) 140px auto;
  gap: 12px;
  align-items: center;
}

.contact-row .key {
  display: flex;
  gap: 8px;
  align-items: center;
  color: var(--text-secondary);
}

.placeholder {
  padding: 16px;
  border-radius: 12px;
  border: 1px dashed var(--border-subtle);
  color: var(--text-tertiary);
  text-align: center;
}

.footer-actions {
  display: flex;
  gap: 12px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
