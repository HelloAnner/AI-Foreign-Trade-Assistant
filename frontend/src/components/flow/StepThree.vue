<template>
  <FlowLayout
    :step="3"
    :total="5"
    title="Step 3: 生成产品切入点分析"
    subtitle="AI 根据客户信息生成可编辑的分析报告。"
  >
    <section class="analysis">
      <div v-if="flowStore.loading.analysis" class="analysis__loading">
        <div class="spinner"></div>
        <p>{{ automationActive ? '后台自动化正在生成切入点分析…' : '正在生成切入点分析…' }}</p>
        <small>AI 正在为您深度分析客户需求，请稍候。</small>
      </div>
      <div v-else-if="!flowStore.analysis" class="analysis__empty">
        <p>暂无分析内容，点击下方按钮生成。</p>
        <button type="button" class="primary" @click="generateAnalysis">生成分析</button>
      </div>
      <div v-else class="analysis__body">
        <header>
          <h2>AI 产品切入点建议</h2>
          <button type="button" class="ghost" @click="toggleEditing">
            <span class="material">edit</span>
            {{ editing ? '完成编辑' : '编辑' }}
          </button>
        </header>
        <div class="section">
          <span>客户核心业务</span>
          <textarea v-if="editing" v-model="flowStore.analysis.core_business" rows="3"></textarea>
          <div v-else class="text-block">{{ flowStore.analysis.core_business }}</div>
        </div>
        <div class="section">
          <span>潜在需求 / 痛点</span>
          <textarea v-if="editing" v-model="flowStore.analysis.pain_points" rows="4"></textarea>
          <div v-else class="text-block">{{ flowStore.analysis.pain_points }}</div>
        </div>
        <div class="section">
          <span>我方切入点</span>
          <textarea v-if="editing" v-model="flowStore.analysis.my_entry_points" rows="4"></textarea>
          <div v-else class="text-block">{{ flowStore.analysis.my_entry_points }}</div>
        </div>
        <div class="section">
          <span>完整报告</span>
          <textarea v-if="editing" v-model="flowStore.analysis.full_report" rows="6"></textarea>
          <div v-else class="text-block">{{ flowStore.analysis.full_report }}</div>
        </div>
      </div>
    </section>

    <template #footer>
      <button class="ghost" type="button" @click="nav.goPrev?.()">返回上一步</button>
      <button
        class="primary"
        type="button"
        :disabled="!flowStore.analysis || flowStore.loading.analysis || automationActive"
        @click="handleNext"
      >
        保存并继续
      </button>
    </template>
  </FlowLayout>
</template>

<script setup>
import { inject, onMounted, ref, computed, watch } from 'vue'
import FlowLayout from './FlowLayout.vue'
import { useFlowStore } from '../../stores/flow'

const flowStore = useFlowStore()
const nav = inject('flowNav', {})
const editing = ref(false)

const automationActive = computed(() => {
  const status = String(flowStore.automationJob?.status || '').toLowerCase()
  return status === 'queued' || status === 'running'
})

const ensureAnalysis = () => {
  if (!flowStore.customerId || flowStore.loading.analysis || automationActive.value) return
  if (!flowStore.analysis) {
    flowStore.fetchAnalysis()
  }
}

onMounted(() => {
  ensureAnalysis()
})

watch(
  () => automationActive.value,
  (value) => {
    if (!value) {
      ensureAnalysis()
    }
  }
)

const generateAnalysis = () => {
  ensureAnalysis()
}

const toggleEditing = () => {
  editing.value = !editing.value
}

const handleNext = async () => {
  if (!flowStore.analysis) return
  await flowStore.persistAnalysis()
  flowStore.step = Math.max(flowStore.step, 4)
  nav.goNext?.()
}
</script>

<style scoped>
.analysis {
  background: var(--surface-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
  padding: 32px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.analysis__loading,
.analysis__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--text-secondary);
  text-align: center;
}

.analysis__body header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.analysis__body h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section span {
  font-size: 14px;
  color: var(--text-secondary);
}

textarea {
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-default);
  background: #fff;
  font-size: 14px;
  resize: vertical;
}

.text-block {
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--surface-subtle);
  white-space: pre-wrap;
  text-align: left;
}

.spinner {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 4px solid rgba(19, 73, 236, 0.15);
  border-top-color: var(--primary-500);
  animation: spin 1s linear infinite;
}

.ghost,
.primary {
  border-radius: var(--radius-full);
  padding: 10px 22px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.ghost {
  border: 1px solid var(--border-default);
  background: #fff;
  color: var(--text-secondary);
}

.ghost:hover {
  border-color: var(--primary-500);
  color: var(--primary-500);
}

.primary {
  border: none;
  background: var(--primary-500);
  color: #fff;
}

.primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .analysis {
    padding: 24px;
  }
}
</style>
