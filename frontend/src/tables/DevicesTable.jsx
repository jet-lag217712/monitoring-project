import DeviceRow from '../devices/DeviceRow.jsx'

export default function DevicesTable({ devices, renderStatus, onDeviceClick }) {
  const entries = Object.entries(devices)

  return (
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
          {entries.map(([ip, device]) => (
            <DeviceRow
              key={ip}
              ip={ip}
              device={device}
              renderStatus={renderStatus}
              onClick={onDeviceClick ? () => onDeviceClick(ip) : undefined}
            />
          ))}

          {entries.length === 0 && (
            <tr>
              <td colSpan={8} style={{ textAlign: 'center', color: 'var(--ink-muted)', padding: '32px 16px' }}>
                No data yet — waiting for first poll
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
