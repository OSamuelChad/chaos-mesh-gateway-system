import { linksToActions } from '../services/hypermedia.js'
import HealthBadge from './HealthBadge.jsx'

const METHOD_STYLES = {
  GET: 'text-signal',
  POST: 'text-healthy',
  PUT: 'text-degraded',
}

/**
 * Painel do nó selecionado. Os botões são gerados EXCLUSIVAMENTE a partir de
 * `detail._links` — o React não conhece regras de negócio das ações.
 */
export default function ActionPanel({ detail, loading, error, runningRel, onAction }) {
  if (!detail && !loading && !error) {
    return (
      <div className="grid h-full place-items-center p-8 text-center">
        <div>
          <p className="text-lg font-medium">Selecione um nó</p>
          <p className="mt-1 text-sm text-ink-400">Clique em um serviço no grafo para carregar seus dados e ações via HATEOAS.</p>
        </div>
      </div>
    )
  }

  const actions = linksToActions(detail?._links)
  const mutations = actions.filter((a) => !a.isSafe)
  const navigation = actions.filter((a) => a.isSafe)

  return (
    <div className="flex h-full flex-col gap-5 p-5">
      <header className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="font-mono text-[10px] uppercase tracking-[0.25em] text-ink-400">nó selecionado</p>
          <h2 className="truncate text-2xl font-semibold tracking-tight">{detail?.name ?? '…'}</h2>
        </div>
        {detail && <HealthBadge health={detail.health} size="lg" />}
      </header>

      {error && <p role="alert" className="rounded-lg border border-critical/40 bg-critical/10 px-3 py-2 text-sm text-critical">{error}</p>}
      {detail?.error && <p className="rounded-lg border border-critical/40 bg-critical/10 px-3 py-2 text-sm text-critical">{detail.error}</p>}

      {detail?.summary?.length > 0 && (
        <dl className="grid grid-cols-3 gap-px overflow-hidden rounded-xl border border-ink-700 bg-ink-700">
          {detail.summary.map((m) => (
            <div key={m.label} className="bg-ink-900 px-3 py-2.5">
              <dt className="text-[11px] text-ink-400">{m.label}</dt>
              <dd className="truncate font-mono text-sm font-bold">{m.value}</dd>
            </div>
          ))}
        </dl>
      )}

      <section aria-labelledby="actions-title">
        <div className="mb-2 flex items-baseline justify-between">
          <h3 id="actions-title" className="font-mono text-[11px] uppercase tracking-[0.2em] text-ink-400">Ações disponíveis</h3>
          <span className="font-mono text-[10px] text-ink-600">via _links</span>
        </div>

        {loading && !detail && <p className="text-sm text-ink-400">Carregando…</p>}
        {detail && mutations.length === 0 && <p className="text-sm text-ink-400">Nenhuma ação oferecida pelo gateway neste estado.</p>}

        <div className="flex flex-col gap-2">
          {mutations.map((action) => (
            <ActionButton key={action.rel} action={action} running={runningRel === action.rel} disabled={Boolean(runningRel)} onClick={() => onAction(action)} primary />
          ))}
        </div>

        {navigation.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-2">
            {navigation.map((action) =>
              action.isDocument ? (
                <a
                  key={action.rel}
                  href={action.href}
                  target="_blank"
                  rel="noreferrer"
                  className="rounded-lg border border-ink-700 px-3 py-1.5 text-sm text-ink-300 transition hover:border-signal hover:text-signal focus-visible:outline-2 focus-visible:outline-signal"
                >
                  {action.title} ↗
                </a>
              ) : (
                <ActionButton key={action.rel} action={action} running={runningRel === action.rel} disabled={Boolean(runningRel)} onClick={() => onAction(action)} />
              ),
            )}
          </div>
        )}
      </section>
    </div>
  )
}

function ActionButton({ action, running, disabled, onClick, primary = false }) {
  const base = 'group flex cursor-pointer items-center gap-3 rounded-lg border text-left transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-signal active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-50'
  const look = primary
    ? 'w-full border-ink-700 bg-ink-850 px-3.5 py-3 hover:border-ink-400 hover:bg-ink-800'
    : 'border-ink-700 px-3 py-1.5 text-sm text-ink-300 hover:border-signal hover:text-signal'

  return (
    <button type="button" onClick={onClick} disabled={disabled} title={action.body ? `payload: ${JSON.stringify(action.body)}` : action.href} className={`${base} ${look}`}>
      <span className={`w-9 shrink-0 font-mono text-[10px] font-bold ${METHOD_STYLES[action.method] ?? 'text-ink-300'}`}>{action.method}</span>
      <span className="min-w-0 flex-1">
        <span className={`block truncate ${primary ? 'font-medium' : ''}`}>{running ? 'Executando…' : action.title}</span>
        {primary && <span className="block truncate font-mono text-[10px] text-ink-400">{action.href}</span>}
      </span>
      {primary && running && <span aria-hidden className="size-4 animate-spin rounded-full border-2 border-ink-600 border-t-signal" />}
    </button>
  )
}
