<template>
  <FlowLayout :step="0" :total="0" title="AI 外贸客户开发助手" subtitle="输入潜在客户公司名称或官网，AI 将自动完成信息获取。">
    <section class="home">
      <h2>智能客户信息获取</h2>
      <p>支持公司中文/英文全称或官网地址，系统会自动匹配并拉取公开信息。</p>
      <form class="home__form" @submit.prevent="handleSubmit">
        <input
          v-model="queryInput"
          :disabled="!automationEnabled && flowStore.resolving"
          @keydown.enter.prevent="handleSubmit"
          type="text"
          placeholder="例如：环球贸易有限公司 或 https://www.example.com"
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
    // 自动化模式：将任务入队到后端 TODO 队列，后台线程统一处理
    try {
      const { enqueueTodo } = await import('../api/flow')
      const resp = await enqueueTodo(trimmed)
      const ui = useUiStore()
      if (resp?.ok) ui.pushToast('已加入任务队列，后台处理中', 'success')
    } catch (err) {
      const ui = useUiStore()
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
  max-width: 640px;
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
