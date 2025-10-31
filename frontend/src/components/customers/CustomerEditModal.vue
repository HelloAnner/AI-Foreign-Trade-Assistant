<template>
  <div class="editor__backdrop" role="dialog" aria-modal="true">
    <div class="editor" v-if="!loading && form">
      <header class="editor__header">
        <div>
          <h3>编辑客户信息</h3>
          <p>一次性调整五个步骤的全部内容。</p>
        </div>
        <button class="icon-button" type="button" @click="emitClose">
          <span class="material">close</span>
        </button>
      </header>

      <div class="editor__content">
        <section class="section">
          <h4>基础信息</h4>
          <div class="grid two">
            <label>
              <span>公司名称</span>
              <input v-model="form.name" type="text" placeholder="如：环球贸易有限公司" />
            </label>
            <label>
              <span>公司官网</span>
              <input v-model="form.website" type="text" placeholder="https://example.com" />
            </label>
            <label>
              <span>国家/地区</span>
              <input v-model="form.country" type="text" placeholder="United States" />
            </label>
            <label class="full">
              <span>公司摘要</span>
              <textarea v-model="form.summary" rows="3" placeholder="公司业务概览"></textarea>
            </label>
          </div>
        </section>

        <section class="section">
          <header class="section__head">
            <h4>潜在联系人</h4>
            <button type="button" class="ghost" @click="addContact">
              <span class="material">add</span>
              新增联系人
            </button>
          </header>
          <div v-if="form.contacts.length" class="contacts">
            <div v-for="(contact, index) in form.contacts" :key="index" class="contact-row">
              <input v-model="contact.name" type="text" placeholder="姓名" />
              <input v-model="contact.title" type="text" placeholder="职位" />
              <input v-model="contact.email" type="email" placeholder="邮箱" />
              <input v-model="contact.phone" type="text" placeholder="电话（选填）" />
              <button class="icon-button" type="button" @click="removeContact(index)">
                <span class="material">delete</span>
              </button>
            </div>
          </div>
          <p v-else class="hint">暂无联系人，点击“新增联系人”补充关键联系人。</p>
        </section>

        <section class="section">
          <h4>AI 评级</h4>
          <div class="grid two">
            <label>
              <span>最终评级</span>
              <select v-model="form.grade">
                <option value="">未评级</option>
                <option value="S">S</option>
                <option value="A">A</option>
                <option value="B">B</option>
                <option value="C">C</option>
                <option value="UNKNOWN">UNKNOWN</option>
              </select>
            </label>
            <label class="full">
              <span>评级理由</span>
              <textarea v-model="form.gradeReason" rows="3" placeholder="说明评级的主要依据"></textarea>
            </label>
          </div>
        </section>

        <section class="section">
          <h4>客户深度分析</h4>
          <div class="grid one">
            <label>
              <span>客户核心业务</span>
              <textarea v-model="form.analysis.core_business" rows="3"></textarea>
            </label>
            <label>
              <span>潜在需求 / 痛点</span>
              <textarea v-model="form.analysis.pain_points" rows="3"></textarea>
            </label>
            <label>
              <span>我方切入点</span>
              <textarea v-model="form.analysis.my_entry_points" rows="3"></textarea>
            </label>
            <label>
              <span>完整报告</span>
              <textarea v-model="form.analysis.full_report" rows="5"></textarea>
            </label>
          </div>
        </section>

        <section class="section">
          <h4>开发信草稿</h4>
          <div v-if="form.email.email_id" class="grid one">
            <label>
              <span>邮件标题</span>
              <input v-model="form.email.subject" type="text" />
            </label>
            <label>
              <span>邮件正文</span>
              <textarea v-model="form.email.body" rows="10"></textarea>
            </label>
          </div>
          <p v-else class="hint">尚未生成开发信草稿，请在主流程中生成后再进行编辑。</p>
        </section>

        <section class="section">
          <h4>自动跟进计划</h4>
          <div class="schedule">
            <div class="schedule__current" v-if="form.followup.next_due">
              <span class="label">当前计划</span>
              <span>{{ formatDate(form.followup.next_due) }}</span>
            </div>
            <div class="schedule__options">
              <button
                v-for="option in scheduleOptions"
                :key="option"
                type="button"
                :class="['ghost', { active: form.followup.delay === option } ]"
                :disabled="!form.email.email_id"
                @click="toggleSchedule(option)"
              >
                {{ option }} 天后
              </button>
            </div>
            <p v-if="!form.email.email_id" class="hint">保存前需要先有开发信草稿才能设置自动跟进。</p>
          </div>
        </section>
      </div>

      <footer class="editor__footer">
        <button class="ghost" type="button" @click="emitClose">取消</button>
        <button class="primary" type="button" :disabled="saving" @click="handleSave">
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </footer>
    </div>
    <div v-else class="editor__loading">
      <div class="spinner"></div>
      <p>加载客户信息…</p>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive, watch, ref } from 'vue'
