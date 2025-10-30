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
} from '../api/flow'
import { useUiStore } from './ui'

export const useFlowStore = defineStore('flow', {
  state: () => ({
    step: 1,
    query: '',
    resolving: false,
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
      this.step = 1
      this.query = ''
      this.resolving = false
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
      this.loading = {
        grade: false,
        analysis: false,
        email: false,
        followup: false,
        schedule: false,
      }
    },
  async startResolve(query) {
    const ui = useUiStore()
    this.resolving = true
    this.query = query
    try {
      const payload = await resolveCompany(query)
      const data = payload.data || {}
      this.resolveResult = data
      this.contacts = (data.contacts || []).map((item) => ({ ...item }))
      this.summary = data.summary || ''
      this.country = data.country || ''
      this.website = data.website || ''

      if (data.customer_id) {
        this.customerId = data.customer_id
        const normalizedGrade = (data.grade || '').toUpperCase()
        if (normalizedGrade && normalizedGrade !== 'UNKNOWN') {
          this.gradeFinal = {
            grade: normalizedGrade,
            reason: data.grade_reason || '',
          }
        } else {
          this.gradeFinal = null
        }
        this.analysis = data.analysis ? { ...data.analysis } : null
        this.emailDraft = data.email_draft ? { ...data.email_draft } : null
        this.followupId = data.followup_id || null
        this.scheduledTask = data.scheduled_task || null
        this.step = Math.max(1, data.last_step || 1)
      } else {
        this.customerId = null
        this.gradeFinal = null
        this.analysis = null
        this.emailDraft = null
        this.followupId = null
        this.scheduledTask = null
        this.step = 1
      }
      this.gradeSuggestion = null
    } catch (error) {
      ui.pushToast(error.message, 'error')
    } finally {
      this.resolving = false
    }
    },
  async saveCompany(company) {
    const ui = useUiStore()
    const payload = {
      name: company.name,
      website: company.website || this.website,
      country: company.country || this.country,
      summary: company.summary || this.summary,
      contacts: this.contacts,
      source_json: this.resolveResult || {},
    }
    try {
      if (this.customerId) {
        await updateCompany(this.customerId, payload)
        ui.pushToast('客户信息已更新', 'success')
      } else {
        const response = await createCompany(payload)
        this.customerId = response.data.customer_id
        ui.pushToast('客户信息已保存', 'success')
      }
      this.website = payload.website
      this.country = payload.country
      this.summary = payload.summary
      this.step = Math.max(this.step, 2)
      this.resolveResult = {
        ...(this.resolveResult && typeof this.resolveResult === 'object' ? this.resolveResult : {}),
        customer_id: this.customerId,
        name: payload.name,
        website: payload.website,
        country: payload.country,
        summary: payload.summary,
        contacts: (payload.contacts || []).map((item) => ({ ...item })),
        last_step: Math.max(this.step, 2),
      }
    } catch (error) {
      ui.pushToast(error.message, 'error')
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
    async createSchedule(delayDays) {
      if (!this.customerId || !this.emailDraft) return
      const ui = useUiStore()
      this.loading.schedule = true
      try {
        const payload = await scheduleFollowup({
          customer_id: this.customerId,
          context_email_id: this.emailDraft.email_id,
          delay_days: delayDays,
        })
        this.scheduledTask = payload.data
        ui.pushToast(`已设置 ${delayDays} 天后的自动跟进`, 'success')
      } catch (error) {
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading.schedule = false
      }
    },
  },
})
