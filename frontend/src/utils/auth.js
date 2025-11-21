const TOKEN_KEY = 'fta:jwt'
const TOKEN_EXPIRY_KEY = 'fta:jwt-exp'
const FALLBACK_TTL_MS = 14 * 24 * 60 * 60 * 1000

const getStore = () => {
  if (typeof window === 'undefined' || !window.localStorage) {
    return null
  }
  return window.localStorage
}

export const getToken = () => {
  const store = getStore()
  if (!store) return ''
  const expiryRaw = store.getItem(TOKEN_EXPIRY_KEY)
  if (expiryRaw) {
    const expiry = Number(expiryRaw)
    if (expiry && Date.now() > expiry) {
      clearToken()
      return ''
    }
  }
  return store.getItem(TOKEN_KEY) || ''
}

export const setToken = (token, expiresAt) => {
  const store = getStore()
  if (!store) return
  if (!token) {
    clearToken()
    return
  }
  store.setItem(TOKEN_KEY, token)
  let expiry = Date.now() + FALLBACK_TTL_MS
  if (expiresAt) {
    const parsed = Date.parse(expiresAt)
    if (!Number.isNaN(parsed)) {
      expiry = parsed
    }
  }
  store.setItem(TOKEN_EXPIRY_KEY, String(expiry))
}

export const clearToken = () => {
  const store = getStore()
  if (!store) return
  store.removeItem(TOKEN_KEY)
  store.removeItem(TOKEN_EXPIRY_KEY)
}

export const isAuthenticated = () => Boolean(getToken())
