<template>
  <FlowLayout
    :step="1"
    :total="5"
    title="Step 1: 智能信息获取与聚合"
    subtitle="请核对 AI 自动拉取的客户信息，可按需补充后继续。"
  >
    <section class="card">
      <header class="card__head">
        <div>
          <h2>客户基础信息</h2>
          <p>官网与国家为后续分析的关键依据。</p>
        </div>
        <p v-if="companyForm.name" class="card__hint">客户名称：{{ companyForm.name }}</p>
      </header>
      <div class="grid">
        <label class="full">
          <span>公司官网</span>
          <input v-model="companyForm.website" type="text" placeholder="https://www.example.com" />
        </label>
        <label>
          <span>国家/地区</span>
          <input v-model="companyForm.country" type="text" list="country-options" placeholder="例如：美国 / United States" />
          <datalist id="country-options">
            <option v-for="country in countryOptions" :key="country" :value="country" />
          </datalist>
        </label>
      </div>
      <div class="overview">
        <span>客户基本信息概述</span>
        <p>{{ companyOverview }}</p>
      </div>
    </section>

    <section v-if="automationJob" class="automation-banner" :class="automationBannerClass">
      <span class="material">autorenew</span>
      <div class="automation-banner__content">
        <p>{{ automationMessage }}</p>
        <small v-if="automationJob.last_error && automationStatus === 'failed'">{{ automationJob.last_error }}</small>
        <small v-else-if="automationStatus === 'running' && automationStageLabel">当前阶段：{{ automationStageLabel }}</small>
      </div>
    </section>

    <section class="card">
      <header class="card__head">
        <div>
          <h2>潜在联系人</h2>
          <p>至少保留一位业务联系人，便于后续外联。</p>
        </div>
        <button class="ghost" type="button" @click="addContact">
          <span class="material">add</span>
          新增联系人
        </button>
      </header>
      <div v-if="contactsLocal.length" class="contacts">
        <div v-for="(contact, index) in contactsLocal" :key="index" class="contact-row">
          <input v-model="contact.name" type="text" placeholder="姓名" />
          <input v-model="contact.title" type="text" placeholder="职位" />
          <input v-model="contact.email" type="email" placeholder="邮箱" />
          <input v-model="contact.phone" type="text" placeholder="电话（选填）" />
          <button class="icon-button" type="button" @click="removeContact(index)" :disabled="contactsLocal.length === 1">
            <span class="material">delete</span>
          </button>
        </div>
      </div>
      <p v-else class="placeholder">暂无联系人，点击右上角按钮新增。</p>
    </section>

    <template #footer>
      <button class="primary" type="button" :disabled="nextDisabled" @click="handleNext">
        {{ flowStore.resolving ? '同步中…' : '下一步' }}
      </button>
    </template>
  </FlowLayout>
</template>

<script setup>
import { inject, onMounted, reactive, ref, watch, computed } from 'vue'
import FlowLayout from './FlowLayout.vue'
import { useFlowStore } from '../../stores/flow'
import { useUiStore } from '../../stores/ui'

const flowStore = useFlowStore()
const uiStore = useUiStore()
const nav = inject('flowNav', { goNext: () => {} })

const defaultCountries = ['美国', '中国', '加拿大', '德国', '法国', '英国', '日本', '新加坡', '澳大利亚', 'United States', 'Canada', 'Germany', 'France', 'United Kingdom', 'Japan', 'Singapore', 'Australia']
const countryOptions = computed(() => {
  const set = new Set(defaultCountries)
  if (flowStore.resolveResult?.country) {
    set.add(flowStore.resolveResult.country)
  }
  if (companyForm.country) {
    set.add(companyForm.country)
  }
  return Array.from(set)
})

const companyForm = reactive({
  name: '',
  website: '',
  country: '',
  summary: '',
})

const contactsLocal = ref([])

const ensureResolve = async (query) => {
  const trimmed = (query || '').trim()
  if (!trimmed || flowStore.resolving) return
  await flowStore.startResolve(trimmed)
}

