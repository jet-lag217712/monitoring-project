import { UtilizationBar } from '../common/UtilizationBar.jsx'
import { formatSysUpTime } from '../utils/formatters.js'

function InfoItem({ label, snmpKey, spanFull, children }) {
  return (
    <div className={`metric-item${spanFull ? ' metric-item-span-full' : ''}`}>
      <div className="metric-label">
        {label}
        {snmpKey && <span className="metric-snmp-key">{snmpKey}</span>}
      </div>
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

export default function DeviceInfoCard({ device, ip }) {
  return (
    <div className="device-info-card">
      <div className="device-info-card-header">
        <div className="chart-card-label">Device Information</div>
      </div>
      <div className="device-info-grid">
        <InfoItem label="System Name" snmpKey="sysName">
          {device.snmp?.sysName ?? '—'}
        </InfoItem>
        <InfoItem label="System Uptime" snmpKey="sysUpTime">
          {formatSysUpTime(device.snmp?.sysUpTime)}
        </InfoItem>
        <InfoItem label="System Description" snmpKey="sysDescr" spanFull>
          {device.snmp?.sysDescr ?? '—'}
        </InfoItem>
        <InfoItem label="IP Address">
          <span className="ups-ip">{ip}</span>
        </InfoItem>
        <InfoItem label="CPU Utilization">
          <UtilizationBar pct={device.cpu_pct ?? 0} />
        </InfoItem>
        <InfoItem label="Memory Utilization">
          {device.memory_pct != null ? `${device.memory_pct}%` : '—'}
        </InfoItem>
        <InfoItem label="Device Temperature">
          {device.temperature_c != null ? `${device.temperature_c}°C` : '—'}
        </InfoItem>
        <InfoItem label="Interface Count">
          {device.interface_count ?? '—'}
        </InfoItem>
        <InfoItem label="Active Interface Count">
          {device.active_interface_count ?? '—'}
        </InfoItem>
        <InfoItem label="Administrative Status">
          <PortStatusBadge status={device.admin_status} />
        </InfoItem>
        <InfoItem label="Operational Status">
          <PortStatusBadge status={device.oper_status} />
        </InfoItem>
      </div>
    </div>
  )
}
