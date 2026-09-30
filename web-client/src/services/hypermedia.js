import { api } from './api.js'

const SAFE_METHODS = new Set(['GET', 'HEAD'])

/**
 * Converte o mapa `_links` em uma lista ordenada de ações.
 * Nenhuma regra de negócio aqui: tudo vem do servidor (href, method, title, body).
 */
export function linksToActions(links = {}) {
  return Object.entries(links)
    .map(([rel, link]) => ({
      rel,
      ...link,
      method: (link.method ?? 'GET').toUpperCase(),
      isSafe: SAFE_METHODS.has((link.method ?? 'GET').toUpperCase()),
      isDocument: Boolean(link.type) && link.type !== 'application/json',
    }))
    .sort((a, b) => Number(a.isSafe) - Number(b.isSafe) || a.rel.localeCompare(b.rel))
}

/**
 * Segue um link HATEOAS usando o método e o payload sugeridos pelo gateway.
 * Respostas de erro com corpo JSON (ex.: 503 de fallback) são devolvidas, não lançadas.
 */
export async function followLink(link) {
  try {
    const response = await api.request({
      url: link.href,
      method: link.method,
      data: SAFE_METHODS.has(link.method) ? undefined : (link.body ?? {}),
    })
    return { ok: true, status: response.status, data: response.data }
  } catch (error) {
    if (error.response?.data && typeof error.response.data === 'object') {
      return { ok: false, status: error.response.status, data: error.response.data }
    }
    throw error
  }
}
