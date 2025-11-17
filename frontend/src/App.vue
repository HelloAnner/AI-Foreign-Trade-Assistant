<template>
  <div class="app-shell">
    <div class="app-content">
      <RouterView />
    </div>
    <AppFooter />
    <transition-group name="toast" tag="div" class="toast-stack">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="toast"
        :class="`toast--${toast.tone}`"
      >
        {{ toast.message }}
      </div>
    </transition-group>
  </div>
</template>

<script setup>
import { storeToRefs } from 'pinia'
import { useUiStore } from './stores/ui'
import AppFooter from './components/AppFooter.vue'

const uiStore = useUiStore()
const { toasts } = storeToRefs(uiStore)
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.app-content {
  flex: 1 0 auto;
}

.toast-stack {
  position: fixed;
  top: 24px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  gap: 12px;
  z-index: 60;
  align-items: center;
}

.toast {
  min-width: 260px;
  padding: 12px 16px;
  border-radius: 12px;
  font-size: 14px;
  background: rgba(15, 23, 42, 0.92);
  color: #fff;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.25);
}

.toast--success {
  background: linear-gradient(135deg, #22c55e, #16a34a);
}

.toast--error {
  background: linear-gradient(135deg, #ef4444, #dc2626);
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.25s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-12px);
}
</style>
