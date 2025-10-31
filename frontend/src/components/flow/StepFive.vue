<template>
  <FlowLayout
    :step="5"
    :total="5"
    title="Step 5: 设置自动化邮件跟进"
    subtitle="选择合适的跟进时间，AI 会在约定日期自动发送邮件。"
  >
    <section class="schedule-card">
      <h2>设置自动跟进邮件</h2>
      <div class="options">
        <button
          v-for="days in options"
          :key="days"
          type="button"
          :class="['option', { active: isActive(days) }]"
          :disabled="flowStore.loading.schedule || !flowStore.emailDraft"
          @click="schedule(days)"
        >
          {{ days }} 天后跟进
        </button>
      </div>
      <p v-if="!flowStore.emailDraft" class="hint">需先保存开发信草稿才能安排自动跟进。</p>
      <div v-if="flowStore.loading.schedule" class="status pending">
        <div class="spinner"></div>
        <span>正在创建跟进任务…</span>
      </div>
      <div v-else-if="flowStore.scheduledTask" class="status success">
        <span class="material">check_circle</span>
        <span>任务设置成功！AI 将在 {{ formatDate(flowStore.scheduledTask.due_at) }} 自动发送跟进邮件。</span>
      </div>
    </section>

    <template #footer>
      <button class="primary" type="button" @click="handleFinish">完成并开始下一个</button>
    </template>
  </FlowLayout>
</template>

<script setup>
import { inject } from 'vue'
import { useRouter } from 'vue-router'
import FlowLayout from './FlowLayout.vue'
import { useFlowStore } from '../../stores/flow'

const flowStore = useFlowStore()
const nav = inject('flowNav', {})
const router = useRouter()
const options = [3, 7, 14]

const isActive = (days) => {
  if (!flowStore.scheduledTask) return false
  if (!flowStore.scheduledTask.due_at) return false
  const due = new Date(flowStore.scheduledTask.due_at)
  const created = new Date(flowStore.scheduledTask.created_at)
  if (Number.isNaN(due.getTime()) || Number.isNaN(created.getTime())) return false
  const diff = Math.round((due - created) / (1000 * 60 * 60 * 24))
  return diff === days
}

const schedule = (days) => {
  if (!flowStore.emailDraft) return
  flowStore.createSchedule(days)
}

const formatDate = (value) => {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

const handleFinish = () => {
  flowStore.resetFlow()
  nav.goTo?.(0)
  router.push({ name: 'home' })
}
</script>

<style scoped>
.schedule-card {
  background: var(--surface-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
  padding: 40px;
  display: flex;
  flex-direction: column;
  gap: 24px;
  align-items: center;
  text-align: center;
}

.schedule-card h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
}

.options {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  justify-content: center;
}

.option {
  border-radius: var(--radius-full);
  border: 1px solid var(--border-default);
  background: #fff;
  padding: 10px 24px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.option:hover:not(:disabled) {
  border-color: var(--primary-500);
  color: var(--primary-500);
}

.option:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.option.active {
  background: rgba(19, 73, 236, 0.14);
  border-color: var(--primary-500);
  color: var(--primary-500);
}

.hint {
  margin: 0;
  font-size: 13px;
  color: var(--text-tertiary);
}

.status {
  display: flex;
  align-items: center;
  gap: 10px;
  border-radius: 14px;
  padding: 14px 18px;
  font-size: 14px;
  font-weight: 500;
}

.status.pending {
  background: rgba(19, 73, 236, 0.12);
  color: var(--primary-500);
}

.status.success {
  background: rgba(34, 197, 94, 0.16);
  color: #0d7a33;
}

.spinner {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 3px solid rgba(19, 73, 236, 0.15);
  border-top-color: var(--primary-500);
  animation: spin 1s linear infinite;
}

.primary {
  border: none;
  border-radius: var(--radius-full);
  background: var(--primary-500);
  color: #fff;
  padding: 10px 26px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .schedule-card {
    padding: 32px 24px;
  }
}
</style>
