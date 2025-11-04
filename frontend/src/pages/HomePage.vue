<template>
  <FlowLayout :step="0" :total="0">
    <section class="home">
      <h2>智能客户信息获取</h2>
      <p>支持公司中文/英文全称或官网地址，系统会自动匹配并拉取公开信息。</p>
      <form class="home__form" @submit.prevent="handleSubmit">
        <input
          v-model="queryInput"
          :disabled="!automationEnabled && flowStore.resolving"
          @keydown.enter.prevent="handleSubmit"
          type="text"
          placeholder="请输入公司名称（支持逗号分隔批量添加：如 大众, AWS）"
        />
        <button type="submit" :disabled="!queryInput || (!automationEnabled && flowStore.resolving)">
          {{ submitLabel }}
        </button>
      </form>
    </section>
  </FlowLayout>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import FlowLayout from '../components/flow/FlowLayout.vue'
import { useRouter } from 'vue-router'
import { useFlowStore } from '../stores/flow'
import { useSettingsStore } from '../stores/settings'
import { useUiStore } from '../stores/ui'

const router = useRouter()
const flowStore = useFlowStore()
const settingsStore = useSettingsStore()

const queryInput = ref(flowStore.query)

const automationEnabled = computed(() => Boolean(settingsStore.data.automation_enabled))

const submitLabel = computed(() => {
  if (automationEnabled.value) {
    return '开始后台自动分析'
  }
  return flowStore.resolving ? '分析中…' : '开始分析'
})

watch(
  () => flowStore.query,
  (value) => {
    if (automationEnabled.value) {
      return
    }
    if (value !== queryInput.value) {
      queryInput.value = value
    }
  }
)

watch(
  () => automationEnabled.value,
  (enabled) => {
    if (!enabled) {
      queryInput.value = flowStore.query
    }
  }
)

onMounted(() => {
  settingsStore.fetchSettings()
})

const handleSubmit = async () => {
  const trimmed = (queryInput.value || '').trim()
  if (!trimmed) return
  if (!automationEnabled.value && flowStore.resolving) return

  if (!settingsStore.loaded && !settingsStore.loading) {
    await settingsStore.fetchSettings()
  }

  if (automationEnabled.value) {
    // 自动化模式：支持以逗号/中文逗号/分号/换行分隔的批量入队
    const items = (trimmed || '')
      .split(/[，,；;\n]+/)
      .map((s) => s.trim())
      .filter(Boolean)

    const ui = useUiStore()
    try {
      const { enqueueTodo } = await import('../api/flow')
      if (items.length <= 1) {
        const resp = await enqueueTodo(items[0] || trimmed)
        if (resp?.ok) ui.pushToast('已加入任务队列，后台处理中', 'success')
      } else {
        const results = await Promise.allSettled(items.map((q) => enqueueTodo(q)))
        const ok = results.filter((r) => r.status === 'fulfilled' && r.value?.ok).length
        const fail = results.length - ok
        if (ok) ui.pushToast(`已入队 ${ok} 条任务`, 'success')
        if (fail) ui.pushToast(`${fail} 条任务入队失败`, 'error')
      }
    } catch (err) {
      ui.pushToast(err.message || '入队失败', 'error')
    }
    queryInput.value = ''
    return
  }

  await flowStore.startResolve(trimmed)
  if (!flowStore.resolving) {
    router.push({ name: 'flow' })
  }
}
</script>

<style scoped>
.home {
  margin: 80px auto 0;
  width: 66.6667%;
  max-width: 1200px;
  background: var(--surface-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  padding: 48px 56px;
  display: flex;
  flex-direction: column;
  gap: 18px;
  text-align: center;
}

.home h2 {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
}

.home p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 15px;
}

.home__form {
  display: flex;
  gap: 16px;
  margin-top: 8px;
}

.home__form input {
  flex: 1;
  padding: 14px 18px;
  border-radius: 16px;
  border: 1px solid var(--border-default);
  background: #fff;
  font-size: 15px;
}

.home__form input:focus {
  outline: none;
  border-color: var(--primary-500);
  box-shadow: 0 0 0 4px rgba(19, 73, 236, 0.14);
}

.home__form button {
  min-width: 160px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--primary-500);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  padding: 12px 28px;
  cursor: pointer;
  transition: background 0.2s ease;
}

.home__form button:hover:not(:disabled) {
  background: var(--primary-600);
}

.home__form button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .home {
    margin-top: 40px;
    width: 100%;
    padding: 32px 24px;
  }

  .home__form {
    flex-direction: column;
  }

  .home__form button {
    width: 100%;
  }
}
</style>
