<template>
  <FlowLayout
    :step="5"
    :total="5"
    title="Step 5: 设置自动化邮件跟进"
    subtitle="选择跟进时间，系统将在到期时自动生成并发送邮件。"
    :progress="100"
  >
    <section class="card">
      <div v-if="flowStore.step < 5" class="placeholder">请先保存首次跟进记录。</div>
      <div v-else class="content">
        <div class="quick-buttons">
          <button
            v-for="days in options"
            :key="days"
            class="ghost"
            :disabled="flowStore.loading.schedule"
            @click="schedule(days)"
          >{{ days }} 天后跟进</button>
        </div>
        <div v-if="flowStore.scheduledTask" class="info">
          已设置任务，计划发送时间：{{ formatDate(flowStore.scheduledTask.due_at) }}
        </div>
      </div>
    </section>

    <template #footer>
      <div class="footer-actions">
        <button class="ghost" type="button" @click="nav.goPrev?.()">返回上一步</button>
        <button class="primary" type="button" @click="flowStore.resetFlow">完成并开始下一个</button>
      </div>
    </template>
  </FlowLayout>
</template>

<script setup>
import { inject } from 'vue'
import FlowLayout from './FlowLayout.vue'
import { useFlowStore } from '../../stores/flow'

const flowStore = useFlowStore()
const nav = inject('flowNav', {})
const options = [3, 7, 14]

const schedule = (days) => {
  flowStore.createSchedule(days)
}

const formatDate = (value) => {
  if (!value) return ''
  return new Date(value).toLocaleString()
}
</script>

<style scoped>
.card {
  background: #ffffff;
  border-radius: 24px;
  padding: 32px;
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.08);
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.placeholder {
  padding: 32px;
  border-radius: 18px;
  border: 1px dashed var(--border-subtle);
  text-align: center;
  color: var(--text-tertiary);
}

.content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.quick-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

button {
  border: none;
  border-radius: 12px;
  padding: 12px 24px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

button.primary {
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
}

button.ghost {
  background: rgba(148, 163, 184, 0.16);
  color: var(--text-secondary);
}

.info {
  padding: 16px;
  border-radius: 14px;
  background: var(--surface-muted);
  color: var(--text-secondary);
}

.footer-actions {
  display: flex;
  gap: 12px;
}
</style>
