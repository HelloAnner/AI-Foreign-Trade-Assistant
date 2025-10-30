import { defineStore } from 'pinia'

let seed = 0

export const useUiStore = defineStore('ui', {
  state: () => ({
    toasts: [],
  }),
  actions: {
    pushToast(message, tone = 'info', timeout = 3000) {
      const id = ++seed
      this.toasts.push({ id, message, tone })
      if (timeout > 0) {
        setTimeout(() => this.dismissToast(id), timeout)
      }
    },
    dismissToast(id) {
      this.toasts = this.toasts.filter((toast) => toast.id !== id)
    },
  },
})
