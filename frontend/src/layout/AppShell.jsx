import AlertBanner from '../alerts/AlertBanner.jsx'
import SearchPopover from '../dashboard/SearchPopover.jsx'
import { APP_VERSION } from '../config/version.js'
import Nav from './Nav.jsx'

export default function AppShell({ children, dashboard, auth }) {
  const hasAlerts = dashboard.alerts.length > 0
  const contentTopPad = hasAlerts ? '124px' : '88px'

  return (
    <div className="app-layout">
      <Nav
        onLogoClick={dashboard.handleBack}
        user={auth?.user}
        onSignOut={auth?.signOut}
        searchQuery={dashboard.searchQuery}
        onSearchQueryChange={value => {
          dashboard.setSearchQuery(value)
          if (!dashboard.searchOpen) {
            dashboard.openSearch()
          }
        }}
        onSearchFocus={dashboard.openSearch}
        onSearchClear={dashboard.closeSearch}
      />
      <AlertBanner alerts={dashboard.alerts} />

      {dashboard.searchOpen ? (
        <SearchPopover
          query={dashboard.searchQuery}
          onClose={dashboard.closeSearch}
          siteHits={dashboard.searchSiteHits}
          deviceHits={dashboard.searchDeviceHits}
          loading={dashboard.searchLoading}
          onSelectSite={dashboard.handleSiteClick}
          onSelectDevice={dashboard.handleSearchDeviceSelect}
        />
      ) : null}

      <main className="page-content" style={{ paddingTop: contentTopPad }}>
        {children}
      </main>

      <footer className="app-footer">v{APP_VERSION}</footer>
    </div>
  )
}