onMounted(async () => {
  if (!flowStore.resolveResult && flowStore.query) {
    await ensureResolve(flowStore.query)
  }
  hydrateFromStore()
})

const hydrateFromStore = () => {
  if (flowStore.resolveResult) {
    companyForm.name = flowStore.resolveResult.name || flowStore.query || ''
    companyForm.website = flowStore.resolveResult.website || ''
    companyForm.country = flowStore.resolveResult.country || ''
    companyForm.summary = flowStore.resolveResult.summary || ''
  } else {
    companyForm.name = flowStore.query || ''
  }
  if (flowStore.website) {
    companyForm.website = flowStore.website
  }
  if (flowStore.country) {
    companyForm.country = flowStore.country
  }
  if (flowStore.summary) {
    companyForm.summary = flowStore.summary
  }
  const contacts = flowStore.contacts && flowStore.contacts.length ? flowStore.contacts : (flowStore.resolveResult?.contacts || [])
  contactsLocal.value = (contacts || []).map((contact) => ({
    name: contact.name || '',
    title: contact.title || '',
    email: contact.email || '',
    phone: contact.phone || '',
    source: contact.source || '',
  }))
  if (!contactsLocal.value.length) {
    contactsLocal.value.push({ name: '', title: '', email: '', phone: '', source: '' })
  }
}

watch(
  () => flowStore.resolveResult,
  () => {
    hydrateFromStore()
  },
  { deep: true }
)

watch(
  () => flowStore.contacts,
  () => {
    if (!flowStore.resolveResult) {
      hydrateFromStore()
    }
  },
  { deep: true }
)

watch(
  () => flowStore.query,
  (value, oldValue) => {
    if (value && value !== oldValue && !flowStore.resolveResult) {
      ensureResolve(value)
    }
  }
)

const addContact = () => {
  contactsLocal.value.push({ name: '', title: '', email: '', phone: '', source: '' })
}

const removeContact = (index) => {
  if (contactsLocal.value.length === 1) return
  contactsLocal.value.splice(index, 1)
}

const sanitizedContacts = computed(() =>
  contactsLocal.value.map((contact, index) => ({
    name: contact.name.trim(),
    title: contact.title.trim(),
    email: contact.email.trim(),
    phone: contact.phone.trim(),
    source: contact.source?.trim() || '',
    is_key: index === 0,
  }))
)

const automationJob = computed(() => flowStore.automationJob)
const automationStatus = computed(() => String(automationJob.value?.status || '').toLowerCase())
const automationActive = computed(() => automationStatus.value === 'queued' || automationStatus.value === 'running')
const automationStage = computed(() => String(automationJob.value?.stage || '').toLowerCase())

const stageLabelMap = {
  grading: '智能评级',
  analysis: '客户分析',
  email: '开发信生成',
  followup: '自动跟进设置',
}

const automationStageLabel = computed(() => {
  if (!automationJob.value) return ''
  return stageLabelMap[automationStage.value] || automationJob.value.stage || ''
})

const automationMessage = computed(() => {
  const status = automationStatus.value
  if (!automationJob.value) {
    return ''
  }
  if (status === 'queued') {
    return '后台自动化流程已排队，系统将依次完成剩余步骤。'
  }
  if (status === 'running') {
    const stageLabel = automationStageLabel.value || '自动化处理'
    return `后台正在执行：${stageLabel}…`
  }
  if (status === 'failed') {
    return '自动化流程执行失败，请查看提示并手动处理。'
  }
  if (status === 'completed') {
    if (automationStage.value === 'stopped') {
      return automationJob.value.last_error || '自动化已结束。'
    }
    return '自动化流程已完成，数据已更新。'
  }
  return '自动化流程状态已更新。'
})

