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
            <div class="schedule__current" v-if="followupSummary">
              <span class="label">当前计划</span>
              <span>{{ followupSummary }}</span>
            </div>
            <div class="schedule__mode">
              <button
                type="button"
                :class="['pill', { active: form.followup.mode !== 'cron' }]"
                @click="form.followup.mode = 'simple'"
              >
                简单延迟
              </button>
              <button
                type="button"
                :class="['pill', { active: form.followup.mode === 'cron' }]"
                @click="form.followup.mode = 'cron'"
              >
                Cron 表达式
              </button>
            </div>
            <div v-if="form.followup.mode !== 'cron'" class="schedule__simple">
              <label>
                <span>延迟时长</span>
                <input v-model.number="form.followup.delayValue" type="number" min="1" />
              </label>
              <label>
                <span>时间单位</span>
                <select v-model="form.followup.delayUnit">
                  <option value="minutes">分钟</option>
                  <option value="hours">小时</option>
                  <option value="days">天</option>
                </select>
              </label>
              <div class="schedule__quick">
                <span>快捷：</span>
                <button type="button" @click="applyFollowupQuick(30, 'minutes')">30 分钟</button>
                <button type="button" @click="applyFollowupQuick(4, 'hours')">4 小时</button>
                <button type="button" @click="applyFollowupQuick(3, 'days')">3 天</button>
                <button type="button" @click="applyFollowupQuick(7, 'days')">7 天</button>
              </div>
            </div>
            <div v-else class="schedule__cron">
              <label>
                <span>Cron 表达式</span>
                <input
                  v-model="form.followup.cronExpression"
                  type="text"
                  placeholder="如：0 9 * * MON"
                />
              </label>
              <p class="hint">支持标准 5/6 位 cron 语法，以及 @daily、@weekly 等描述。</p>
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
    mode: 'simple',
    delayValue: 3,
    delayUnit: 'days',
    cronExpression: '',
    next_due: '',
  },
  sourceJSON: null,
})

const original = reactive({
  grade: '',
  gradeReason: '',
  analysisEnabled: false,
  emailEnabled: false,
  followup: {
    mode: 'simple',
    delayValue: 3,
    delayUnit: 'days',
    cronExpression: '',
    next_due: '',
  },
})

const saving = ref(false)

