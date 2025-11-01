import { defineStore } from 'pinia'
import {
  listCustomers,
  getCustomerDetail,
  deleteCustomer as deleteCustomerRequest,
  triggerAutomation,
} from '../api/customers'
import { useUiStore } from './ui'

export const useCustomersStore = defineStore('customers', {
  state: () => ({
    loading: false,
    items: [],
    total: 0,
    page: 1,
    pageSize: 8,
    filters: {
      grade: '',
      country: '',
      status: '',
      sort: 'created_desc',
      q: '',
    },
    detailLoading: false,
    detail: null,
  }),
  actions: {
    async fetchList(extra = {}, options = {}) {
      const ui = useUiStore()
      this.loading = true
      try {
        const { skipAdjust = false } = options || {}
        const params = { ...this.filters, ...extra }
        if (!params.q) delete params.q

        let limit = Number(params.limit ?? this.pageSize)
        if (!Number.isFinite(limit) || limit <= 0) {
          limit = this.pageSize || 8
        }
        params.limit = limit

        let offset = Number(params.offset ?? (this.page - 1) * limit)
        if (!Number.isFinite(offset) || offset < 0) {
          offset = 0
        }
        params.offset = offset

        const payload = await listCustomers(params)
        if (payload.ok) {
          const data = payload.data || {}
          this.items = Array.isArray(data.items) ? data.items : []
          this.total = Number.isFinite(data.total) ? data.total : 0
          const serverLimit = Number(data.limit ?? limit)
          this.pageSize = serverLimit > 0 ? serverLimit : limit
          const serverOffset = Number(data.offset ?? offset)
          this.page = Math.floor(serverOffset / this.pageSize) + 1
          if (this.total === 0) {
            this.page = 1
          }

          if (!skipAdjust && this.total > 0 && serverOffset >= this.total && this.page > 1) {
            const maxPage = Math.max(1, Math.ceil(this.total / this.pageSize))
            if (this.page > maxPage) {
              this.page = maxPage
              await this.fetchList(
                {
                  ...extra,
                  limit: this.pageSize,
                  offset: (this.page - 1) * this.pageSize,
                },
                { skipAdjust: true }
              )
              return
            }
          }
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
        this.page = 1
      }
    },
    async setPage(page) {
      const target = Number.isInteger(page) ? page : 1
      const normalized = Math.max(1, target)
      if (normalized === this.page) return
      this.page = normalized
      await this.fetchList()
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
    async removeCustomer(customerId) {
      const ui = useUiStore()
      if (!customerId) {
        ui.pushToast('无效的客户 ID', 'error')
        return false
      }
      try {
        const payload = await deleteCustomerRequest(customerId)
        if (!payload.ok) {
          ui.pushToast(payload.error || '删除客户失败', 'error')
          return false
        }
        if (this.detail && this.detail.id === customerId) {
          this.detail = null
        }
        await this.fetchList()
        ui.pushToast('客户已删除', 'success')
        return true
      } catch (error) {
        console.error('Failed to delete customer', error)
        ui.pushToast(error.message, 'error')
        return false
      }
    },
    async rerunAutomation(customerId) {
      const ui = useUiStore()
      if (!customerId) {
        ui.pushToast('无效的客户 ID', 'error')
        return null
      }
      try {
        const payload = await triggerAutomation(customerId)
        if (!payload.ok) {
          ui.pushToast(payload.error || '触发自动分析失败', 'error')
          return null
        }
        ui.pushToast('后台开始自动分析', 'success')
        if (this.detail && this.detail.id === customerId) {
          this.detail = {
            ...this.detail,
            automation_job: payload.data || null,
          }
        }
        return payload.data || null
      } catch (error) {
        console.error('Failed to rerun automation', error)
        ui.pushToast(error.message, 'error')
        return null
      }
    },
  },
})