const automationBannerClass = computed(() => {
  const status = automationStatus.value
  if (status === 'failed') return 'automation-banner--error'
  if (status === 'completed' && automationStage.value === 'stopped') return 'automation-banner--warning'
  if (status === 'completed') return 'automation-banner--success'
  return 'automation-banner--info'
})

const nextDisabled = computed(() => {
  if (flowStore.resolving) return true
  if (!companyForm.website.trim()) return true
  if (!companyForm.country.trim()) return true
  if (automationActive.value) return true
  return false
})

const companyOverview = computed(() => {
  if (companyForm.summary?.trim()) return companyForm.summary.trim()
  if (flowStore.resolveResult?.summary) return flowStore.resolveResult.summary
  return '暂无概述，生成分析后将自动补全。'
})

const handleNext = async () => {
  if (nextDisabled.value) return
  if (!companyForm.name) {
    companyForm.name = flowStore.query || companyForm.website
  }
  flowStore.contacts = sanitizedContacts.value
  try {
    await flowStore.saveCompany({
      name: companyForm.name,
      website: companyForm.website,
      country: companyForm.country,
      summary: companyForm.summary,
    })
    if (automationActive.value || automationStatus.value === 'queued' || automationStatus.value === 'running') {
      uiStore.pushToast('客户信息已保存，后台自动化流程将继续完成后续步骤。', 'info')
      return
    }
    nav.goNext?.()
  } catch (error) {
    // 错误已在 store 内部提示
  }
}
</script>

<style scoped>
.card {
  background: var(--surface-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
  padding: 32px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card__head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.card__head h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}

.card__head p {
  margin: 6px 0 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.card__hint {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}

label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
  color: var(--text-secondary);
}

label.full {
  grid-column: 1 / -1;
}

.overview {
  margin-top: 12px;
  padding: 18px;
  border-radius: 14px;
  background: var(--surface-subtle);
  border: 1px solid var(--border-default);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.overview span {
  font-size: 14px;
  color: var(--text-secondary);
}

.overview p {
  margin: 0;
  font-size: 14px;
  color: var(--text-primary);
  text-align: left;
  white-space: pre-wrap;
}

.automation-banner {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px 20px;
  border-radius: 14px;
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
}

.automation-banner .material {
  font-size: 20px;
  color: var(--primary-500);
}

.automation-banner__content {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--text-secondary);
}

.automation-banner__content p {
  margin: 0;
  font-weight: 600;
  color: var(--text-primary);
}

.automation-banner--info {
  background: rgba(59, 130, 246, 0.08);
}

.automation-banner--success {
  background: rgba(34, 197, 94, 0.12);
}

.automation-banner--success .material {
  color: #16a34a;
}

.automation-banner--warning {
  background: rgba(234, 179, 8, 0.12);
}

.automation-banner--warning .material {
  color: #d97706;
}

.automation-banner--error {
  background: rgba(248, 113, 113, 0.14);
}

.automation-banner--error .material {
  color: #ef4444;
}

input,
select {
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-default);
  background: #fff;
  font-size: 14px;
}

.contacts {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.contact-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)) 40px;
  gap: 12px;
  align-items: center;
}

.icon-button {
  border: none;
  background: transparent;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 6px;
  cursor: pointer;
  color: var(--text-secondary);
}

.icon-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.icon-button:hover:not(:disabled) {
  color: var(--primary-500);
}

.placeholder {
  margin: 0;
  padding: 18px;
  border: 1px dashed var(--border-default);
  border-radius: 14px;
  color: var(--text-tertiary);
  text-align: center;
}

.ghost {
  border: 1px solid var(--border-default);
  background: #fff;
  border-radius: var(--radius-full);
  padding: 8px 18px;
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.primary {
  border: none;
  border-radius: var(--radius-full);
  background: var(--primary-500);
  color: #fff;
  padding: 12px 28px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
}

.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.material {
  font-family: 'Material Symbols Outlined';
  font-size: 18px;
}

@media (max-width: 768px) {
  .card {
    padding: 24px;
  }

  .contact-row {
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  }
}
</style>
