// Recharts will go here once installed: npm install recharts
// import { LineChart, Line, ResponsiveContainer, Tooltip } from 'recharts'

function StatusBadge({ status }) {
  const map = {
    1: ['ok', 'Healthy'],
    2: ['caution', 'Warning'],
    3: ['alert', 'Critical'],
  }
  const [cls, label] = map[status] ?? ['ok', 'Unknown']
  return <span className={`status-badge ${cls}`}><span className="badge-dot" />{label}</span>
}

function UtilizationBar({ pct }) {
  const cls = pct > 90 ? 'critical' : pct > 75 ? 'low' : ''
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <div className="mini-bar-wrap" style={{ width: 64 }}>
        <div className={`mini-bar ${cls}`} style={{ width: `${pct}%` }} />
      </div>
      <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: '0.72rem' }}>
        {pct}%
      </span>
    </div>
  )
}

export default function SiteDetail({ data, onBack }) {
  if (!data) {
    return (
      <div>
        <button className="back-btn" onClick={onBack}>← All Sites</button>
        <div className="skeleton" style={{ height: 200, borderRadius: 16 }} />
      </div>
    )
  }

  const { location, summary, latest } = data
  const devices = latest?.devices ?? {}

  return (
    <div>
      <button className="back-btn" onClick={onBack}>← All Sites</button>

      <div className="site-detail-header">
        <div>
          <div className="page-eyebrow">
            <span className="eyebrow-dot" />
            Site Detail
          </div>
          <h1 className="page-title">{location}</h1>
          <p className="page-sub">
            {summary?.total_devices ?? '—'} devices · {summary?.online_count ?? '—'} online · {summary?.idf_count ?? '—'} IDFs
          </p>
        </div>
        {summary?.active_alerts > 0 && (
          <span className="status-badge alert" style={{ height: 'fit-content' }}>
            <span className="badge-dot" /> Critical Alerts Active
          </span>
        )}
      </div>

      <div className="ups-table-wrap">
        <table className="ups-table">
          <thead>
            <tr>
              <th>IP Address</th>
              <th>Hostname</th>
              <th>Role</th>
              <th>Status</th>
              <th>CPU</th>
              <th>Memory</th>
              <th>Uptime</th>
              <th>Latency</th>
            </tr>
          </thead>
          <tbody>
            {Object.entries(devices).map(([ip, device]) => (
              <tr key={ip}>
                <td><span className="ups-ip">{ip}</span></td>
                <td style={{ color: 'var(--ink-muted)', fontSize: '0.8rem' }}>{device.hostname ?? '—'}</td>
                <td style={{ color: 'var(--ink-muted)', fontSize: '0.8rem' }}>{device.role ?? '—'}</td>
                <td><StatusBadge status={device.status} /></td>
                <td><UtilizationBar pct={device.cpu_pct ?? 0} /></td>
                <td>
                  <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: '0.75rem' }}>
                    {device.memory_pct ?? '—'}%
                  </span>
                </td>
                <td>
                  <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: '0.75rem' }}>
                    {device.uptime_days ?? '—'} days
                  </span>
                </td>
                <td>
                  <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: '0.75rem' }}>
                    {device.latency_ms != null ? `${device.latency_ms} ms` : '—'}
                  </span>
                </td>
              </tr>
            ))}

            {Object.keys(devices).length === 0 && (
              <tr>
                <td colSpan={7} style={{ textAlign: 'center', color: 'var(--ink-muted)', padding: '32px 16px' }}>
                  No data yet — waiting for first poll
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
