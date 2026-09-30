import { useCallback, useRef, useState } from 'react'
import ActionPanel from './components/ActionPanel.jsx'
import ActivityLog from './components/ActivityLog.jsx'
import LoginScreen from './components/LoginScreen.jsx'
import MeshGraph from './components/MeshGraph.jsx'
import ResponseViewer from './components/ResponseViewer.jsx'
import TopBar from './components/TopBar.jsx'
import { useTopology } from './hooks/useTopology.js'
import { describeError } from './services/api.js'
import { followLink } from './services/hypermedia.js'
import { useSession } from './services/session.jsx'

const MAX_LOG_ENTRIES = 40

export default function App() {
  const { session, logout } = useSession()
  return session ? <Dashboard username={session.username} onLogout={logout} /> : <LoginScreen />
}

function Dashboard({ username, onLogout }) {
  const { topology, error: topologyError, updatedAt, refresh } = useTopology()
  const [selected, setSelected] = useState(null) // { id, href }
  const [detail, setDetail] = useState(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState('')
  const [runningRel, setRunningRel] = useState(null)
  const [lastResponse, setLastResponse] = useState(null)
  const [log, setLog] = useState([])
  const logId = useRef(0)

  const addLog = useCallback((message, tone = 'info') => {
    logId.current += 1
    const entry = { id: logId.current, message, tone, time: new Date().toLocaleTimeString('pt-BR') }
    setLog((prev) => [entry, ...prev].slice(0, MAX_LOG_ENTRIES))
  }, [])

  // Clique no nó: segue o link "self" que o próprio gateway enviou na topologia.
  const loadDetail = useCallback(async (href) => {
    setDetailLoading(true)
    setDetailError('')
    try {
      const result = await followLink({ href, method: 'GET' })
      setDetail(result.data)
      if (!result.ok) setDetailError(result.data?.error ?? `HTTP ${result.status}`)
    } catch (err) {
      setDetailError(describeError(err))
    } finally {
      setDetailLoading(false)
    }
  }, [])

  const handleSelect = useCallback(
    (node) => {
      const href = node._links?.self?.href
      if (!href) return
      setSelected({ id: node.id, href })
      setDetail(null)
      loadDetail(href)
      addLog(`GET ${href}`, 'info')
    },
    [loadDetail, addLog],
  )

  // Executa qualquer link HATEOAS, sem saber o que ele faz.
  const handleAction = useCallback(
    async (action) => {
      setRunningRel(action.rel)
      try {
        const result = await followLink(action)
        setLastResponse({ status: result.status, method: action.method, href: action.href, data: result.data })
        addLog(`${action.title} → HTTP ${result.status}`, result.ok ? 'ok' : result.status === 503 ? 'warn' : 'error')

        // Se a resposta descreve o nó selecionado, ela já traz o novo _links.
        if (result.data?.id && result.data.id === selected?.id) {
          setDetail(result.data)
        } else if (selected) {
          await loadDetail(selected.href)
        }
      } catch (err) {
        addLog(`${action.title} → ${describeError(err)}`, 'error')
      } finally {
        setRunningRel(null)
        refresh()
      }
    },
    [selected, loadDetail, addLog, refresh],
  )

  const meshHealth = topology?.nodes?.find((n) => n.id === 'gateway')?.health

  return (
    <div className="flex h-full min-h-0 flex-col">
      <TopBar
        username={username}
        breakerState={topology?.breaker_state}
        meshHealth={meshHealth}
        lastUpdate={updatedAt?.toLocaleTimeString('pt-BR')}
        onLogout={onLogout}
      />

      <main className="grid min-h-0 flex-1 grid-cols-1 gap-px overflow-y-auto bg-ink-800 lg:grid-cols-[minmax(0,1fr)_400px] lg:grid-rows-[minmax(0,1fr)_18rem] lg:overflow-hidden">
        <section aria-label="Malha de serviços" className="bg-grid relative flex flex-col bg-ink-950 md:min-h-[440px] md:block">
          <div className="pointer-events-none z-10 max-w-sm px-5 pt-5 md:absolute md:left-0 md:top-0">
            <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">Malha de rede</h1>
            <p className="mt-1 text-sm text-ink-400">Gateway → Resilience → Payment · atualização a cada 2s</p>
          </div>
          {topologyError && (
            <p role="alert" className="absolute right-5 top-20 z-10 md:top-5 rounded-lg border border-critical/40 bg-ink-900 px-3 py-2 text-sm text-critical">
              {topologyError}
            </p>
          )}
          <div className="h-[640px] md:h-full">
            {topology ? (
              <MeshGraph topology={topology} selectedId={selected?.id} onSelect={handleSelect} />
            ) : (
              <div className="grid h-full place-items-center font-mono text-sm text-ink-400">conectando ao gateway…</div>
            )}
          </div>
        </section>

        <aside className="flex flex-col bg-ink-900 lg:row-span-2 lg:min-h-0">
          <div className="min-h-[320px] flex-1 overflow-auto lg:min-h-0">
            <ActionPanel detail={detail} loading={detailLoading} error={detailError} runningRel={runningRel} onAction={handleAction} />
          </div>
          <div className="flex max-h-64 min-h-0 flex-col border-t border-ink-800">
            <h2 className="px-5 pb-1 pt-3 font-mono text-[11px] uppercase tracking-[0.2em] text-ink-400">Atividade</h2>
            <ActivityLog entries={log} />
          </div>
        </aside>

        <section aria-label="Última resposta" className="h-72 min-h-0 bg-ink-900 lg:h-auto">
          <ResponseViewer response={lastResponse} />
        </section>
      </main>
    </div>
  )
}
