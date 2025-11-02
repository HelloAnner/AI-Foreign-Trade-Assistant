import axios from 'axios'

const http = axios.create({
  baseURL: '/api',
  timeout: 300000,
})

http.interceptors.response.use(
  (response) => response,
  (error) => {
    const message =
      error.response?.data?.error || error.message || '请求失败，请稍后再试。'
    const wrapped = new Error(message)
    if (error.response) {
      wrapped.response = error.response
    }
    return Promise.reject(wrapped)
  }
)

export default http
