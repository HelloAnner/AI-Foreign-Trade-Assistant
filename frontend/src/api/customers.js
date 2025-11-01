import http from './http'

export const listCustomers = async (params = {}) => {
  const { data } = await http.get('/customers', { params })
  return data
}

export const getCustomerDetail = async (customerId) => {
  const { data } = await http.get(`/customers/${customerId}`)
  return data
}

export const deleteCustomer = async (customerId) => {
  const { data } = await http.delete(`/customers/${customerId}`)
  return data
}

export const triggerAutomation = async (customerId) => {
  const { data } = await http.post(`/companies/${customerId}/automation`)
  return data
}
