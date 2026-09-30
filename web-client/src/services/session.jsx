import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { api, login as requestLogin, onUnauthorized } from './api.js'
import { clearToken, readToken, storeToken } from './tokenStorage.js'

const SessionContext = createContext(null)

/** Estado global de autenticação: token JWT persistido em sessionStorage. */
export function SessionProvider({ children }) {
  const [session, setSession] = useState(() => readToken())

  const logout = useCallback(() => {
    clearToken()
    setSession(null)
  }, [])

  useEffect(() => onUnauthorized(logout), [logout])

  const login = useCallback(async (username, password) => {
    const data = await requestLogin(username, password)
    const next = { token: data.access_token, username, expiresAt: data.expires_at, links: data._links }
    storeToken(next)
    setSession(next)
    return next
  }, [])

  const value = useMemo(() => ({ session, login, logout, api }), [session, login, logout])
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
}

export function useSession() {
  const ctx = useContext(SessionContext)
  if (!ctx) throw new Error('useSession deve ser usado dentro de <SessionProvider>')
  return ctx
}
