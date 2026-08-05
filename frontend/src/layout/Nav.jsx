import logoUrl from '../../assets/logo.svg'
import SearchBar from '../common/SearchBar.jsx'

export default function Nav({
  onLogoClick,
  user,
  onSignOut,
  searchQuery,
  onSearchQueryChange,
  onSearchFocus,
  onSearchClear,
}) {
  return (
    <nav className="app-nav">
      <span className="nav-logo" onClick={onLogoClick}>
        <span className="logo-mark">
          <img src={logoUrl} alt="Equate Logo" />
        </span>
        Equate
      </span>

      <SearchBar
        variant="nav"
        value={searchQuery}
        onChange={onSearchQueryChange}
        onFocus={onSearchFocus}
        onClear={onSearchClear}
        placeholder="Search sites, hostname, or IP…"
        id="nav-search"
      />

      <div className="nav-right">
        {user && (
          <>
            {(user.name || user.email) && (
              <span className="nav-user" title={user.email || user.name}>
                {user.name || user.email}
              </span>
            )}
            <button type="button" className="nav-sign-out" onClick={onSignOut}>
              Log out
            </button>
          </>
        )}
      </div>
    </nav>
  )
}
