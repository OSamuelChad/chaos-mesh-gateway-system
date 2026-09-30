import { healthStyle } from './health.js'

export default function HealthBadge({ health, size = 'sm' }) {
  const style = healthStyle(health)
  const padding = size === 'lg' ? 'px-3 py-1 text-xs' : 'px-2 py-0.5 text-[10px]'
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full border font-mono font-bold uppercase tracking-widest ${padding} ${style.text} ${style.bg} ${style.border}`}>
      <span className="size-1.5 rounded-full" style={{ backgroundColor: style.color }} />
      {style.label}
    </span>
  )
}