const formatDate = (value) => {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

const describeSchedule = (task) => {
  if (!task) return ''
  const due = formatDate(task.due_at || task.next_due)
  if (task.mode === 'cron' && task.cron_expression) {
    return due ? `Cron：${task.cron_expression}（下次 ${due}）` : `Cron：${task.cron_expression}`
  }
  if (task.delay_value && task.delay_unit) {
    const unitLabel = { minutes: '分钟', hours: '小时', days: '天' }[task.delay_unit] || task.delay_unit
    return due
      ? `${task.delay_value}${unitLabel} 后执行，预计 ${due}`
      : `${task.delay_value}${unitLabel} 后执行`
  }
  return due ? `预计 ${due} 执行` : ''
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
  const task = customer.scheduled_task || null
  form.followup.mode = task?.mode || 'simple'
  form.followup.delayValue = task?.delay_value || 3
  form.followup.delayUnit = task?.delay_unit || 'days'
  form.followup.cronExpression = task?.cron_expression || ''
  form.followup.next_due = task?.due_at || ''
  form.sourceJSON = customer.source_json || null

  original.grade = form.grade
  original.gradeReason = form.gradeReason
  original.analysisEnabled = Boolean(customer.analysis)
  original.emailEnabled = Boolean(customer.email_draft)
  Object.assign(original.followup, {
    mode: form.followup.mode,
    delayValue: form.followup.delayValue,
    delayUnit: form.followup.delayUnit,
    cronExpression: form.followup.cronExpression,
    next_due: form.followup.next_due,
  })
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

const applyFollowupQuick = (value, unit) => {
  form.followup.mode = 'simple'
  form.followup.delayValue = value
  form.followup.delayUnit = unit
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

const followupSummary = computed(() =>
  describeSchedule({
    mode: form.followup.mode,
    cron_expression: form.followup.cronExpression,
    delay_value: form.followup.delayValue,
    delay_unit: form.followup.delayUnit,
    due_at: form.followup.next_due,
    next_due: form.followup.next_due,
  })
)

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

    let scheduleUpdated = false
    if (form.email.email_id) {
      const mode = (form.followup.mode || 'simple').toLowerCase() === 'cron' ? 'cron' : 'simple'
      const trimmedCron = form.followup.cronExpression?.trim() || ''
      const delayValue = Number(form.followup.delayValue) > 0 ? Number(form.followup.delayValue) : 3
      const delayUnit = form.followup.delayUnit || 'days'
      const prev = original.followup || {}
      let shouldSchedule = false
      if (mode === 'cron') {
        if (trimmedCron) {
          shouldSchedule =
            prev.mode !== 'cron' ||
            trimmedCron !== (prev.cronExpression || '').trim()
        }
      } else {
        shouldSchedule =
          prev.mode === 'cron' ||
          delayValue !== Number(prev.delayValue || 0) ||
          delayUnit !== (prev.delayUnit || 'days')
      }
      if (shouldSchedule) {
        const request = {
          customer_id: form.id,
          context_email_id: form.email.email_id,
          mode,
        }
        if (mode === 'cron') {
          request.cron_expression = trimmedCron
        } else {
          request.delay_value = delayValue
          request.delay_unit = delayUnit
        }
        const scheduleResp = await scheduleFollowup(request)
        if (scheduleResp?.ok && scheduleResp.data) {
          const task = scheduleResp.data
          form.followup.next_due = task.due_at || form.followup.next_due
          form.followup.mode = task.mode || mode
          form.followup.delayValue = task.delay_value || delayValue
          form.followup.delayUnit = task.delay_unit || delayUnit
          form.followup.cronExpression = task.cron_expression || trimmedCron
          Object.assign(original.followup, {
            mode: form.followup.mode,
            delayValue: form.followup.delayValue,
            delayUnit: form.followup.delayUnit,
            cronExpression: form.followup.cronExpression,
            next_due: form.followup.next_due,
          })
          if (describeSchedule(task)) {
            ui.pushToast(`自动跟进已更新：${describeSchedule(task)}`, 'success')
          } else {
            ui.pushToast('自动跟进已更新', 'success')
          }
          scheduleUpdated = true
        }
      }
    }

    if (!scheduleUpdated) {
      ui.pushToast('客户信息已保存', 'success')
    }
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
  gap: 16px;
}

.schedule__current {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 12px;
  background: rgba(19, 73, 236, 0.12);
  color: var(--primary-500);
  font-size: 13px;
}

.schedule__current .label {
  font-weight: 600;
}

.schedule__mode {
  display: inline-flex;
  gap: 6px;
  padding: 4px;
  border-radius: var(--radius-full);
  background: rgba(15, 23, 42, 0.05);
}

.schedule__mode .pill {
  border: none;
  border-radius: var(--radius-full);
  padding: 6px 16px;
  font-size: 13px;
  font-weight: 600;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.schedule__mode .pill.active {
  background: #fff;
  color: var(--primary-500);
  box-shadow: 0 6px 16px rgba(19, 73, 236, 0.12);
}

.schedule__simple {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-end;
}

.schedule__simple label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}

.schedule__simple input,
.schedule__simple select {
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--border-default);
  font-size: 13px;
  min-width: 110px;
}

.schedule__quick {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  align-items: center;
  font-size: 12px;
  color: var(--text-tertiary);
}

.schedule__quick button {
  border: none;
  border-radius: var(--radius-full);
  padding: 6px 12px;
  background: rgba(19, 73, 236, 0.12);
  color: var(--primary-500);
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s ease;
}

.schedule__quick button:hover {
  background: rgba(19, 73, 236, 0.18);
}

.schedule__cron {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.schedule__cron label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}

.schedule__cron input {
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--border-default);
  font-size: 13px;
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
