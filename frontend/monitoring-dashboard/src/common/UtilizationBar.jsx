export function getUtilizationClass(value) {
  return value > 90 ? 'critical' : value > 75 ? 'low' : ''
}

export function MiniBar({ value }) {
  return (
    <div className="mini-bar-wrap">
      <div className={`mini-bar ${getUtilizationClass(value)}`} style={{ width: `${value}%` }} />
    </div>
  )
}

export function UtilizationBar({ pct }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <div className="mini-bar-wrap" style={{ width: 64 }}>
        <div className={`mini-bar ${getUtilizationClass(pct)}`} style={{ width: `${pct}%` }} />
      </div>
      <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: '0.72rem' }}>
        {pct}%
      </span>
    </div>
  )
}
