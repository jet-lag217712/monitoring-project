function StatusBadge({ status }) {
  const labels = { ok: 'OK', caution: 'Caution', alert: 'Alert' }
  return (
    <span className={`status-badge ${status}`}>
      <span className="badge-dot" />
      {labels[status] ?? status}
    </span>
  )
}

function MiniBar({ value }) {
  const cls = value > 90 ? 'critical' : value > 75 ? 'low' : ''
  return (
    <div className="mini-bar-wrap">
      <div className={`mini-bar ${cls}`} style={{ width: `${value}%` }} />
    </div>
  )
}

export default function SiteCard({ site, onClick }) {
  const {
    location,
    type = 'Campus',
    status = 'ok',
    idf_count = 0,
    device_count = 0,
    online_count = 0,
    avg_cpu = 0,
    avg_memory = 0,
    active_alerts = 0,
  } = site

  return (
    <div
      className={`site-card status-${status}`}
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={e => e.key === 'Enter' && onClick?.()}
    >
      <div className="site-card-header">
        <div>
          <div className="site-name">{location}</div>
          <div className="site-type">{type}</div>
        </div>
        <StatusBadge status={status} />
      </div>

      <div className="site-metrics">
        <div className="metric-item">
          <div className="metric-label">Avg CPU</div>
          <div className="metric-value">{avg_cpu}%</div>
        </div>
        <div className="metric-item">
          <div className="metric-label">Avg memory</div>
          <div className="metric-value">{avg_memory}%</div>
        </div>
        <div className="metric-item">
          <div className="metric-label">Devices online</div>
          <div className="metric-value">{online_count} / {device_count}</div>
        </div>
        <div className="metric-item">
          <div className="metric-label">Active alerts</div>
          <div className="metric-value" style={{ color: status === 'alert' ? 'var(--status-alert)' : 'inherit' }}>
            {active_alerts > 0 ? active_alerts : 'None'}
          </div>
        </div>
      </div>

      <MiniBar value={avg_cpu} />

      <div className="site-card-footer">
        <span className="ups-count">{idf_count} IDFs</span>
        <span className="card-arrow">→</span>
      </div>
    </div>
  )
}
