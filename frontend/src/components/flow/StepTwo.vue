<template>
  <FlowLayout
    :step="2"
    :total="5"
    title="Step 2: AI 辅助客户价值评级"
    subtitle="结合评级规则自动给出 A/B/C 建议，可人工调整结果。"
    :progress="40"
  >
    <section class="card">
      <div v-if="!flowStore.customerId" class="placeholder">请先完成 Step 1 并保存客户信息。</div>
      <div v-else class="content">
        <button class="primary" :disabled="flowStore.loading.grade" @click="flowStore.fetchGrade">
          {{ flowStore.loading.grade ? '评估中…' : flowStore.gradeSuggestion ? '重新评估' : '获取 AI 评级' }}
        </button>

        <transition name="fade">
          <div v-if="flowStore.gradeSuggestion" key="grade" class="result">
            <p class="grade">AI 建议等级：<strong>{{ flowStore.gradeSuggestion.suggested_grade }}</strong></p>
            <p class="reason">理由：{{ flowStore.gradeSuggestion.reason }}</p>
            <div class="actions">
              <button class="primary" type="button" @click="confirm('A')">确认 A 级</button>
              <button class="ghost" type="button" @click="confirm('B')">调整为 B 级</button>
              <button class="ghost" type="button" @click="confirm('C')">调整为 C 级</button>
            </div>
          </div>
        </transition>

        <div v-if="flowStore.gradeFinal" class="status">
          <span>当前等级：{{ flowStore.gradeFinal.grade }}</span>
          <small v-if="flowStore.gradeFinal.reason">理由：{{ flowStore.gradeFinal.reason }}</small>
        </div>
      </div>
    </section>

    <template #footer>
      <div class="footer-actions">
        <button class="ghost" type="button" @click="nav.goPrev?.()">返回上一步</button>
        <button
          class="primary"
          type="button"
          :disabled="!flowStore.gradeFinal || flowStore.gradeFinal.grade === 'C' || flowStore.gradeFinal.grade === 'B'"
          @click="() => nav.goNext?.()"
        >
          保存并继续
        </button>
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

const confirm = (grade) => {
  const reason = flowStore.gradeSuggestion?.reason || ''
  flowStore.confirmGrade(grade, reason)
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
  min-height: 360px;
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

.result {
  background: var(--surface-muted);
  border-radius: 16px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.grade {
  font-size: 20px;
  margin: 0;
}

.reason {
  color: var(--text-secondary);
  margin: 0;
}

.actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.status {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--text-secondary);
}

.footer-actions {
  display: flex;
  gap: 12px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
