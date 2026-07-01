import AppShell from './layout/AppShell.jsx'
import DashboardPage from './dashboard/DashboardPage.jsx'
import { useNetworkDashboard } from './hooks/useNetworkDashboard.js'
import './index.css'

export default function App() {
  const dashboard = useNetworkDashboard()

  return (
    <AppShell dashboard={dashboard}>
      <DashboardPage dashboard={dashboard} />
    </AppShell>
  )
}
