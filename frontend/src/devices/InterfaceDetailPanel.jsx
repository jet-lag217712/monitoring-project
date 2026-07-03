import InterfaceTrafficChart from '../charts/InterfaceTrafficChart.jsx'
import { formatBytes, formatNumber, formatTimestamp } from '../utils/formatters.js'

function InfoItem({ label, children }) {
  return (
    <div className="metric-item">
      <div className="metric-label">{label}</div>
      <div className="metric-value">{children}</div>
    </div>
  )
}

function PortStatusBadge({ status }) {
  const isUp = status === 'up'
  return (
    <span className={`status-badge ${isUp ? 'ok' : 'alert'}`} style={{ fontSize: '0.58rem' }}>
      <span className="badge-dot" />
      {isUp ? 'Up' : 'Down'}
    </span>
  )
}

export default function InterfaceDetailPanel({ iface }) {
  if (!iface) {
    return (
      <div className="interface-detail-panel">
        <p style={{ color: 'var(--ink-muted)', fontSize: '0.88rem' }}>Select an interface to view details.</p>
      </div>
    )
  }

  return (
    <div className="interface-detail-panel">
      <div className="chart-card-label" style={{ marginBottom: 16 }}>Interface Detail</div>
      <div className="device-info-grid">
        <InfoItem label="Interface Name">{iface.name ?? '—'}</InfoItem>
        <InfoItem label="Administrative Status">
          <PortStatusBadge status={iface.admin_status} />
        </InfoItem>
        <InfoItem label="Operational Status">
          <PortStatusBadge status={iface.oper_status} />
        </InfoItem>
        <InfoItem label="Port Speed">{iface.speed ?? '—'}</InfoItem>
        <InfoItem label="Duplex">{iface.duplex ?? '—'}</InfoItem>
        <InfoItem label="Port Utilization">
          {iface.utilization_pct != null ? `${iface.utilization_pct}%` : '—'}
        </InfoItem>
        <InfoItem label="Bytes In">{formatBytes(iface.bytes_in)}</InfoItem>
        <InfoItem label="Bytes Out">{formatBytes(iface.bytes_out)}</InfoItem>
        <InfoItem label="Packets In">{formatNumber(iface.packets_in)}</InfoItem>
        <InfoItem label="Packets Out">{formatNumber(iface.packets_out)}</InfoItem>
        <InfoItem label="Errors In">{formatNumber(iface.errors_in)}</InfoItem>
        <InfoItem label="Errors Out">{formatNumber(iface.errors_out)}</InfoItem>
        <InfoItem label="Last Status Change">{formatTimestamp(iface.last_status_change)}</InfoItem>
        <InfoItem label="Description">{iface.description ?? '—'}</InfoItem>
      </div>

      <div style={{ marginTop: 20 }}>
        <InterfaceTrafficChart data={iface.traffic_history} interfaceName={iface.name} />
      </div>
    </div>
  )
}
