import { defineStore } from 'pinia'
import { fetchSettings, saveSettings, testLLM, testSMTP, testSearch } from '../api/settings'
import { useUiStore } from './ui'

export const useSettingsStore = defineStore('settings', {
  state: () => ({
    loading: false,
    data: {
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
    },
  }),
  actions: {
    async fetchSettings() {
      const ui = useUiStore()
      this.loading = true
      try {
        const payload = await fetchSettings()
        if (payload.ok) {
          this.data = { ...this.data, ...payload.data }
        } else if (payload.error) {
          ui.pushToast(payload.error, 'error')
        }
      } catch (error) {
        console.error('Failed to load settings', error)
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading = false
      }
    },
    async saveSettings(partial) {
      const ui = useUiStore()
      this.loading = true
      try {
        const payload = await saveSettings({ ...this.data, ...partial })
        if (payload.ok) {
          this.data = { ...this.data, ...partial }
          ui.pushToast('配置已保存', 'success')
        } else if (payload.error) {
          ui.pushToast(payload.error, 'error')
        }
      } catch (error) {
        console.error('Failed to save settings', error)
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading = false
      }
    },
    async testLLM() {
      const ui = useUiStore()
      try {
        const payload = await testLLM()
        if (payload.ok) {
          ui.pushToast(payload.data.message || 'LLM 测试成功', 'success')
        } else {
          ui.pushToast(payload.error || 'LLM 测试失败', 'error')
        }
      } catch (error) {
        ui.pushToast(error.message, 'error')
      }
    },
    async testSMTP() {
      const ui = useUiStore()
      try {
        const payload = await testSMTP()
        if (payload.ok) {
          ui.pushToast('测试邮件已发送，请检查邮箱', 'success')
        } else {
          ui.pushToast(payload.error || 'SMTP 测试失败', 'error')
        }
      } catch (error) {
        ui.pushToast(error.message, 'error')
      }
    },
    async testSearch() {
      const ui = useUiStore()
      try {
        const payload = await testSearch()
        if (payload.ok) {
          ui.pushToast('搜索 API 测试成功', 'success')
        } else {
          ui.pushToast(payload.error || '搜索测试失败', 'error')
        }
      } catch (error) {
        ui.pushToast(error.message, 'error')
      }
    },
  },
})
