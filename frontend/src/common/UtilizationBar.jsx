export function getUtilizationClass(value) {
  return value > 90 ? 'critical' : value > 75 ? 'low' : ''
}

function clampPercent(pct) {
  const n = Number(pct)
  if (Number.isNaN(n)) return 0
  return Math.min(100, Math.max(0, n))
}

export function MiniBar({ value }) {
  const pct = clampPercent(value)
  return (
    <div className="mini-bar-wrap">
      <div className={`mini-bar ${getUtilizationClass(pct)}`} style={{ width: `${pct}%` }} />
    </div>
  )
}

export function UtilizationBar({ pct }) {
  const value = clampPercent(pct)
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <div className="mini-bar-wrap" style={{ width: 64 }}>
        <div className={`mini-bar ${getUtilizationClass(value)}`} style={{ width: `${value}%` }} />
      </div>
      <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: '0.72rem' }}>
        {value.toFixed(2)}%
      </span>
    </div>
  )
}
