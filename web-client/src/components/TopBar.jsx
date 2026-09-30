import HealthBadge from './HealthBadge.jsx'

export default function TopBar({ username, breakerState, meshHealth, lastUpdate, onLogout }) {
  return (
    <header className="flex flex-wrap items-center justify-between gap-3 border-b border-ink-800 bg-ink-900/70 px-4 py-3 backdrop-blur sm:px-6">
      <div className="flex items-center gap-3">
        <div className="flex gap-1" aria-hidden>
          <span className="size-2.5 rounded-full bg-healthy" />
          <span className="size-2.5 rounded-full bg-degraded" />
          <span className="size-2.5 rounded-full bg-critical" />
        </div>
        <div>
          <p className="text-sm font-semibold leading-tight">Chaos Mesh</p>
          <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-400">resilience visualizer</p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 font-mono text-xs">
        <span className="hidden text-ink-400 sm:inline">malha</span>
        <HealthBadge health={meshHealth} />
        <span className="rounded-md border border-ink-700 px-2 py-1 text-ink-300">
          breaker <strong className="text-ink-100">{breakerState ?? '—'}</strong>
        </span>
        {lastUpdate && <span className="hidden text-ink-400 md:inline">atualizado {lastUpdate}</span>}
        <span className="text-ink-400">·</span>
        <span className="text-ink-300">{username}</span>
        <button type="button" onClick={onLogout} className="cursor-pointer rounded-md border border-ink-700 px-2.5 py-1 text-ink-300 transition hover:border-critical hover:text-critical focus-visible:outline-2 focus-visible:outline-signal">
          sair
        </button>
      </div>
    </header>
  )
}
