import logoUrl from '../../assets/logo.svg'

export default function Nav({ onLogoClick, sites, dataMode }) {
  const siteCount = sites.length
  const unitCount = sites.reduce((sum, site) => sum + (site.device_count ?? 0), 0)

  return (
    <nav>
      <span className="nav-logo" onClick={onLogoClick}>
        <span className="logo-mark">
          <img src={logoUrl} alt="Equate Logo" />
        </span>
          Network Dashboard
      </span>

      <div className="nav-right">
        <span>{siteCount} sites</span>
        <span className="nav-sep">·</span>
        <span>{unitCount} network devices</span>
      </div>
    </nav>
  )
}
