import http from './http'
import { encryptValue } from '../utils/crypto'

export const login = async (password) => {
  const cipher = await encryptValue(password)
  const { data } = await http.post('/auth/login', { cipher })
  return data
}
