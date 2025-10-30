<template>
  <div class="flow-layout">
    <header class="flow-header">
      <div class="flow-header__title">AI 外贸客户开发助手</div>
      <button class="flow-header__settings" @click="goSettings">设置</button>
    </header>
    <div class="flow-progress">流程进度：第 {{ flowStore.step }} 步 / 5</div>
    <main class="flow-content">
      <section class="step-card">
        <div class="step-card__head">
          <div>
            <h2 class="step-card__title">Step 1 · 智能信息获取与聚合</h2>
            <p class="step-card__description">输入客户公司名或官网地址，系统自动聚合基础信息与联系人。</p>
          </div>
          <span class="step-card__badge">必填</span>
        </div>
        <div class="step-card__body">
          <div class="form-row">
            <input
              v-model="queryInput"
              :disabled="flowStore.resolving"
              type="text"
              placeholder="输入客户公司全名或官网地址"
            />
            <button
              class="primary"
              :disabled="flowStore.resolving || !queryInput"
              @click="handleResolve"
            >
              {{ flowStore.resolving ? '分析中...' : '开始分析' }}
            </button>
          </div>
          <div v-if="flowStore.resolveResult" class="result-block">
            <div class="field-grid">
              <label>
                <span>公司名称</span>
                <input v-model="companyForm.name" type="text" placeholder="如：Global Tech Inc." />
              </label>
              <label>
                <span>官网</span>
                <input v-model="companyForm.website" type="text" placeholder="https://" />
              </label>
              <label>
                <span>国家/地区</span>
                <input v-model="companyForm.country" type="text" placeholder="United States" />
              </label>
            </div>
            <label>
              <span>AI 摘要</span>
              <textarea v-model="companyForm.summary" rows="4" placeholder="公司业务概览"></textarea>
            </label>
            <div class="contacts-header">
              <div>
                <span>潜在联系人</span>
                <small>可编辑，新增重要联系人时勾选“重点”</small>
              </div>
              <button class="ghost" @click="addContact">新增联系人</button>
            </div>
            <div v-if="contactsLocal.length" class="contacts-list">
              <div
                v-for="(contact, index) in contactsLocal"
                :key="index"
                class="contact-row"
              >
                <input v-model="contact.name" type="text" placeholder="姓名" />
                <input v-model="contact.title" type="text" placeholder="职位" />
                <input v-model="contact.email" type="email" placeholder="邮箱" />
                <label class="contact-key">
                  <input type="checkbox" v-model="contact.is_key" />
                  重点
                </label>
                <button class="ghost" @click="removeContact(index)">删除</button>
              </div>
            </div>
            <div v-else class="empty-placeholder">暂无联系人，请手动添加。</div>
            <div class="actions">
              <button class="primary" @click="handleSaveCompany">保存并继续</button>
              <button
                v-if="flowStore.customerId"
                class="ghost"
                @click="handleSyncContacts"
              >更新联系人</button>
            </div>
          </div>
        </div>
      </section>

      <section class="step-card" :class="{ 'step-card--disabled': !flowStore.customerId }">
        <div class="step-card__head">
          <div>
            <h2 class="step-card__title">Step 2 · AI 辅助客户价值评级</h2>
            <p class="step-card__description">基于评级规则自动给出 A/B/C 建议，可人工调整。</p>
          </div>
        </div>
        <div class="step-card__body">
          <div v-if="!flowStore.customerId" class="empty-placeholder">请先完成 Step 1。</div>
          <div v-else class="grade-block">
            <button class="primary" :disabled="flowStore.loading.grade" @click="flowStore.fetchGrade">
              {{ flowStore.loading.grade ? '评估中...' : '获取 AI 评级' }}
            </button>
            <div v-if="flowStore.gradeSuggestion" class="grade-result">
              <p>AI 建议等级：<strong>{{ flowStore.gradeSuggestion.suggested_grade }}</strong></p>
              <p class="reason">理由：{{ flowStore.gradeSuggestion.reason }}</p>
              <div class="grade-actions">
                <button class="primary" @click="flowStore.confirmGrade('A', flowStore.gradeSuggestion.reason)">确认 A 级</button>
                <button class="ghost" @click="flowStore.confirmGrade('B', flowStore.gradeSuggestion.reason)">调整为 B 级</button>
                <button class="ghost" @click="flowStore.confirmGrade('C', flowStore.gradeSuggestion.reason)">调整为 C 级</button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="step-card" :class="{ 'step-card--disabled': flowStore.step < 3 }">
        <div class="step-card__head">
          <div>
            <h2 class="step-card__title">Step 3 · 生成产品切入点分析</h2>
            <p class="step-card__description">结合客户业务与我方优势，生成可编辑的切入报告。</p>
          </div>
        </div>
        <div class="step-card__body">
          <div v-if="flowStore.step < 3" class="empty-placeholder">请先完成前两步。</div>
          <div v-else>
            <button class="primary" :disabled="flowStore.loading.analysis" @click="flowStore.fetchAnalysis">
              {{ flowStore.loading.analysis ? '生成中...' : flowStore.analysis ? '重新生成' : '生成分析' }}
            </button>
            <div v-if="flowStore.analysis" class="analysis-block">
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
              <div class="actions">
                <button class="primary" @click="handleAnalysisNext">保存并继续</button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="step-card" :class="{ 'step-card--disabled': flowStore.step < 4 }">
        <div class="step-card__head">
          <div>
            <h2 class="step-card__title">Step 4 · 生成个性化开发信</h2>
            <p class="step-card__description">一键生成开发信草稿，确认后保存为首次跟进记录。</p>
          </div>
        </div>
        <div class="step-card__body">
          <div v-if="flowStore.step < 4" class="empty-placeholder">请先生成切入点分析。</div>
          <div v-else>
            <button class="primary" :disabled="flowStore.loading.email" @click="flowStore.generateEmail">
              {{ flowStore.loading.email ? '生成中...' : flowStore.emailDraft ? '重新生成' : '生成开发信' }}
            </button>
            <div v-if="flowStore.emailDraft" class="email-block">
              <label>
                <span>邮件标题</span>
                <input v-model="flowStore.emailDraft.subject" type="text" />
              </label>
              <label>
                <span>邮件正文</span>
                <textarea v-model="flowStore.emailDraft.body" rows="8"></textarea>
              </label>
              <div class="actions">
                <button class="primary" :disabled="flowStore.loading.followup" @click="flowStore.saveInitialFollowup()">
                  {{ flowStore.loading.followup ? '保存中...' : '保存为首次跟进记录' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="step-card" :class="{ 'step-card--disabled': flowStore.step < 5 }">
        <div class="step-card__head">
          <div>
            <h2 class="step-card__title">Step 5 · 设置自动化邮件跟进</h2>
            <p class="step-card__description">选择跟进时间，系统将在到期时自动生成并发送邮件。</p>
          </div>
        </div>
        <div class="step-card__body">
          <div v-if="flowStore.step < 5" class="empty-placeholder">请先保存首次跟进记录。</div>
          <div v-else class="followup-block">
            <div class="quick-buttons">
              <button
                v-for="opt in [3, 7, 14]"
                :key="opt"
                class="ghost"
                :disabled="flowStore.loading.schedule"
                @click="flowStore.createSchedule(opt)"
              >{{ opt }} 天后跟进</button>
            </div>
            <div v-if="flowStore.scheduledTask" class="scheduled-info">
              已设置任务，计划发送时间：{{ formatDate(flowStore.scheduledTask.due_at) }}
            </div>
            <div class="actions">
              <button class="primary" @click="flowStore.resetFlow">完成并开始下一个</button>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useFlowStore } from '../stores/flow'

const router = useRouter()
const flowStore = useFlowStore()

const queryInput = ref('')
const companyForm = ref({ name: '', website: '', country: '', summary: '' })
const contactsLocal = ref([])

watch(
  () => flowStore.resolveResult,
  (value) => {
    if (!value) return
    companyForm.value = {
      name: value?.name || flowStore.query,
      website: value?.website || value?.candidates?.[0]?.url || '',
      country: value?.country || '',
      summary: value?.summary || '',
    }
    contactsLocal.value = (value?.contacts || []).map((item) => ({ ...item }))
  }
)

watch(
  () => flowStore.step,
  (step) => {
    if (step === 1 && !flowStore.resolveResult) {
      queryInput.value = ''
      companyForm.value = { name: '', website: '', country: '', summary: '' }
      contactsLocal.value = []
    }
    if (step === 2 && !flowStore.gradeSuggestion) {
      flowStore.fetchGrade()
    }
  }
)

watch(
  () => flowStore.contacts,
  (value) => {
    if (!value) return
    contactsLocal.value = value.map((item) => ({ ...item }))
  },
  { deep: true }
)

const handleResolve = () => {
  flowStore.startResolve(queryInput.value)
}

const handleSaveCompany = () => {
  flowStore.contacts = contactsLocal.value.map((item) => ({ ...item }))
  flowStore.saveCompany(companyForm.value)
}

const handleSyncContacts = () => {
  flowStore.updateContacts(contactsLocal.value.map((item) => ({ ...item })))
}

const addContact = () => {
  contactsLocal.value.push({ name: '', title: '', email: '', is_key: contactsLocal.value.length === 0 })
}

const removeContact = (index) => {
  contactsLocal.value.splice(index, 1)
}

const handleAnalysisNext = async () => {
  await flowStore.persistAnalysis()
  flowStore.step = 4
}

const goSettings = () => {
  router.push({ name: 'settings' })
}

const formatDate = (value) => {
  if (!value) return ''
  return new Date(value).toLocaleString()
}
</script>

<style scoped>
.flow-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--surface-background);
  color: var(--text-primary);
}

