import AlertBanner from '../alerts/AlertBanner.jsx'
import Nav from './Nav.jsx'

export default function AppShell({ children, dashboard, auth }) {
  const hasAlerts = dashboard.alerts.length > 0
  const contentTopPad = hasAlerts ? '124px' : '88px'

  return (
    <div className="app-layout">
      <Nav
        onLogoClick={dashboard.handleBack}
        sites={dashboard.sites}
        dataMode={dashboard.dataMode}
        user={auth?.user}
        onSignOut={auth?.signOut}
      />
      <AlertBanner alerts={dashboard.alerts} />

      <main className="page-content" style={{ paddingTop: contentTopPad }}>
        {children}
      </main>
    </div>
  )
}
