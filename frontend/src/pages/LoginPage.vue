<template>
  <div class="login-shell">
    <div class="login-card">
      <div class="login-head">
        <h1>访问控制</h1>
        <p>系统已开启安全防护，请输入管理员口令后再进入。</p>
      </div>
      <form class="login-form" @submit.prevent="handleSubmit">
        <label>
          <span>访问口令</span>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            placeholder="foreign_***"
          />
        </label>
        <button type="submit" :disabled="loading">
          {{ loading ? '校验中…' : '进入系统' }}
        </button>
        <p v-if="errorMessage" class="login-error">{{ errorMessage }}</p>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login } from '../api/auth'
import { setToken } from '../utils/auth'
import { useUiStore } from '../stores/ui'

const route = useRoute()
const router = useRouter()
const uiStore = useUiStore()

const password = ref('')
const loading = ref(false)
const errorMessage = ref('')

const handleSubmit = async () => {
  const trimmed = (password.value || '').trim()
  if (!trimmed) {
    errorMessage.value = '请输入访问口令'
    return
  }

  loading.value = true
  errorMessage.value = ''
  try {
    const resp = await login(trimmed)
    if (resp.ok) {
      const token = resp.data?.token
      if (!token) {
        throw new Error('服务端未返回 token')
      }
      setToken(token, resp.data?.expires_at)
      uiStore.pushToast('登录成功', 'success')
      const redirect = typeof route.query.redirect === 'string' && route.query.redirect ? route.query.redirect : '/'
      router.replace(redirect)
    } else {
      errorMessage.value = resp.error || '登录失败，请重试'
    }
  } catch (err) {
    errorMessage.value = err.message || '登录失败，请稍后再试'
  } finally {
    loading.value = false
    password.value = ''
  }
}
</script>

<style scoped>
.login-shell {
  min-height: calc(100vh - 80px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 16px 120px;
}

.login-card {
  width: min(480px, 100%);
  background: var(--surface-card);
  border-radius: 28px;
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
  padding: 48px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.login-head h1 {
  margin: 0 0 4px;
  font-size: 28px;
}

.login-head p {
  margin: 0;
  color: var(--text-secondary);
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.login-form label {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 14px;
  color: var(--text-secondary);
}

.login-form input {
  border-radius: 16px;
  border: 1px solid var(--border-default);
  padding: 14px 18px;
  font-size: 15px;
  background: #fff;
}

.login-form input:focus {
  outline: none;
  border-color: var(--primary-500);
  box-shadow: 0 0 0 4px rgba(19, 73, 236, 0.14);
}

.login-form button {
  border: none;
  border-radius: var(--radius-full);
  padding: 14px 0;
  font-size: 16px;
  font-weight: 600;
  background: linear-gradient(135deg, var(--primary-500), #0f36d2);
  color: #fff;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.login-form button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.login-error {
  margin: 0;
  color: var(--danger-500);
  font-size: 14px;
  text-align: center;
}

@media (max-width: 640px) {
  .login-card {
    padding: 32px 24px;
    border-radius: 20px;
  }
}
</style>
