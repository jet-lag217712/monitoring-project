import { UtilizationBar } from '../common/UtilizationBar.jsx'
import { formatSysUpTime } from '../utils/formatters.js'

function InfoItem({ label, snmpKey, span2, spanFull, rowStart, children }) {
  const spanClass = spanFull
    ? ' metric-item-span-full'
    : span2
      ? ' metric-item-span-2'
      : rowStart
        ? ' metric-item-row-start'
        : ''

  return (
    <div className={`metric-item${spanClass}`}>
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

function formatIds(ids) {
  if (!Array.isArray(ids) || ids.length === 0) return '—'
  return ids.join(', ')
}

function ComponentList({ title, components, valueSuffix }) {
  if (!Array.isArray(components) || components.length === 0) return null
  return (
    <div className="device-component-panel">
      <div className="chart-card-label">{title}</div>
      <div className="device-info-grid">
        {components.map(component => (
          <InfoItem key={component.component_id} label={component.name || component.component_id} span2>
            <span>
              {component.value != null ? `${component.value}${valueSuffix || (component.unit ? ` ${component.unit}` : '')}` : '—'}
              {component.status ? ` · ${component.status}` : ''}
            </span>
          </InfoItem>
        ))}
      </div>
    </div>
  )
}

export default function DeviceInfoCard({ device, ip }) {
  const cpuDisplay = device.cpu_pct != null ? device.cpu_pct : null
  const memDisplay = device.memory_pct != null ? device.memory_pct : null

  return (
    <div className="device-info-card">
      <div className="device-info-card-header">
        <div className="chart-card-label">Device Information</div>
      </div>
      <div className="device-info-grid">
        <InfoItem label="System Name" snmpKey="sysName" span2>
          {device.snmp?.sysName ?? '—'}
        </InfoItem>
        <InfoItem label="System Description" snmpKey="sysDescr" span2>
          {device.snmp?.sysDescr ?? '—'}
        </InfoItem>
        <InfoItem label="sysObjectID" snmpKey="sysObjectID" span2>
          {device.snmp?.sysObjectID ?? '—'}
        </InfoItem>
        <InfoItem label="System Uptime" snmpKey="sysUpTime">
          {formatSysUpTime(device.snmp?.sysUpTime)}
        </InfoItem>
        <InfoItem label="IP Address">
          <span className="ups-ip">{ip}</span>
        </InfoItem>
        <InfoItem label="Vendor / Model" span2>
          {[device.vendor, device.model].filter(Boolean).join(' ') || '—'}
        </InfoItem>
        <InfoItem label="Serial">
          {device.serial_number ?? '—'}
        </InfoItem>
        <InfoItem label="Profile">
          {device.profile ?? '—'}
        </InfoItem>
        <InfoItem label="Interface Count">
          {device.interface_count ?? '—'}
        </InfoItem>
        <InfoItem label="Active Interface Count">
          {device.active_interface_count ?? '—'}
        </InfoItem>
        <InfoItem label="CPU Utilization">
          {cpuDisplay != null ? <UtilizationBar pct={cpuDisplay} /> : '—'}
        </InfoItem>
        <InfoItem label="Memory Utilization">
          {memDisplay != null ? <UtilizationBar pct={memDisplay} /> : '—'}
        </InfoItem>
        <InfoItem label="Device Temperature">
          {device.temperature_c != null ? `${device.temperature_c}°C` : '—'}
        </InfoItem>
        <InfoItem label="Health Reason" span2>
          {device.status_reason ?? '—'}
        </InfoItem>
        <InfoItem label="Upstream Devices" span2>
          {formatIds(device.upstream_device_ids)}
        </InfoItem>
        <InfoItem label="Unavailable Upstreams" span2>
          {formatIds(device.unavailable_upstream_device_ids)}
        </InfoItem>
        <InfoItem label="Root Cause Devices" span2>
          {formatIds(device.root_cause_device_ids)}
        </InfoItem>
        <InfoItem label="Administrative Status" rowStart>
          <PortStatusBadge status={device.admin_status} />
        </InfoItem>
        <InfoItem label="Operational Status">
          <PortStatusBadge status={device.oper_status} />
        </InfoItem>
      </div>

      <ComponentList title="Temperature Components" components={device.temperature_components} valueSuffix="°C" />
      <ComponentList title="Power Components" components={device.power_components} />
    </div>
  )
}
