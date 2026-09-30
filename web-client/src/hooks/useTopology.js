import { useCallback, useEffect, useRef, useState } from 'react'
import { api, describeError } from '../services/api.js'

const TOPOLOGY_PATH = '/api/v1/mesh/topology'

/** Consulta a topologia periodicamente, sem sobrepor requisições. */
export function useTopology(intervalMs = 2000) {
  const [topology, setTopology] = useState(null)
  const [error, setError] = useState('')
  const [updatedAt, setUpdatedAt] = useState(null)
  const inFlight = useRef(false)

  const refresh = useCallback(async () => {
    if (inFlight.current) return
    inFlight.current = true
    try {
      const { data } = await api.get(TOPOLOGY_PATH)
      setTopology(data)
      setError('')
      setUpdatedAt(new Date())
    } catch (err) {
      setError(describeError(err))
    } finally {
      inFlight.current = false
    }
  }, [])

  useEffect(() => {
    refresh()
    const id = setInterval(refresh, intervalMs)
    return () => clearInterval(id)
  }, [refresh, intervalMs])

  return { topology, error, updatedAt, refresh }
}
