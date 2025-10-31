<template>
  <FlowLayout
    :step="2"
    :total="5"
    title="Step 2: AI 辅助客户价值评级"
    subtitle="系统根据客户画像自动给出评级，确认后进入下一步。"
  >
    <section class="rating-card">
      <div v-if="!flowStore.customerId" class="rating-card__placeholder">请先完成 Step 1 并保存客户信息。</div>
      <div v-else class="rating-card__content">
        <div v-if="flowStore.loading.grade || !flowStore.gradeSuggestion" class="rating-card__spinner">
          <div class="spinner"></div>
          <p>AI 正在进行价值评估…</p>
        </div>
        <template v-else>
          <p class="label">AI 推荐评级</p>
          <div class="badge" :class="`badge--${flowStore.gradeSuggestion.suggested_grade.toLowerCase()}`">
            {{ flowStore.gradeSuggestion.suggested_grade }}级
          </div>
          <p class="reason">理由：{{ flowStore.gradeSuggestion.reason }}</p>
          <div class="actions">
            <button type="button" class="solid" @click="handleConfirm('A')">确认 A级（继续分析）</button>
            <button type="button" class="outline" @click="handleConfirm('B')">调整为 B级（归档）</button>
            <button type="button" class="outline" @click="handleConfirm('C')">调整为 C级（忽略）</button>
          </div>
        </template>
        <div v-if="flowStore.gradeFinal" class="status">
          <span>当前等级：{{ flowStore.gradeFinal.grade }}</span>
          <small v-if="flowStore.gradeFinal.reason">理由：{{ flowStore.gradeFinal.reason }}</small>
        </div>
      </div>
    </section>
  </FlowLayout>
</template>

<script setup>
import { inject, onMounted, watch } from 'vue'
import FlowLayout from './FlowLayout.vue'
import { useFlowStore } from '../../stores/flow'

const flowStore = useFlowStore()
const nav = inject('flowNav', {})

const ensureGrade = () => {
  if (!flowStore.customerId || flowStore.loading.grade) return
  if (!flowStore.gradeSuggestion) {
    flowStore.fetchGrade()
  }
}

onMounted(() => {
  ensureGrade()
})

watch(
  () => flowStore.customerId,
  () => {
    ensureGrade()
  }
)

const handleConfirm = async (grade) => {
  if (!flowStore.customerId) return
  const reason = flowStore.gradeSuggestion?.reason || ''
  await flowStore.confirmGrade(grade, reason)
  if (grade === 'A') {
    nav.goNext?.()
  } else {
    nav.goTo?.(4) // 归档或忽略直接跳到最后一步
  }
}
</script>

<style scoped>
.rating-card {
  background: var(--surface-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  padding: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  text-align: center;
}

.rating-card__placeholder {
  color: var(--text-tertiary);
  font-size: 15px;
}

.rating-card__content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.rating-card__spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--text-secondary);
}

.spinner {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 4px solid rgba(19, 73, 236, 0.15);
  border-top-color: var(--primary-500);
  animation: spin 1s linear infinite;
}

.label {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.badge {
  min-width: 120px;
  padding: 12px 24px;
  border-radius: 999px;
  font-size: 24px;
  font-weight: 700;
}

.badge--s {
  background: rgba(202, 138, 4, 0.14);
  color: #92400e;
}

.badge--a {
  background: rgba(59, 130, 246, 0.14);
  color: #1d4ed8;
}

.badge--b {
  background: rgba(99, 102, 241, 0.14);
  color: #4338ca;
}

.badge--c {
  background: rgba(148, 163, 184, 0.18);
  color: #475569;
}

.reason {
  margin: 0;
  max-width: 420px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: center;
}

.solid,
.outline {
  border-radius: var(--radius-full);
  padding: 10px 22px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.solid {
  border: none;
  background: var(--primary-500);
  color: #fff;
}

.solid:hover {
  background: var(--primary-600);
}

.outline {
  border: 1px solid var(--border-default);
  background: #fff;
  color: var(--text-secondary);
}

.outline:hover {
  border-color: var(--primary-500);
  color: var(--primary-500);
}

.status {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--text-secondary);
  font-size: 14px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .rating-card {
    padding: 32px 24px;
  }
}
</style>
