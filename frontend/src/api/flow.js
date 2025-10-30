import http from './http'

export const resolveCompany = async (query) => {
  const { data } = await http.post('/companies/resolve', { query })
  return data
}

export const createCompany = async (payload) => {
  const { data } = await http.post('/companies', payload)
  return data
}

export const replaceContacts = async (customerId, contacts) => {
  const { data } = await http.post(`/companies/${customerId}/contacts`, {
    contacts,
  })
  return data
}

export const suggestGrade = async (customerId) => {
  const { data } = await http.post(`/companies/${customerId}/grade/suggest`)
  return data
}

export const confirmGrade = async (customerId, payload) => {
  const { data } = await http.post(`/companies/${customerId}/grade/confirm`, payload)
  return data
}

export const generateAnalysis = async (customerId) => {
  const { data } = await http.post(`/companies/${customerId}/analysis`)
  return data
}

export const updateAnalysis = async (customerId, payload) => {
  const { data } = await http.put(`/companies/${customerId}/analysis`, payload)
  return data
}

export const generateEmailDraft = async (customerId) => {
  const { data } = await http.post(`/companies/${customerId}/email-draft`)
  return data
}

export const updateEmailDraft = async (emailId, payload) => {
  const { data } = await http.put(`/emails/${emailId}`, payload)
  return data
}

export const saveFirstFollowup = async (customerId, emailId, notes = '') => {
  const { data } = await http.post(`/companies/${customerId}/followup/first-save`, {
    email_id: emailId,
    notes,
  })
  return data
}

export const scheduleFollowup = async (payload) => {
  const { data } = await http.post('/followups/schedule', payload)
  return data
}

export const listScheduledTasks = async (status) => {
  const { data } = await http.get('/scheduled-tasks', {
    params: { status },
  })
  return data
}

export const runTaskNow = async (taskId) => {
  const { data } = await http.post(`/scheduled-tasks/${taskId}/run-now`)
  return data
}
