<template>
  <FlowLayout
    :step="3"
    :total="5"
    title="Step 3: 生成产品切入点分析"
    subtitle="结合客户业务与我方优势，生成可编辑的切入报告。"
    :progress="60"
  >
    <section class="card">
      <div v-if="flowStore.step < 3" class="placeholder">请先确认客户为 A 级并完成前两步。</div>
      <div v-else class="content">
        <button class="primary" :disabled="flowStore.loading.analysis" @click="flowStore.fetchAnalysis">
          {{ flowStore.loading.analysis ? '生成中…' : flowStore.analysis ? '重新生成' : '生成分析' }}
        </button>

        <transition name="fade">
          <div v-if="flowStore.analysis" key="analysis" class="analysis-form">
            <label>
              <span>核心业务</span>
              <textarea v-model="flowStore.analysis.core_business" rows="3"></textarea>
            </label>
            <label>
              <span>潜在痛点</span>
              <textarea v-model="flowStore.analysis.pain_points" rows="3"></textarea>
            </label>
            <label>
              <span>我方切入点</span>
              <textarea v-model="flowStore.analysis.my_entry_points" rows="3"></textarea>
            </label>
            <label>
              <span>完整报告</span>
              <textarea v-model="flowStore.analysis.full_report" rows="5"></textarea>
            </label>
          </div>
        </transition>
      </div>
    </section>

    <template #footer>
      <div class="footer-actions">
        <button class="ghost" type="button" @click="nav.goPrev?.()">返回上一步</button>
        <button class="primary" type="button" @click="goNext" :disabled="!flowStore.analysis">
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

const goNext = async () => {
  await flowStore.persistAnalysis()
  flowStore.step = Math.max(flowStore.step, 4)
  nav.goNext?.()
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
  min-height: 420px;
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

.analysis-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

label {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

textarea {
  width: 100%;
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-subtle);
  background: var(--surface-muted);
  font-size: 14px;
  resize: vertical;
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
