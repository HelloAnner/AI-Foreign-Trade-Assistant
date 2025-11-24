const RSA_PREFIX = 'rsa:'
const AES_PREFIX = 'enc:'
const CTR_PREFIX = 'ctr:'
const LEGACY_PREFIXES = [RSA_PREFIX, AES_PREFIX, CTR_PREFIX]
const SENSITIVE_KEY_PATTERNS = [/password/i, /api_key/i, /secret/i, /token/i]
const MASKED_SECRET_PLACEHOLDER = '******'

const encoder = new TextEncoder()
const decoder = new TextDecoder()

let publicKeyPromise
let cachedMeta
let aesKeyPromise
let webCryptoSupported = false

// Check if Web Crypto is supported
try {
  if (typeof window !== 'undefined' && window.crypto && window.crypto.subtle) {
    webCryptoSupported = true
  }
} catch (e) {
  webCryptoSupported = false
}

const getSymmetricSecret = () => {
  return import.meta.env.VITE_APP_ENCRYPTION_KEY || 'AI_FTA::crypto::v1'
}

const getAesKey = () => {
  if (aesKeyPromise) return aesKeyPromise
  if (!webCryptoSupported) {
    throw new Error('当前环境不支持 Web Crypto，无法使用 AES-GCM 加密')
  }
  aesKeyPromise = (async () => {
    const secret = getSymmetricSecret()
    const hash = await window.crypto.subtle.digest('SHA-256', encoder.encode(secret))
    return window.crypto.subtle.importKey('raw', hash, { name: 'AES-GCM' }, false, ['encrypt', 'decrypt'])
  })()
  return aesKeyPromise
}

const pemToArrayBuffer = (pem) => {
  const clean = pem.replace(/-----BEGIN PUBLIC KEY-----/g, '').replace(/-----END PUBLIC KEY-----/g, '').replace(/\s+/g, '')
  const binary = window.atob(clean)
  const len = binary.length
  const buffer = new ArrayBuffer(len)
  const view = new Uint8Array(buffer)
  for (let i = 0; i < len; i += 1) {
    view[i] = binary.charCodeAt(i)
  }
  return buffer
}

const arrayBufferToBase64 = (buffer) => {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  const chunk = 0x8000
  for (let i = 0; i < bytes.length; i += chunk) {
    const sub = bytes.subarray(i, i + chunk)
    binary += String.fromCharCode(...sub)
  }
  return window.btoa(binary)
}

const fetchPublicKey = async () => {
  const resp = await fetch('/api/auth/public-key', { credentials: 'same-origin' })
  const payload = await resp.json()
  if (!payload.ok) {
    throw new Error(payload.error || '无法获取加密公钥')
  }
  if (!payload.data?.public_key) {
    throw new Error('服务端未返回公钥')
  }
  cachedMeta = payload.data
  return cachedMeta
}

const getCryptoKey = async () => {
  if (publicKeyPromise) return publicKeyPromise
  if (!webCryptoSupported) {
    throw new Error('当前环境不支持 Web Crypto，无法使用 RSA-OAEP 加密')
  }
  publicKeyPromise = (async () => {
    const meta = await fetchPublicKey()
    const binary = pemToArrayBuffer(meta.public_key)
    const cryptoKey = await window.crypto.subtle.importKey(
      'spki',
      binary,
      { name: 'RSA-OAEP', hash: 'SHA-256' },
      false,
      ['encrypt']
    )
    return cryptoKey
  })()
  return publicKeyPromise
}

export const resetCryptoKey = () => {
  publicKeyPromise = null
  cachedMeta = null
}

const encryptWithFallback = async (text) => {
  // Fallback CTR encryption using crypto-js when Web Crypto is not available
  try {
    // Dynamic import for crypto-js to avoid bundling issues
    const CryptoJS = await import('crypto-js')

    const secret = getSymmetricSecret()
    const key = CryptoJS.SHA256(secret).toString(CryptoJS.enc.Hex)

    // Generate 16 bytes IV for CTR mode
    const iv = CryptoJS.lib.WordArray.random(16)

    // Encrypt with AES-256-CTR
    const encrypted = CryptoJS.AES.encrypt(
      text,
      CryptoJS.enc.Hex.parse(key),
      {
        iv: iv,
        mode: CryptoJS.mode.CTR,
        padding: CryptoJS.pad.NoPadding
      }
    )

    // Combine IV + ciphertext
    const combined = iv.clone()
    combined.concat(encrypted.ciphertext)

    // Return CTR prefix
    return `${CTR_PREFIX}${combined.toString(CryptoJS.enc.Base64)}`
  } catch (error) {
    console.error('Fallback encryption failed:', error)
    throw new Error('加密失败，当前环境不支持 Web Crypto，且后备加密库加载失败')
  }
}