.flow-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 32px 48px 16px;
}

.flow-header__title {
  font-size: 28px;
  font-weight: 700;
}

.flow-header__settings {
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: 999px;
  padding: 10px 24px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.flow-header__settings:hover {
  border-color: var(--accent-color);
  color: var(--accent-color);
}

.flow-progress {
  margin: 0 48px 24px;
  color: var(--text-tertiary);
  font-size: 14px;
}

.flow-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0 48px 48px;
}

.step-card {
  background: var(--surface-elevated);
  border-radius: 20px;
  padding: 24px 32px;
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.05);
}

.step-card--disabled {
  opacity: 0.5;
  pointer-events: none;
}

.step-card__head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.step-card__title {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 8px;
}

.step-card__description {
  margin: 0;
  color: var(--text-secondary);
}

.step-card__badge {
  padding: 4px 12px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.1);
  color: var(--accent-color);
  font-size: 12px;
}

.step-card__body {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-row {
  display: flex;
  gap: 12px;
}

.form-row input {
  flex: 1;
}

input,
textarea {
  width: 100%;
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--border-subtle);
  background: var(--surface-muted);
  font-size: 14px;
  color: var(--text-primary);
}

textarea {
  resize: vertical;
}

button {
  border-radius: 12px;
  border: none;
  padding: 12px 24px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

button.primary {
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
}

button.primary:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

button.ghost {
  background: rgba(148, 163, 184, 0.12);
  color: var(--text-secondary);
}

.result-block,
.analysis-block,
.email-block,
.followup-block {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}

.contacts-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.contacts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.contact-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(160px, 1fr)) 80px auto;
  gap: 12px;
  align-items: center;
}

.contact-key {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}

.actions {
  display: flex;
  gap: 12px;
}

.empty-placeholder {
  color: var(--text-tertiary);
  font-size: 14px;
  padding: 24px;
  border: 1px dashed var(--border-subtle);
  border-radius: 12px;
  text-align: center;
}

.grade-block,
.grade-result,
.grade-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.grade-result .reason {
  color: var(--text-secondary);
  font-size: 14px;
}

.quick-buttons {
  display: flex;
  gap: 12px;
}

.scheduled-info {
  font-size: 14px;
  color: var(--text-secondary);
}
</style>
