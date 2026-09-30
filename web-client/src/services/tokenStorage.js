const KEY = 'chaos-mesh.session'

/** Lê a sessão salva; descarta se estiver corrompida ou expirada. */
export function readToken() {
  try {
    const raw = sessionStorage.getItem(KEY)
    if (!raw) return null
    const session = JSON.parse(raw)
    if (!session?.token || (session.expiresAt && new Date(session.expiresAt) <= new Date())) {
      sessionStorage.removeItem(KEY)
      return null
    }
    return session
  } catch {
    return null
  }
}

export function storeToken(session) {
  try {
    sessionStorage.setItem(KEY, JSON.stringify(session))
  } catch {
    // sessionStorage indisponível (modo privado): a sessão vive só em memória.
  }
}

export function clearToken() {
  try {
    sessionStorage.removeItem(KEY)
  } catch {
    // ignorado: nada a limpar
  }
}
