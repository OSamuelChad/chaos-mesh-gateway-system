/** Mapeamento visual de saúde → cores (verde, amarelo, vermelho). */
export const HEALTH = {
  HEALTHY: { label: 'Healthy', color: 'var(--color-healthy)', text: 'text-healthy', bg: 'bg-healthy/10', border: 'border-healthy/40' },
  DEGRADED: { label: 'Degraded', color: 'var(--color-degraded)', text: 'text-degraded', bg: 'bg-degraded/10', border: 'border-degraded/50' },
  CRITICAL: { label: 'Critical', color: 'var(--color-critical)', text: 'text-critical', bg: 'bg-critical/10', border: 'border-critical/60' },
  UNKNOWN: { label: 'Unknown', color: 'var(--color-ink-400)', text: 'text-ink-300', bg: 'bg-ink-700/40', border: 'border-ink-600' },
}

export function healthStyle(health) {
  return HEALTH[health] ?? HEALTH.UNKNOWN
}
