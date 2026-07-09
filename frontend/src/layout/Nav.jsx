import logoUrl from '../../assets/logo.svg'

export default function Nav({ onLogoClick, sites, dataMode, user, onSignOut }) {
  const siteCount = sites.length
  const unitCount = sites.reduce((sum, site) => sum + (site.device_count ?? 0), 0)

  return (
    <nav className="app-nav">
      <span className="nav-logo" onClick={onLogoClick}>
        <span className="logo-mark">
          <img src={logoUrl} alt="Equate Logo" />
        </span>
        Equate
      </span>

      <div className="nav-right">
        {user?.email && (
          <>
            <span className="nav-user" title={user.email}>
              {user.name || user.email}
            </span>
            <button type="button" className="nav-sign-out" onClick={onSignOut}>
              Log out
            </button>
          </>
        )}
      </div>
    </nav>
  )
}
