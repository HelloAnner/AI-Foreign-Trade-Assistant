import http from './http'
import { encryptSensitivePayload, decryptSensitivePayload } from '../utils/crypto'

const preparePayload = async (payload) => {
  if (!payload) return payload
  return encryptSensitivePayload(payload)
}

export const fetchSettings = async () => {
  const { data } = await http.get('/settings')
  if (data?.ok && data.data) {
    data.data = await decryptSensitivePayload(data.data)
  }
  return data
}

export const saveSettings = async (payload) => {
  const body = await preparePayload(payload)
  const { data } = await http.put('/settings', body)
  if (data?.ok && data.data) {
    data.data = await decryptSensitivePayload(data.data)
  }
  return data
}

export const testLLM = async () => {
  const { data } = await http.post('/settings/test-llm')
  return data
}

export const testSMTP = async (payload) => {
  const body = payload ? await preparePayload(payload) : {}
  const { data } = await http.post('/settings/test-smtp', body)
  return data
}

export const testSearch = async () => {
  const { data } = await http.post('/settings/test-search')
  return data
}
