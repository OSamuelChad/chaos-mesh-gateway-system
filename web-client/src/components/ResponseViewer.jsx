/** Exibe o último JSON retornado pelo gateway, destacando o nó _links. */
export default function ResponseViewer({ response }) {
  if (!response) {
    return <p className="p-5 text-sm text-ink-400">As respostas das ações aparecem aqui.</p>
  }
  const statusColor = response.status < 300 ? 'text-healthy' : response.status === 503 ? 'text-degraded' : 'text-critical'

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-3 border-b border-ink-700 px-5 py-2.5 font-mono text-xs">
        <span className={`font-bold ${statusColor}`}>{response.status}</span>
        <span className="text-ink-300">{response.method}</span>
        <span className="truncate text-ink-400">{response.href}</span>
      </div>
      <pre className="min-h-0 flex-1 overflow-auto p-5 font-mono text-[11px] leading-relaxed text-ink-300">{JSON.stringify(response.data, null, 2)}</pre>
    </div>
  )
}
