<template>
  <FlowLayout
    :step="4"
    :total="5"
    title="Step 4: 生成个性化开发信"
    subtitle="确认并微调邮件内容，保存为首次跟进记录。"
  >
    <section class="mail-card">
      <div v-if="flowStore.loading.email" class="mail-card__loading">
        <div class="spinner"></div>
        <p>{{ automationActive ? '后台自动化正在生成个性化开发信…' : '正在生成个性化开发信…' }}</p>
      </div>
      <div v-else-if="!flowStore.emailDraft" class="mail-card__empty">
        <p>尚未生成开发信草稿，点击下方按钮立即生成。</p>
        <button type="button" class="primary" @click="generateEmail">生成开发信</button>
      </div>
      <div v-else class="mail-card__body">
        <label>
          <span>邮件标题</span>
          <input v-model="flowStore.emailDraft.subject" type="text" />
        </label>
        <label>
          <span>邮件正文</span>
          <textarea v-model="flowStore.emailDraft.body" rows="12"></textarea>
        </label>
        <label>
          <span>首次跟进备注（选填）</span>
          <textarea v-model="followupNotes" rows="4" placeholder="记录与客户沟通的背景或期待的行动"></textarea>
        </label>
      </div>
    </section>

    <template #footer>
      <button
        class="primary"
        type="button"
        :disabled="!flowStore.emailDraft || flowStore.loading.followup || flowStore.loading.email || automationActive"
        @click="handleSave"
      >
        {{ flowStore.loading.followup ? '保存中…' : '保存为首次跟进记录' }}
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
const followupNotes = ref('')

const automationActive = computed(() => {
  const status = String(flowStore.automationJob?.status || '').toLowerCase()
  return status === 'queued' || status === 'running'
})

const ensureEmailDraft = () => {
  if (!flowStore.customerId || flowStore.loading.email || automationActive.value) return
  if (!flowStore.emailDraft) {
    flowStore.generateEmail()
  }
}

onMounted(() => {
  ensureEmailDraft()
})

watch(
  () => automationActive.value,
  (value) => {
    if (!value) {
      ensureEmailDraft()
    }
  }
)

const generateEmail = () => {
  ensureEmailDraft()
}

const handleSave = async () => {
  if (!flowStore.emailDraft) return
  await flowStore.saveInitialFollowup(followupNotes.value)
  nav.goNext?.()
}
</script>

<style scoped>
.mail-card {
  background: var(--surface-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
  padding: 32px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.mail-card__loading,
.mail-card__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-align: center;
  color: var(--text-secondary);
}

.mail-card__body {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
  color: var(--text-secondary);
}

input,
textarea {
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-default);
  background: #fff;
  font-size: 14px;
}

textarea {
  resize: vertical;
}

.spinner {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 4px solid rgba(19, 73, 236, 0.15);
  border-top-color: var(--primary-500);
  animation: spin 1s linear infinite;
}

.primary {
  border: none;
  border-radius: var(--radius-full);
  background: var(--primary-500);
  color: #fff;
  padding: 10px 24px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
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
  .mail-card {
    padding: 24px;
  }
}
</style>
