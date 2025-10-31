<template>
  <FlowLayout
    :step="5"
    :total="5"
    title="Step 5: 设置自动化邮件跟进"
    subtitle="选择合适的跟进时间，AI 会在约定日期自动发送邮件。"
  >
    <section class="schedule-card">
      <h2>设置自动跟进邮件</h2>
      <p class="subtitle">选择触发方式，AI 会在设定时间自动发送跟进邮件。</p>

      <div class="mode-switch">
        <button
          type="button"
          :class="['mode-option', { active: mode === 'simple' }]"
          @click="mode = 'simple'"
        >
          简单延迟
        </button>
        <button
          type="button"
          :class="['mode-option', { active: mode === 'cron' }]"
          @click="mode = 'cron'"
        >
          Cron 表达式
        </button>
      </div>

      <div v-if="mode === 'simple'" class="simple-config">
        <div class="simple-inputs">
          <label>
            <span>延迟时长</span>
            <input
              v-model.number="simpleValue"
              type="number"
              min="1"
              step="1"
              :disabled="flowStore.loading.schedule"
            />
          </label>
          <label>
            <span>时间单位</span>
            <select v-model="simpleUnit" :disabled="flowStore.loading.schedule">
              <option value="minutes">分钟</option>
              <option value="hours">小时</option>
              <option value="days">天</option>
            </select>
          </label>
        </div>
        <div class="quick-picks">
          <span>快捷选择：</span>
          <button type="button" class="chip" @click="applyQuick(30, 'minutes')">30 分钟</button>
          <button type="button" class="chip" @click="applyQuick(4, 'hours')">4 小时</button>
          <button type="button" class="chip" @click="applyQuick(3, 'days')">3 天</button>
          <button type="button" class="chip" @click="applyQuick(7, 'days')">7 天</button>
        </div>
      </div>

      <div v-else class="cron-config">
        <label>
          <span>Cron 表达式</span>
          <input
            v-model="cronExpression"
            type="text"
            placeholder="如：0 9 * * MON"
            :disabled="flowStore.loading.schedule"
          />
        </label>
        <p class="hint">
          使用标准 5/6 位 cron 表达式，支持 <code>@daily</code>、<code>@weekly</code> 等描述。
        </p>
      </div>

      <p v-if="!flowStore.emailDraft" class="hint">需先保存开发信草稿才能安排自动跟进。</p>

      <div class="actions">
        <button
          class="primary"
          type="button"
          :disabled="disableSchedule"
          @click="handleSchedule"
        >
          {{ flowStore.loading.schedule ? '正在创建…' : '保存自动跟进' }}
        </button>
      </div>

      <div v-if="flowStore.loading.schedule" class="status pending">
        <div class="spinner"></div>
        <span>正在创建跟进任务…</span>
      </div>
      <div v-else-if="flowStore.scheduledTask" class="status success">
        <span class="material">check_circle</span>
        <span>{{ scheduledSummary }}</span>
      </div>
    </section>

    <template #footer>
      <button class="primary" type="button" @click="handleFinish">完成并开始下一个</button>
    </template>
  </FlowLayout>
</template>

<script setup>
import { computed, inject, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import FlowLayout from './FlowLayout.vue'
import { useFlowStore } from '../../stores/flow'

const flowStore = useFlowStore()
const nav = inject('flowNav', {})
const router = useRouter()
const mode = ref('simple')
const simpleValue = ref(3)
const simpleUnit = ref('days')
const cronExpression = ref('')

watch(
  () => flowStore.scheduledTask,
  (task) => {
    if (!task) {
      mode.value = 'simple'
      simpleValue.value = 3
      simpleUnit.value = 'days'
      cronExpression.value = ''
      return
    }
    if (task.mode === 'cron') {
      mode.value = 'cron'
      cronExpression.value = task.cron_expression || ''
      simpleValue.value = 3
      simpleUnit.value = 'days'
    } else {
      mode.value = 'simple'
      simpleValue.value = task.delay_value || 3
      simpleUnit.value = task.delay_unit || 'days'
      cronExpression.value = task.cron_expression || ''
    }
  },
  { immediate: true }
)

const disableSchedule = computed(() => {
  if (!flowStore.emailDraft || flowStore.loading.schedule) return true
  if (mode.value === 'cron') {
    return !cronExpression.value.trim()
  }
  return !simpleValue.value || Number(simpleValue.value) <= 0
})

const scheduledSummary = computed(() => {
  const task = flowStore.scheduledTask
  if (!task) return ''
  const due = formatDate(task.due_at)
  if (task.mode === 'cron' && task.cron_expression) {
    return `已设置 Cron：${task.cron_expression}，下次执行时间 ${due}`
  }
  if (task.delay_value && task.delay_unit) {
    return `已设置在 ${task.delay_value}${unitLabel(task.delay_unit)} 后跟进，预计于 ${due} 触发`
  }
  return `AI 将在 ${due} 自动发送跟进邮件。`
})

const applyQuick = (value, unit) => {
  if (flowStore.loading.schedule) return
  mode.value = 'simple'
  simpleValue.value = value
  simpleUnit.value = unit
}

const handleSchedule = () => {
  if (!flowStore.emailDraft || flowStore.loading.schedule) return
  if (mode.value === 'cron') {
    flowStore.createSchedule({
      mode: 'cron',
      cronExpression: cronExpression.value.trim(),
    })
  } else {
    flowStore.createSchedule({
      mode: 'simple',
      delayValue: Number(simpleValue.value),
      delayUnit: simpleUnit.value,
    })
  }
}

const formatDate = (value) => {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

const unitLabel = (unit) => {
  switch (unit) {
    case 'minutes':
      return ' 分钟'
    case 'hours':
      return ' 小时'
    case 'days':
      return ' 天'
    default:
      return ''
  }
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

.subtitle {
  margin: 4px 0 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.mode-switch {
  display: inline-flex;
  padding: 4px;
  border-radius: var(--radius-full);
  background: rgba(15, 23, 42, 0.04);
  gap: 6px;
  margin: 8px 0 16px;
}

.mode-option {
  border: none;
  border-radius: var(--radius-full);
  padding: 8px 18px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mode-option.active {
  background: #fff;
  color: var(--primary-500);
  box-shadow: 0 6px 16px rgba(19, 73, 236, 0.1);
}

.simple-config,
.cron-config {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.simple-inputs {
  display: flex;
  gap: 20px;
  align-items: flex-end;
  flex-wrap: wrap;
  justify-content: center;
}

.simple-inputs label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
  color: var(--text-secondary);
}

.simple-inputs input,
.simple-inputs select,
.cron-config input {
  padding: 12px 16px;
  border-radius: 12px;
  border: 1px solid var(--border-default);
  font-size: 14px;
  min-width: 120px;
}

.quick-picks {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: center;
  font-size: 13px;
  color: var(--text-secondary);
}

.quick-picks .chip {
  border: none;
  border-radius: var(--radius-full);
  padding: 6px 14px;
  background: rgba(19, 73, 236, 0.12);
  color: var(--primary-500);
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s ease;
}

.quick-picks .chip:hover {
  background: rgba(19, 73, 236, 0.18);
}

.cron-config label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
  color: var(--text-secondary);
}

.cron-config code {
  background: rgba(15, 23, 42, 0.08);
  padding: 0 6px;
  border-radius: 6px;
}

.actions {
  display: flex;
  justify-content: center;
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

  .simple-inputs {
    gap: 12px;
  }
}
</style>
