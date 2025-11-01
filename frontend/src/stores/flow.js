import { defineStore } from 'pinia'
import {
  resolveCompany,
  createCompany,
  updateCompany,
  replaceContacts,
  suggestGrade,
  confirmGrade,
  generateAnalysis,
  updateAnalysis,
  generateEmailDraft,
  updateEmailDraft,
  saveFirstFollowup,
  scheduleFollowup,
  enqueueAutomation,
} from '../api/flow'
import { getCustomerDetail } from '../api/customers'
import { useUiStore } from './ui'

const humanizeUnit = (unit) => {
  switch (unit) {
    case 'minutes':
      return '分钟'
    case 'hours':
      return '小时'
    case 'days':
      return '天'
    default:
      return unit || ''
  }
}

const formatDueAt = (value) => {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

const scheduleToastMessage = (task) => {
  if (!task) return '自动跟进任务已保存'
  if (task.mode === 'cron' && task.cron_expression) {
    const next = formatDueAt(task.due_at)
    return next
      ? `已启用 Cron 自动跟进（${task.cron_expression}），下次执行时间 ${next}`
      : `已启用 Cron 自动跟进（${task.cron_expression}）`
  }
  if (task.delay_value && task.delay_unit) {
    const unit = humanizeUnit(task.delay_unit)
    const next = formatDueAt(task.due_at)
    return next
      ? `已设置 ${task.delay_value}${unit} 后自动跟进，预计 ${next} 触发`
      : `已设置 ${task.delay_value}${unit} 后自动跟进`
  }
  return '自动跟进任务已保存'
}

export const useFlowStore = defineStore('flow', {
  state: () => ({
    step: 1,
    query: '',
    resolving: false,
    automationQueueing: false,
    automationBacklog: [],
    resolveResult: null,
    contacts: [],
    summary: '',
    country: '',
    website: '',
    customerId: null,
    gradeSuggestion: null,
    gradeFinal: null,
    analysis: null,
    emailDraft: null,
    followupId: null,
    scheduledTask: null,
    automationJob: null,
    _automationTimer: null,
    loading: {
      grade: false,
      analysis: false,
      email: false,
      followup: false,
      schedule: false,
    },
  }),
  actions: {
    resetFlow() {
      this.stopAutomationPolling()
      this.step = 1
      this.query = ''
      this.resolving = false
      this.automationQueueing = false
      this.automationBacklog = []
      this.resolveResult = null
      this.contacts = []
      this.summary = ''
      this.country = ''
      this.website = ''
      this.customerId = null
      this.gradeSuggestion = null
      this.gradeFinal = null
      this.analysis = null
      this.emailDraft = null
      this.followupId = null
      this.scheduledTask = null
      this.automationJob = null
      this.loading = {
        grade: false,
        analysis: false,
        email: false,
        followup: false,
        schedule: false,
      }
    },
    setAutomationJob(job) {
      if (job) {
        this.automationJob = { ...job }
        const status = String(job.status || '').toLowerCase()
        if (status === 'queued' || status === 'running') {
          this.startAutomationPolling()
        } else {
          this.stopAutomationPolling()
        }
      } else {
        this.automationJob = null
        this.stopAutomationPolling()
      }
    },
    startAutomationPolling() {
      if (this._automationTimer) return
      this._automationTimer = window.setInterval(() => {
        this.refreshCustomerDetail(true)
      }, 4000)
    },
    stopAutomationPolling() {
      if (this._automationTimer) {
        window.clearInterval(this._automationTimer)
        this._automationTimer = null
      }
    },
    async refreshCustomerDetail(silent = false) {
      if (!this.customerId) return
      const ui = useUiStore()
      try {
        const payload = await getCustomerDetail(this.customerId)
        if (payload.ok && payload.data) {
          const detail = payload.data || {}
          const normalized = { ...detail, customer_id: detail.id }
          const hasAutomation = Object.prototype.hasOwnProperty.call(normalized, 'automation_job')
          this.applyResolvedData(normalized)
          if (!hasAutomation && this.automationJob) {
            this.setAutomationJob(null)
          }
          const job = hasAutomation ? normalized.automation_job : this.automationJob
          const status = job?.status ? String(job.status).toLowerCase() : ''
          if (!job || (status !== 'queued' && status !== 'running')) {
            this.stopAutomationPolling()
          }
        } else if (!silent) {
          ui.pushToast(payload.error || '刷新客户信息失败', 'error')
        }
      } catch (error) {
        if (!silent) {
          console.error('Failed to refresh customer detail', error)
          ui.pushToast(error.message, 'error')
        }
      }
    },
    applyResolvedData(payload = {}) {
      if (!payload || typeof payload !== 'object') return

      const customerId = payload.customer_id ?? payload.id ?? null
      if (customerId) {
        this.customerId = customerId
      }

      if (Array.isArray(payload.contacts)) {
        this.contacts = payload.contacts.map((item, index) => ({
          name: item.name?.trim() || '',
          title: item.title?.trim() || '',
          email: item.email?.trim() || '',
          phone: item.phone?.trim() || '',
          source: item.source?.trim() || '',
          is_key: index === 0 || Boolean(item.is_key),
        }))
      }

      if (payload.summary !== undefined && payload.summary !== null) {
        this.summary = payload.summary
      }
      if (payload.country) {
        this.country = payload.country
      }
      if (payload.website) {
        this.website = payload.website
      }
      if (payload.name) {
        this.query = this.query || payload.name
      }

      const normalizedGrade = (payload.grade || '').toUpperCase()
      if (normalizedGrade && normalizedGrade !== 'UNKNOWN') {
        this.gradeFinal = {
          grade: normalizedGrade,
          reason: payload.grade_reason || '',
        }
      } else if (Object.prototype.hasOwnProperty.call(payload, 'grade')) {
        this.gradeFinal = null
      }

      if (payload.analysis) {
        this.analysis = { ...payload.analysis }
      } else if (Object.prototype.hasOwnProperty.call(payload, 'analysis') && !payload.analysis) {
        this.analysis = null
      }

      if (payload.email_draft) {
        this.emailDraft = { ...payload.email_draft }
      } else if (Object.prototype.hasOwnProperty.call(payload, 'email_draft') && !payload.email_draft) {
        this.emailDraft = null
      }

      if (Object.prototype.hasOwnProperty.call(payload, 'followup_id')) {
        this.followupId = payload.followup_id || null
      }

      if (Object.prototype.hasOwnProperty.call(payload, 'scheduled_task')) {
        this.scheduledTask = payload.scheduled_task || null
      }

      if (Object.prototype.hasOwnProperty.call(payload, 'automation_job')) {
        this.setAutomationJob(payload.automation_job)
      }

      const lastStep = Number(payload.last_step || 0)
      if (lastStep > 0) {
        this.step = Math.max(this.step, lastStep)
      }

      const base = this.resolveResult && typeof this.resolveResult === 'object' ? this.resolveResult : {}
      const merged = { ...base, ...payload }
      if (customerId) {
        merged.customer_id = customerId
      }
      if (Array.isArray(this.contacts) && this.contacts.length) {
        merged.contacts = this.contacts.map((item) => ({ ...item }))
      }
      this.resolveResult = merged
    },
  async startResolve(query) {
    const ui = useUiStore()
    this.resolving = true
    this.query = query
    try {
      const payload = await resolveCompany(query)
      const data = payload.data || {}
      this.applyResolvedData(data)
      if (data.customer_id) {
        this.step = Math.max(1, data.last_step || this.step || 1)
      } else {
        this.customerId = null
        this.gradeFinal = null
        this.analysis = null
        this.emailDraft = null
        this.followupId = null
        this.scheduledTask = null
        this.setAutomationJob(null)
        this.step = 1
      }
      this.gradeSuggestion = null
    } catch (error) {
      ui.pushToast(error.message, 'error')
    } finally {
      this.resolving = false
    }
    },
    async saveCompany(company, options = {}) {
      const ui = useUiStore()
      const { silent = false, advanceStep = true } = options || {}
      const payload = {
        name: company.name,
        website: company.website || this.website,
        country: company.country || this.country,
        summary: company.summary || this.summary,
        contacts: this.contacts,
      }
      if (this.resolveResult && typeof this.resolveResult === 'object') {
        payload.source_json = this.resolveResult
      }
      try {
        let response
        if (this.customerId) {
          response = await updateCompany(this.customerId, payload)
          if (!response?.ok) {
            const message = response?.error || '更新客户信息失败'
            if (!silent) {
              ui.pushToast(message, 'error')
            }
            throw new Error(message)
          }
          if (!silent) {
            ui.pushToast('客户信息已更新', 'success')
          }
        } else {
          response = await createCompany(payload)
          if (!response?.ok) {
            const message = response?.error || '客户信息保存失败'
            if (!silent) {
              ui.pushToast(message, 'error')
            }
            throw new Error(message)
          }
          if (!silent) {
            ui.pushToast('客户信息已保存', 'success')
          }
        }
        const responseData = response?.data || {}
        if (!this.customerId && responseData.customer_id) {
          this.customerId = responseData.customer_id
        }
        if (Object.prototype.hasOwnProperty.call(responseData, 'automation_job')) {
          this.setAutomationJob(responseData.automation_job)
        }
        this.website = payload.website
        this.country = payload.country
        this.summary = payload.summary
        const nextStep = advanceStep ? Math.max(this.step, 2) : this.step
        this.step = nextStep
        this.resolveResult = {
          ...(this.resolveResult && typeof this.resolveResult === 'object' ? this.resolveResult : {}),
          customer_id: this.customerId,
          name: payload.name,
          website: payload.website,
          country: payload.country,
          summary: payload.summary,
          contacts: (payload.contacts || []).map((item) => ({ ...item })),
          last_step: nextStep,
          automation_job: this.automationJob,
        }
        return responseData
      } catch (error) {
        if (!silent) {
          ui.pushToast(error.message, 'error')
        }
        throw error
      }
    },
    async queueAutomation(query) {
      const ui = useUiStore()
      const trimmed = (query || '').trim()
      if (!trimmed) {
        return Promise.resolve(null)
      }
      ui.pushToast('添加成功，开始排队分析', 'success')
      return new Promise((resolve, reject) => {
        this.automationBacklog.push({ query: trimmed, resolve, reject })
        this.processAutomationQueue()
      })
    },
    async processAutomationQueue() {
      if (this.automationQueueing) {
        return
      }
      const next = this.automationBacklog.shift()
      if (!next) {
        return
      }
      this.automationQueueing = true
      try {
        const job = await this.executeAutomationForQuery(next.query)
        next.resolve(job)
      } catch (error) {
        next.reject(error)
      } finally {
        this.automationQueueing = false
        if (this.automationBacklog.length) {
          this.processAutomationQueue()
        }
      }
    },
    async executeAutomationForQuery(trimmed) {
      const ui = useUiStore()
      try {
        const payload = await resolveCompany(trimmed)
        const data = payload?.data || {}
        if (data && trimmed && !data.name) {
          data.name = trimmed
        }
        this.applyResolvedData(data)
        this.query = ''
        const existingId = Number(data.customer_id || data.id || 0)
        if (existingId > 0) {
          this.customerId = existingId
        } else {
          this.customerId = null
          this.gradeFinal = null
          this.analysis = null
          this.emailDraft = null
          this.followupId = null
          this.scheduledTask = null
          this.setAutomationJob(null)
          this.step = 1
        }
        const contacts = Array.isArray(data.contacts)
          ? data.contacts.map((contact, index) => ({
              name: contact.name?.trim() || '',
              title: contact.title?.trim() || '',
              email: contact.email?.trim() || '',
              phone: contact.phone?.trim() || '',
              source: contact.source?.trim() || '',
              is_key: index === 0 || Boolean(contact.is_key),
            }))
          : []
        this.contacts = contacts
        const responseData = await this.saveCompany(
          {
            name: (data.name || trimmed).trim() || trimmed,
            website: data.website || '',
            country: data.country || '',
            summary: data.summary || '',
          },
          { silent: true, advanceStep: false }
        )
        let job = responseData?.automation_job || this.automationJob || null
        if (!job && this.customerId) {
          const automationPayload = await enqueueAutomation(this.customerId)
          if (!automationPayload?.ok) {
            throw new Error(automationPayload?.error || '触发自动分析失败')
          }
          job = automationPayload?.data || null
        }
        if (job) {
          this.setAutomationJob(job)
        }
        this.query = ''
        return job
      } catch (error) {
        ui.pushToast(error.message, 'error')
        this.refreshCustomerDetail(true)
        throw error
      }
    },
    async updateContacts(contacts) {
      const ui = useUiStore()
      this.contacts = contacts
      if (!this.customerId) return
      try {
        await replaceContacts(this.customerId, contacts)
        ui.pushToast('联系人已更新', 'success')
      } catch (error) {
        ui.pushToast(error.message, 'error')
      }
    },
    async fetchGrade() {
      if (!this.customerId) return
      const ui = useUiStore()
      this.loading.grade = true
      try {
        const payload = await suggestGrade(this.customerId)
        this.gradeSuggestion = payload.data
      } catch (error) {
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading.grade = false
      }
    },
    async confirmGrade(grade, reason = '') {
      if (!this.customerId) return
      const ui = useUiStore()
      try {
        await confirmGrade(this.customerId, { grade, reason })
        this.gradeFinal = { grade, reason }
        if (grade === 'A') {
          this.step = 3
        } else {
          this.step = 5
          ui.pushToast('非 A 级客户，流程已归档', 'info')
        }
      } catch (error) {
        ui.pushToast(error.message, 'error')
      }
    },
    async fetchAnalysis() {
      if (!this.customerId) return
      const ui = useUiStore()
      this.loading.analysis = true
      try {
        const payload = await generateAnalysis(this.customerId)
        this.analysis = payload.data
        this.step = 3
      } catch (error) {
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading.analysis = false
      }
    },
    async persistAnalysis() {
      if (!this.customerId || !this.analysis) return
      const ui = useUiStore()
      try {
        await updateAnalysis(this.customerId, {
          core_business: this.analysis.core_business,
          pain_points: this.analysis.pain_points,
          my_entry_points: this.analysis.my_entry_points,
          full_report: this.analysis.full_report,
        })
        ui.pushToast('分析内容已更新', 'success')
      } catch (error) {
        ui.pushToast(error.message, 'error')
      }
    },
    async generateEmail() {
      if (!this.customerId) return
      const ui = useUiStore()
      this.loading.email = true
      try {
        const payload = await generateEmailDraft(this.customerId)
        this.emailDraft = payload.data
        this.step = 4
      } catch (error) {
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading.email = false
      }
    },
    async saveInitialFollowup(notes = '') {
      if (!this.customerId || !this.emailDraft) return
      const ui = useUiStore()
      this.loading.followup = true
      try {
        await updateEmailDraft(this.emailDraft.email_id, {
          subject: this.emailDraft.subject,
          body: this.emailDraft.body,
        })
        const payload = await saveFirstFollowup(this.customerId, this.emailDraft.email_id, notes)
        this.followupId = payload.data.followup_id
        this.step = 5
        ui.pushToast('首次跟进记录已保存', 'success')
      } catch (error) {
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading.followup = false
      }
    },
    async createSchedule(options) {
      if (!this.customerId || !this.emailDraft) return
      const ui = useUiStore()
      this.loading.schedule = true
      try {
        const mode = (options?.mode || 'simple').toLowerCase()
        const request = {
          customer_id: this.customerId,
          context_email_id: this.emailDraft.email_id,
          mode,
        }
        if (mode === 'cron') {
          request.cron_expression = options?.cronExpression || ''
        } else {
          const value = Number(options?.delayValue)
          request.delay_value = Number.isFinite(value) && value > 0 ? value : 3
          request.delay_unit = options?.delayUnit || 'days'
        }
        const payload = await scheduleFollowup(request)
        this.scheduledTask = payload.data
        ui.pushToast(scheduleToastMessage(payload.data), 'success')
      } catch (error) {
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading.schedule = false
      }
    },
  },
})
