import { memo, useEffect } from 'react'
import { Handle, Position, useUpdateNodeInternals } from '@xyflow/react'
import HealthBadge from './HealthBadge.jsx'
import { healthStyle } from './health.js'

/** Nó customizado do React Flow: cor, brilho e pulso refletem a saúde do serviço. */
function ServiceNode({ id, data, selected }) {
  const updateNodeInternals = useUpdateNodeInternals()
  // Os handles mudam de lado no layout vertical: o React Flow precisa remedi-los.
  useEffect(() => updateNodeInternals(id), [id, data.narrow, updateNodeInternals])

  const style = healthStyle(data.health)
  const isCritical = data.health === 'CRITICAL'

  return (
    <div className="relative">
      {isCritical && (
        <span aria-hidden className="absolute inset-0 animate-pulse-ring rounded-2xl border-2" style={{ borderColor: style.color }} />
      )}
      <div
        className={`relative w-64 rounded-2xl border-2 bg-ink-900/95 p-4 transition-[box-shadow,transform] duration-300 ${selected ? 'scale-[1.03]' : ''}`}
        style={{
          borderColor: style.color,
          boxShadow: `0 0 0 ${selected ? 4 : 0}px color-mix(in srgb, ${style.color} 25%, transparent), 0 18px 50px -12px color-mix(in srgb, ${style.color} 45%, transparent)`,
        }}
      >
        <Handle type="target" position={data.narrow ? Position.Top : Position.Left} className="!size-2.5 !border-2 !border-ink-900" style={{ background: style.color }} />
        <div className="flex items-center justify-between gap-2">
          <HealthBadge health={data.health} />
          {!data.reachable && <span className="font-mono text-[10px] uppercase text-critical">offline</span>}
        </div>
        <p className="mt-2.5 text-lg font-semibold leading-tight">{data.label}</p>
        <p className="font-mono text-[10px] uppercase tracking-wider text-ink-400">{data.role}</p>
        <dl className="mt-3 grid grid-cols-3 gap-2 border-t border-ink-700 pt-3">
          {data.summary?.map((metric) => (
            <div key={metric.label} className="min-w-0">
              <dt className="truncate text-[10px] text-ink-400">{metric.label}</dt>
              <dd className={`truncate font-mono text-xs font-bold ${style.text}`}>{metric.value}</dd>
            </div>
          ))}
        </dl>
        <Handle type="source" position={data.narrow ? Position.Bottom : Position.Right} className="!size-2.5 !border-2 !border-ink-900" style={{ background: style.color }} />
      </div>
    </div>
  )
}

export default memo(ServiceNode)
