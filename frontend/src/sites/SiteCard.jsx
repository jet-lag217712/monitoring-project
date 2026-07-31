import { SiteStatusBadge } from '../common/StatusBadge.jsx'
import { MiniBar } from '../common/UtilizationBar.jsx'
import { formatPercent } from '../utils/formatters.js'

export default function SiteCard({ site, onClick }) {
  const {
    location,
    status = 'ok',
    device_count = 0,
    online_count = 0,
    avg_cpu = 0,
    avg_memory = 0,
    active_alerts = 0,
    warning_count = 0,
    critical_count = 0,
    unknown_count = 0,
    dependency_impacted_count = 0,
    site_dependency_impacted = false,
    root_cause_site_ids = [],
    unavailable_upstream_site_ids = [],
  } = site

  const rootCauseLabel = root_cause_site_ids.length > 0
    ? root_cause_site_ids.join(', ')
    : null

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
          {site_dependency_impacted && rootCauseLabel && (
            <div className="site-dependency-note" style={{ color: 'var(--status-unknown)', fontSize: '0.68rem', marginTop: '0.25rem' }}>
              Impacted by {rootCauseLabel}
            </div>
          )}
        </div>
        <SiteStatusBadge status={status} />
      </div>

      <div className="site-metrics">
        <div className="metric-item">
          <div className="metric-label">Avg CPU</div>
          <div className="metric-value">{formatPercent(avg_cpu)}%</div>
          <MiniBar value={avg_cpu} />
        </div>
        <div className="metric-item">
          <div className="metric-label">Avg memory</div>
          <div className="metric-value">{formatPercent(avg_memory)}%</div>
          <MiniBar value={avg_memory} />
        </div>
        <div className="metric-item">
          <div className="metric-label">Devices online</div>
          <div className="metric-value">{online_count} / {device_count}</div>
        </div>
        <div className="metric-item">
          <div className="metric-label">Health</div>
          <div className="metric-value" style={{ fontSize: '0.72rem' }}>
            {critical_count > 0 && (
              <span style={{ color: 'var(--status-alert)' }}>{critical_count} crit </span>
            )}
            {warning_count > 0 && (
              <span style={{ color: 'var(--status-caution)' }}>{warning_count} warn </span>
            )}
            {unknown_count > 0 && (
              <span style={{ color: 'var(--status-unknown)' }}>{unknown_count} unk </span>
            )}
            {critical_count === 0 && warning_count === 0 && unknown_count === 0 && 'Healthy'}
          </div>
        </div>
        <div className="metric-item">
          <div className="metric-label">Dependency impact</div>
          <div className="metric-value">{dependency_impacted_count || 'None'}</div>
        </div>
        <div className="metric-item">
          <div className="metric-label">Active alerts</div>
          <div className="metric-value" style={{ color: status === 'alert' ? 'var(--status-alert)' : 'inherit' }}>
            {active_alerts > 0 ? active_alerts : 'None'}
          </div>
        </div>
      </div>

      <div className="site-card-footer">
        <span className="card-arrow">→</span>
      </div>
    </div>
  )
}
