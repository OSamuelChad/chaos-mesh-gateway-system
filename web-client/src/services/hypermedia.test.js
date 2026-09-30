import { describe, expect, it } from 'vitest'
import { linksToActions } from './hypermedia.js'

describe('linksToActions', () => {
  it('converte o mapa _links em ações, com as mutações antes das consultas', () => {
    const actions = linksToActions({
      self: { href: '/api/v1/services/payment', method: 'GET', title: 'Atualizar' },
      heal: { href: '/api/v1/services/payment/chaos/reset', method: 'post', title: 'Heal' },
      'inject-latency': { href: '/x', method: 'PUT', title: 'Latência', body: { latency_ms: 2000 } },
    })

    expect(actions.map((a) => a.rel)).toEqual(['heal', 'inject-latency', 'self'])
    expect(actions[0].method).toBe('POST')
    expect(actions[1].body).toEqual({ latency_ms: 2000 })
    expect(actions[2].isSafe).toBe(true)
  })

  it('marca links para documentos não-JSON', () => {
    const [docs] = linksToActions({ docs: { href: '/swagger/index.html', method: 'GET', title: 'Docs', type: 'text/html' } })
    expect(docs.isDocument).toBe(true)
  })

  it('retorna lista vazia sem links', () => {
    expect(linksToActions(undefined)).toEqual([])
  })
})
