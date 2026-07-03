import InterfaceRow from '../devices/InterfaceRow.jsx'

export default function InterfacesTable({ interfaces, selectedKey, onSelect }) {
  const entries = interfaces ?? []

  return (
    <div className="ups-table-wrap">
      <table className="ups-table">
        <thead>
          <tr>
            <th>Interface</th>
            <th>Status</th>
            <th>Utilization</th>
            <th>Speed</th>
          </tr>
        </thead>
        <tbody>
          {entries.map(iface => (
            <InterfaceRow
              key={iface.if_index ?? iface.name}
              iface={iface}
              selected={selectedKey}
              onSelect={onSelect}
            />
          ))}

          {entries.length === 0 && (
            <tr>
              <td colSpan={4} style={{ textAlign: 'center', color: 'var(--ink-muted)', padding: '32px 16px' }}>
                No interfaces reported
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
