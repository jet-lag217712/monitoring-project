import { DeviceStatusBadge } from '../common/StatusBadge.jsx'
import { UtilizationBar } from '../common/UtilizationBar.jsx'
import { formatUptime } from '../utils/formatters.js'

function InfoItem({ label, children }) {
  return (
    <div className="metric-item">
      <div className="metric-label">{label}</div>
      <div className="metric-value">{children}</div>
    </div>
  )
}

export default function DeviceInfoCard({ device, ip }) {
  const psu1 = device.power_supply?.psu1_v
  const psu2 = device.power_supply?.psu2_v
  const psuText =
    psu1 != null && psu2 != null ? `PSU1: ${psu1}V · PSU2: ${psu2}V` : '—'

  return (
    <div className="device-info-card">
      <div className="device-info-card-header">
        <div className="chart-card-label">Device Information</div>
      </div>
      <div className="device-info-grid">
        <InfoItem label="Device Name">{device.name ?? device.hostname ?? '—'}</InfoItem>
        <InfoItem label="Hostname">{device.hostname ?? '—'}</InfoItem>
        <InfoItem label="IP Address">
          <span className="ups-ip">{ip}</span>
        </InfoItem>
        <InfoItem label="Vendor">{device.vendor ?? '—'}</InfoItem>
        <InfoItem label="Model">{device.model ?? '—'}</InfoItem>
        <InfoItem label="Serial Number">{device.serial_number ?? '—'}</InfoItem>
        <InfoItem label="Role">{device.role ?? '—'}</InfoItem>
        <InfoItem label="Uptime">{formatUptime(device.uptime_days)}</InfoItem>
        <InfoItem label="CPU Utilization">
          <UtilizationBar pct={device.cpu_pct ?? 0} />
        </InfoItem>
        <InfoItem label="RAM Utilization">
          {device.memory_pct != null ? `${device.memory_pct}%` : '—'}
        </InfoItem>
        <InfoItem label="Device Temperature">
          {device.temperature_c != null ? `${device.temperature_c}°C` : '—'}
        </InfoItem>
        <InfoItem label="Latency">
          {device.latency_ms != null ? `${device.latency_ms} ms` : '—'}
        </InfoItem>
        <InfoItem label="Power Supply">{psuText}</InfoItem>
        <InfoItem label="Overall Status">
          <DeviceStatusBadge status={device.status} />
        </InfoItem>
      </div>
    </div>
  )
}
