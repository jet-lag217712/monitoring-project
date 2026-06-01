import logoUrl from '../assets/logo.svg'

export default function Nav({ onLogoClick, siteCount, unitCount, dataMode }) {
  return (
    <nav>
      <span className="nav-logo" onClick={onLogoClick}>
        <span className="logo-mark">
          <img src={logoUrl} alt="Equate Network Dashboard logo" />
        </span>
        Equate Network Dashboard
      </span>

      <div className="nav-right">
        <span>{siteCount} sites</span>
        <span className="nav-sep">·</span>
        <span>{unitCount} network devices</span>
        <span className="nav-sep">·</span>
        <span style={{ color: dataMode === 'live' ? 'var(--status-ok)' : 'var(--status-caution)' }}>
          ● {dataMode}
        </span>
      </div>
    </nav>
  )
}
