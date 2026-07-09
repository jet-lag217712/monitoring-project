import AppShell from './layout/AppShell.jsx'
import DashboardPage from './dashboard/DashboardPage.jsx'
import SignInPage from './auth/SignInPage.jsx'
import { useAuth } from './hooks/useAuth.js'
import { useNetworkDashboard } from './hooks/useNetworkDashboard.js'
import './index.css'

export default function App() {
  const auth = useAuth()
  const dashboard = useNetworkDashboard({
    enabled: auth.isAuthenticated,
    onUnauthorized: auth.signOut,
  })

  if (!auth.isAuthenticated) {
    return <SignInPage auth={auth} />
  }

  return (
    <AppShell dashboard={dashboard} auth={auth}>
      <DashboardPage dashboard={dashboard} />
    </AppShell>
  )
}
