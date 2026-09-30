const TONE = { ok: 'bg-healthy', warn: 'bg-degraded', error: 'bg-critical', info: 'bg-signal' }

export default function ActivityLog({ entries }) {
  if (entries.length === 0) {
    return <p className="p-5 text-sm text-ink-400">Nenhum evento ainda.</p>
  }
  return (
    <ol className="divide-y divide-ink-800 overflow-auto">
      {entries.map((entry) => (
        <li key={entry.id} className="flex items-start gap-3 px-5 py-2.5">
          <span className={`mt-1.5 size-1.5 shrink-0 rounded-full ${TONE[entry.tone] ?? TONE.info}`} />
          <div className="min-w-0 flex-1">
            <p className="text-sm">{entry.message}</p>
            <p className="font-mono text-[10px] text-ink-400">{entry.time}</p>
          </div>
        </li>
      ))}
    </ol>
  )
}
