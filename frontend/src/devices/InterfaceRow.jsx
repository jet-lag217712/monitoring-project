import { UtilizationBar } from '../common/UtilizationBar.jsx'
import { getInterfaceSelectionKey } from '../utils/deviceData.js'

const monoTextStyle = {
  fontFamily: "'JetBrains Mono', monospace",
  fontSize: '0.75rem',
}

export default function InterfaceRow({ iface, selected, onSelect }) {
  const key = getInterfaceSelectionKey(iface)
  const isSelected = selected === key

  const isUp = String(iface.status ?? iface.oper_status ?? '').toLowerCase() === 'up'

  const handleActivate = () => onSelect?.(key)

  return (
    <tr
      className={`clickable-row${isSelected ? ' selected' : ''}`}
      onClick={handleActivate}
      onKeyDown={e => e.key === 'Enter' && handleActivate()}
      role="button"
      tabIndex={0}
      aria-selected={isSelected}
    >
      <td>
        <span style={monoTextStyle}>{iface.name}</span>
      </td>
      <td>
        <span className={`status-badge ${isUp ? 'ok' : 'alert'}`} style={{ fontSize: '0.58rem' }}>
          <span className="badge-dot" />
          {isUp ? 'Up' : 'Down'}
        </span>
      </td>
      <td>
        <UtilizationBar pct={iface.utilization_pct ?? 0} />
      </td>
      <td style={{ color: 'var(--ink-muted)', fontSize: '0.8rem' }}>{iface.speed ?? '—'}</td>
    </tr>
  )
}
