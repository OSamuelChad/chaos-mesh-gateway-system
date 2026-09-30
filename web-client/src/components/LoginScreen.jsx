import { useState } from 'react'
import { describeError } from '../services/api.js'
import { useSession } from '../services/session.jsx'

export default function LoginScreen() {
  const { login } = useSession()
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event) {
    event.preventDefault()
    if (!username.trim() || !password) {
      setError('Informe usuário e senha.')
      return
    }
    setLoading(true)
    setError('')
    try {
      await login(username.trim(), password)
    } catch (err) {
      setError(describeError(err))
      setLoading(false)
    }
  }

  return (
    <main className="bg-grid relative grid min-h-full place-items-center overflow-hidden px-4 py-12">
      <div aria-hidden className="pointer-events-none absolute -top-40 left-1/2 h-[480px] w-[780px] -translate-x-1/2 rounded-full bg-signal/10 blur-3xl" />

      <section aria-labelledby="login-title" className="relative w-full max-w-md">
        <header className="mb-8">
          <div className="mb-6 flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.3em] text-ink-400">
            <span className="size-2 rounded-full bg-healthy" />
            <span className="size-2 rounded-full bg-degraded" />
            <span className="size-2 rounded-full bg-critical" />
            <span className="ml-2">chaos-mesh / gateway</span>
          </div>
          <h1 id="login-title" className="text-4xl font-semibold leading-tight tracking-tight">
            Resilience
            <br />
            <span className="text-signal">Visualizer</span>
          </h1>
          <p className="mt-3 text-ink-300">Autentique-se no API Gateway para observar e perturbar a malha de serviços.</p>
        </header>

        <form onSubmit={handleSubmit} className="rounded-2xl border border-ink-700 bg-ink-900/80 p-6 shadow-2xl shadow-black/40 backdrop-blur" noValidate>
          <label className="block">
            <span className="font-mono text-xs uppercase tracking-widest text-ink-400">Usuário</span>
            <input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              maxLength={64}
              className="mt-2 w-full rounded-lg border border-ink-700 bg-ink-950 px-3 py-2.5 font-mono text-sm outline-none transition focus:border-signal focus:ring-2 focus:ring-signal/30"
            />
          </label>
          <label className="mt-4 block">
            <span className="font-mono text-xs uppercase tracking-widest text-ink-400">Senha</span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              maxLength={128}
              placeholder="password"
              className="mt-2 w-full rounded-lg border border-ink-700 bg-ink-950 px-3 py-2.5 font-mono text-sm outline-none transition placeholder:text-ink-600 focus:border-signal focus:ring-2 focus:ring-signal/30"
            />
          </label>

          {error && (
            <p role="alert" className="mt-4 rounded-lg border border-critical/40 bg-critical/10 px-3 py-2 text-sm text-critical">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="mt-6 w-full cursor-pointer rounded-lg bg-signal px-4 py-2.5 font-semibold text-ink-950 transition hover:bg-sky-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-signal active:scale-[0.99] disabled:cursor-wait disabled:opacity-60"
          >
            {loading ? 'Autenticando…' : 'Obter token JWT'}
          </button>
          <p className="mt-4 text-center font-mono text-[11px] text-ink-400">demo: admin / password</p>
        </form>
      </section>
    </main>
  )
}
