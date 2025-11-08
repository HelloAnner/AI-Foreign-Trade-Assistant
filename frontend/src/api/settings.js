import http from './http'

export const fetchSettings = async () => {
  const { data } = await http.get('/settings')
  return data
}

export const saveSettings = async (payload) => {
  const { data } = await http.put('/settings', payload)
  return data
}

export const testLLM = async () => {
  const { data } = await http.post('/settings/test-llm')
  return data
}

export const testSMTP = async (payload) => {
  const body = payload ?? {}
  const { data } = await http.post('/settings/test-smtp', body)
  return data
}

export const testSearch = async () => {
  const { data } = await http.post('/settings/test-search')
  return data
}