import { useUiStore } from '../../stores/ui'
import {
  updateCompany,
  confirmGrade,
  updateAnalysis,
  updateEmailDraft,
  scheduleFollowup,
} from '../../api/flow'

const props = defineProps({
  customer: { type: Object, default: null },
  loading: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'updated'])

const ui = useUiStore()

const form = reactive({
  id: null,
  name: '',
  website: '',
  country: '',
  summary: '',
  contacts: [],
  grade: '',
  gradeReason: '',
  analysis: {
    core_business: '',
    pain_points: '',
    my_entry_points: '',
    full_report: '',
  },
  email: {
    email_id: null,
    subject: '',
    body: '',
  },
  followup: {
    delay: null,
    next_due: '',
  },
  sourceJSON: null,
})

const original = reactive({
  grade: '',
  gradeReason: '',
  analysisEnabled: false,
  emailEnabled: false,
  followupDue: '',
})

const scheduleOptions = [3, 7, 14]
const saving = ref(false)

const formatDate = (value) => {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

const hydrateForm = (customer) => {
  if (!customer) return
  form.id = customer.id
  form.name = customer.name || ''
  form.website = customer.website || ''
  form.country = customer.country || ''
  form.summary = customer.summary || ''
  form.contacts = (customer.contacts || []).map((contact, index) => ({
    name: contact.name || '',
    title: contact.title || '',
    email: contact.email || '',
    phone: contact.phone || '',
    source: contact.source || '',
    is_key: index === 0 || Boolean(contact.is_key),
  }))
  form.grade = customer.grade || ''
  form.gradeReason = customer.grade_reason || ''
  form.analysis = {
    core_business: customer.analysis?.core_business || '',
    pain_points: customer.analysis?.pain_points || '',
    my_entry_points: customer.analysis?.my_entry_points || '',
    full_report: customer.analysis?.full_report || '',
  }
  form.email = {
    email_id: customer.email_draft?.email_id || null,
    subject: customer.email_draft?.subject || '',
    body: customer.email_draft?.body || '',
  }
  form.followup = {
    delay: null,
    next_due: customer.scheduled_task?.due_at || '',
  }
  form.sourceJSON = customer.source_json || null

  original.grade = form.grade
  original.gradeReason = form.gradeReason
  original.analysisEnabled = Boolean(customer.analysis)
  original.emailEnabled = Boolean(customer.email_draft)
  original.followupDue = form.followup.next_due
}

watch(
  () => props.customer,
  (value) => {
    hydrateForm(value)
  },
  { immediate: true }
)

const emitClose = () => {
  if (!saving.value) emit('close')
}

const addContact = () => {
  form.contacts.push({
    name: '',
    title: '',
    email: '',
    phone: '',
    source: '',
    is_key: form.contacts.length === 0,
  })
}

const removeContact = (index) => {
  form.contacts.splice(index, 1)
}

const toggleSchedule = (days) => {
  form.followup.delay = form.followup.delay === days ? null : days
}

const sanitizedContacts = computed(() => {
  const mapped = form.contacts.map((contact, index) => ({
    name: contact.name?.trim() || '',
    title: contact.title?.trim() || '',
    email: contact.email?.trim() || '',
    phone: contact.phone?.trim() || '',
    source: contact.source?.trim() || '',
    is_key: index === 0,
  }))
  if (mapped.length && !mapped.some((contact) => contact.is_key)) {
    mapped[0].is_key = true
  }
  return mapped
})

const handleSave = async () => {
  if (!form.id) return
  saving.value = true
  try {
    const payload = {
      name: form.name,
      website: form.website,
      country: form.country,
      summary: form.summary,
      contacts: sanitizedContacts.value,
    }
    if (form.sourceJSON) {
      payload.source_json = form.sourceJSON
    }
    await updateCompany(form.id, payload)

    if (form.grade && (form.grade !== original.grade || form.gradeReason !== original.gradeReason)) {
      await confirmGrade(form.id, { grade: form.grade, reason: form.gradeReason?.trim() || '' })
    }

    const hasAnalysisInput =
      form.analysis.core_business?.trim() ||
      form.analysis.pain_points?.trim() ||
      form.analysis.my_entry_points?.trim() ||
      form.analysis.full_report?.trim()
    if (hasAnalysisInput) {
      await updateAnalysis(form.id, {
        core_business: form.analysis.core_business,
        pain_points: form.analysis.pain_points,
        my_entry_points: form.analysis.my_entry_points,
        full_report: form.analysis.full_report,
      })
    }

    if (form.email.email_id) {
      await updateEmailDraft(form.email.email_id, {
        subject: form.email.subject,
        body: form.email.body,
      })
    }

    if (form.followup.delay && form.email.email_id) {
      const scheduleResp = await scheduleFollowup({
        customer_id: form.id,
        context_email_id: form.email.email_id,
        delay_days: form.followup.delay,
      })
      if (scheduleResp?.ok && scheduleResp.data?.due_at) {
        form.followup.next_due = scheduleResp.data.due_at
      }
    }

    ui.pushToast('客户信息已保存', 'success')
    emit('updated')
  } catch (error) {
    console.error('Failed to save customer', error)
    ui.pushToast(error.message || '保存客户信息失败', 'error')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.editor__backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  z-index: 120;
}

.editor {
  width: min(100%, 980px);
  max-height: calc(100vh - 64px);
  background: #fff;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  box-shadow: 0 24px 48px rgba(15, 23, 42, 0.2);
}

.editor__header {
  padding: 24px 32px;
  border-bottom: 1px solid var(--border-default);
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.editor__header h3 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
}

.editor__header p {
  margin: 6px 0 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.editor__content {
  padding: 24px 32px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
}

.section__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.grid {
  display: grid;
  gap: 16px;
}

.grid.one {
  grid-template-columns: 1fr;
}

.grid.two {
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
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
textarea,
select {
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--border-default);
  background: #fff;
  font-size: 14px;
}

textarea {
  resize: vertical;
}

.contacts {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.contact-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)) auto;
  gap: 12px;
  align-items: center;
  background: var(--surface-subtle);
  border-radius: 14px;
  padding: 14px;
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

.icon-button:hover {
  color: var(--primary-500);
}

.material {
  font-family: 'Material Symbols Outlined';
  font-size: 20px;
}

.hint {
  color: var(--text-tertiary);
  font-size: 13px;
  margin: 0;
}

.schedule {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.schedule__options {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
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
  transition: background 0.2s ease, color 0.2s ease;
}

.ghost:hover:not(:disabled) {
  color: var(--primary-500);
  border-color: var(--primary-500);
}

.ghost.active {
  background: rgba(19, 73, 236, 0.14);
  border-color: var(--primary-500);
  color: var(--primary-500);
}

.ghost:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.editor__footer {
  padding: 20px 32px;
  border-top: 1px solid var(--border-default);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.primary {
  border: none;
  border-radius: var(--radius-full);
  background: var(--primary-500);
  color: #fff;
  padding: 10px 26px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.editor__loading {
  width: 320px;
  background: #fff;
  border-radius: var(--radius-lg);
  padding: 32px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  box-shadow: 0 24px 48px rgba(15, 23, 42, 0.2);
}

.spinner {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 4px solid rgba(19, 73, 236, 0.18);
  border-top-color: var(--primary-500);
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .editor__backdrop {
    padding: 12px;
  }
  .editor {
    max-height: calc(100vh - 24px);
  }
  .grid.two {
    grid-template-columns: 1fr;
  }
  .contact-row {
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  }
}
</style>
