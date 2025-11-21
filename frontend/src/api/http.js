import axios from 'axios'
import { getToken, clearToken } from '../utils/auth'

let redirecting = false

const triggerLoginRedirect = () => {
  if (typeof window === 'undefined' || redirecting) return
  const { pathname, search } = window.location
  if (pathname === '/login') return
  redirecting = true
  const redirect = encodeURIComponent(`${pathname}${search}`)
  const target = redirect && redirect !== encodeURIComponent('/login') ? `/login?redirect=${redirect}` : '/login'
  window.location.assign(target)
}

const http = axios.create({
  baseURL: '/api',
  timeout: 300000,
})

http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => response,
  (error) => {
    const message =
      error.response?.data?.error || error.message || '请求失败，请稍后再试。'
    const wrapped = new Error(message)
    const status = error.response?.status
    if (error.response) {
      wrapped.response = error.response
    }
    const isLoginRequest = (error.config?.url || '').includes('/auth/login')
    if (status === 401 && !isLoginRequest) {
      clearToken()
      triggerLoginRedirect()
    }
    return Promise.reject(wrapped)
  }
)

export default http
