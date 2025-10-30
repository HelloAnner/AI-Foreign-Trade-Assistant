<template>
  <FlowLayout
    :step="4"
    :total="5"
    title="Step 4: 生成个性化开发信"
    subtitle="生成开发信草稿，确认后保存为首次跟进记录。"
    :progress="80"
  >
    <section class="card">
      <div v-if="flowStore.step < 4" class="placeholder">请先完成切入点分析。</div>
      <div v-else class="content">
        <button class="primary" :disabled="flowStore.loading.email" @click="flowStore.generateEmail">
          {{ flowStore.loading.email ? '生成中…' : flowStore.emailDraft ? '重新生成' : '生成开发信' }}
        </button>

        <transition name="fade">
          <div v-if="flowStore.emailDraft" key="email" class="email-form">
            <label>
              <span>邮件标题</span>
              <input v-model="flowStore.emailDraft.subject" type="text" />
            </label>
            <label>
              <span>邮件正文</span>
              <textarea v-model="flowStore.emailDraft.body" rows="10"></textarea>
            </label>
          </div>
        </transition>
      </div>
    </section>

    <template #footer>
      <div class="footer-actions">
        <button class="ghost" type="button" @click="nav.goPrev?.()">返回上一步</button>
        <button
          class="primary"
          type="button"
          :disabled="flowStore.loading.followup || !flowStore.emailDraft"
          @click="handleSave"
        >
          {{ flowStore.loading.followup ? '保存中…' : '保存并继续' }}
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

const handleSave = async () => {
  await flowStore.saveInitialFollowup()
  flowStore.step = Math.max(flowStore.step, 5)
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

.email-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

label {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

input,
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