export const encryptValue = async (text, withPrefix = true) => {
  if (!text) return ''

  // If Web Crypto is not supported, use fallback
  if (!webCryptoSupported) {
    return encryptWithFallback(text)
  }

  try {
    const key = await getCryptoKey()
    const encoded = encoder.encode(text)
    const cipherBuffer = await window.crypto.subtle.encrypt({ name: 'RSA-OAEP' }, key, encoded)
    const base64 = arrayBufferToBase64(cipherBuffer)
    return withPrefix ? `${RSA_PREFIX}${base64}` : base64
  } catch (error) {
    resetCryptoKey()
    throw error
  }
}

const isAlreadyEncrypted = (value) => {
  if (typeof value !== 'string') return false
  return LEGACY_PREFIXES.some((prefix) => value.startsWith(prefix))
}

const decryptAES = async (value) => {
  if (!value) return value
  const raw = atob(value)
  const length = raw.length
  const nonceLength = 12
  if (length <= nonceLength) {
    throw new Error('密文格式不正确')
  }
  const nonce = new Uint8Array(nonceLength)
  const data = new Uint8Array(length - nonceLength)
  for (let i = 0; i < nonceLength; i += 1) {
    nonce[i] = raw.charCodeAt(i)
  }
  for (let i = nonceLength; i < length; i += 1) {
    data[i - nonceLength] = raw.charCodeAt(i)
  }
  const key = await getAesKey()
  const plainBuffer = await window.crypto.subtle.decrypt({ name: 'AES-GCM', iv: nonce }, key, data)
  return decoder.decode(plainBuffer)
}

const decryptValue = async (value) => {
  if (typeof value !== 'string') {
    return value
  }
  const trimmed = value.trim()
  if (!trimmed) return trimmed
  if (trimmed.startsWith(AES_PREFIX)) {
    const payload = trimmed.slice(AES_PREFIX.length)
    return decryptAES(payload)
  }
  return value
}

const shouldEncryptKey = (key) => {
  if (!key) return false
  return SENSITIVE_KEY_PATTERNS.some((pattern) => pattern.test(key))
}

export const encryptSensitivePayload = async (payload) => {
  if (!payload || typeof payload !== 'object') return payload
  const clone = { ...payload }
  for (const [key, value] of Object.entries(clone)) {
    if (!shouldEncryptKey(key) || typeof value !== 'string') continue
    const trimmed = value.trim()
    if (!trimmed || trimmed === MASKED_SECRET_PLACEHOLDER || isAlreadyEncrypted(trimmed)) continue
    clone[key] = await encryptValue(trimmed)
  }
  return clone
}

export const decryptSensitivePayload = async (payload) => {
  if (!payload || typeof payload !== 'object') return payload
  const clone = { ...payload }
  for (const [key, value] of Object.entries(clone)) {
    if (typeof value !== 'string') continue
    if (!shouldEncryptKey(key) && !value.startsWith(AES_PREFIX)) continue
    try {
      const decrypted = await decryptValue(value)
      clone[key] = decrypted
    } catch (error) {
      console.error(`decrypt field ${key} failed`, error)
      clone[key] = ''
    }
  }
  return clone
}

export const encryptFields = async (payload, fields) => {
  if (!payload || !Array.isArray(fields) || !fields.length) return payload
  const clone = { ...payload }
  for (const field of fields) {
    const value = clone[field]
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (!trimmed || trimmed === MASKED_SECRET_PLACEHOLDER || isAlreadyEncrypted(trimmed)) {
        continue
      }
      clone[field] = await encryptValue(value)
    }
  }
  return clone
}

export { RSA_PREFIX, MASKED_SECRET_PLACEHOLDER }
