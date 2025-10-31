import { defineStore } from 'pinia'
import { listCustomers, getCustomerDetail } from '../api/customers'
import { useUiStore } from './ui'

export const useCustomersStore = defineStore('customers', {
  state: () => ({
    loading: false,
    items: [],
    filters: {
      grade: '',
      country: '',
      status: '',
      sort: 'last_followup_desc',
      q: '',
    },
    detailLoading: false,
    detail: null,
  }),
  actions: {
    async fetchList(extra = {}) {
      const ui = useUiStore()
      this.loading = true
      try {
        const params = {
          ...this.filters,
          ...extra,
        }
        if (!params.q) delete params.q
        const payload = await listCustomers(params)
        if (payload.ok) {
          this.items = payload.data || []
        } else {
          ui.pushToast(payload.error || '加载客户列表失败', 'error')
        }
      } catch (error) {
        console.error('Failed to load customers', error)
        ui.pushToast(error.message, 'error')
      } finally {
        this.loading = false
      }
    },
    setFilter(key, value) {
      if (Object.prototype.hasOwnProperty.call(this.filters, key)) {
        this.filters[key] = value
      }
    },
    async fetchDetail(customerId) {
      const ui = useUiStore()
      this.detailLoading = true
      try {
        const payload = await getCustomerDetail(customerId)
        if (payload.ok) {
          this.detail = payload.data
        } else {
          ui.pushToast(payload.error || '加载客户详情失败', 'error')
        }
      } catch (error) {
        console.error('Failed to load customer detail', error)
        ui.pushToast(error.message, 'error')
      } finally {
        this.detailLoading = false
      }
    },
    clearDetail() {
      this.detail = null
    },
  },
})
