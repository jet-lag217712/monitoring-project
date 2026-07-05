import { UtilizationBar } from '../common/UtilizationBar.jsx'

const monoTextStyle = {
  fontFamily: "'JetBrains Mono', monospace",
  fontSize: '0.75rem',
}

const mutedCellStyle = {
  color: 'var(--ink-muted)',
  fontSize: '0.8rem',
}

export default function DeviceRow({ ip, device, renderStatus, onClick }) {
  const interactiveProps = onClick
    ? {
        className: 'clickable-row',
        onClick,
        onKeyDown: e => e.key === 'Enter' && onClick(),
        role: 'button',
        tabIndex: 0,
      }
    : {}

  return (
    <tr {...interactiveProps}>
      <td><span className="ups-ip">{ip}</span></td>
      <td style={mutedCellStyle}>{device.hostname ?? '—'}</td>
      <td style={mutedCellStyle}>{device.role ?? '—'}</td>
      <td>{renderStatus(device.status)}</td>
      <td><UtilizationBar pct={device.cpu_pct ?? 0} /></td>
      <td><UtilizationBar pct={device.memory_pct ?? 0} /></td>
      <td>
        <span style={monoTextStyle}>
          {device.uptime_days ?? '—'} days
        </span>
      </td>
      <td>
        <span style={monoTextStyle}>
          {device.latency_ms != null ? `${device.latency_ms} ms` : '—'}
        </span>
      </td>
    </tr>
  )
}
