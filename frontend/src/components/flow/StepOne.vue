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
          <select v-model="companyForm.country">
            <option value="">请选择</option>
            <option v-for="country in countryOptions" :key="country" :value="country">{{ country }}</option>
          </select>
        </label>
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

const flowStore = useFlowStore()
const nav = inject('flowNav', { goNext: () => {} })

const countryOptions = ['美国', '德国', '新加坡', '英国', '日本', '中国', '加拿大']

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

const nextDisabled = computed(() => {
  if (flowStore.resolving) return true
  if (!companyForm.website.trim()) return true
  if (!companyForm.country.trim()) return true
  return false
})

const handleNext = async () => {
  if (nextDisabled.value) return
  if (!companyForm.name) {
    companyForm.name = flowStore.query || companyForm.website
  }
  flowStore.contacts = sanitizedContacts.value
  await flowStore.saveCompany({
    name: companyForm.name,
    website: companyForm.website,
    country: companyForm.country,
    summary: companyForm.summary,
  })
  nav.goNext?.()
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
