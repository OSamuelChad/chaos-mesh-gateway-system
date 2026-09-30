import axios from 'axios'
import { readToken } from './tokenStorage.js'

/**
 * Cliente HTTP do gateway. Os hrefs vindos do HATEOAS são caminhos absolutos
 * (ex.: /api/v1/services/payment), resolvidos contra a mesma origem —
 * o Vite (dev) ou o Nginx (Docker) encaminham /api ao API Gateway.
 */
export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '',
  timeout: 60_000,
  headers: { 'Content-Type': 'application/json' },
})

const unauthorizedListeners = new Set()

/** Registra um callback para 401; retorna a função de cancelamento. */
export function onUnauthorized(listener) {
  unauthorizedListeners.add(listener)
  return () => unauthorizedListeners.delete(listener)
}

api.interceptors.request.use((config) => {
  const session = readToken()
  if (session?.token) {
    config.headers.Authorization = `Bearer ${session.token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const isLogin = error.config?.url?.endsWith('/auth/login')
    if (error.response?.status === 401 && !isLogin) {
      unauthorizedListeners.forEach((listener) => listener())
    }
    return Promise.reject(error)
  },
)

export async function login(username, password) {
  const { data } = await api.post('/api/v1/auth/login', { username, password })
  return data
}

/** Extrai uma mensagem amigável de um erro do axios. */
export function describeError(error) {
  if (error.response?.data?.error) return error.response.data.error
  if (error.response?.data?.result?.message) return error.response.data.result.message
  if (error.code === 'ECONNABORTED') return 'Tempo de resposta esgotado.'
  if (!error.response) return 'Gateway inacessível. Verifique se os serviços estão rodando.'
  return `Erro HTTP ${error.response.status}`
}
