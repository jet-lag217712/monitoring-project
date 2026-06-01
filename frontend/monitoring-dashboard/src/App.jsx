import { useCallback, useDeferredValue, useEffect, useState } from 'react'
import Nav from './Nav.jsx'
import AlertBanner from './AlertBanner.jsx'
import SitesGrid from './SitesGrid.jsx'
import SiteDetail from './SiteDetail.jsx'
import { demoModes, mockScenarios, mockSiteDetails, mockSitesResponse, mockTestConfig } from './mockData.js'
import './index.css'

const POLL_INTERVAL_MS = 5000
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:8000'

function apiUrl(path) {
  return new URL(path, API_BASE_URL).toString()
}

function normalizeSites(data) {
  return Object.entries(data).map(([siteId, value]) => ({
    site_id: siteId,
    location: value.location,
    type: value.type,
    status: value.status?.toLowerCase() ?? 'ok',
    ...value.latest?.summary,
  }))
}

function buildAlerts(list) {
  return list
    .filter(site => site.status !== 'ok')
    .map(site => ({
      site: site.location,
      message: site.status === 'alert' ? 'Critical device issue' : 'Performance warning',
    }))
}

export default function App() {
  const [sites, setSites] = useState(() => normalizeSites(mockSitesResponse))
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedSite, setSelectedSite] = useState(null)
  const [siteDetail, setSiteDetail] = useState(null)
  const [lastUpdated, setLastUpdated] = useState(null)
  const [alerts, setAlerts] = useState(() => buildAlerts(normalizeSites(mockSitesResponse)))
  const [dataMode, setDataMode] = useState('demo')
  const [demoScenarioId, setDemoScenarioId] = useState('all-healthy')
  const deferredSearchQuery = useDeferredValue(searchQuery)

  const applyDemoScenario = useCallback((scenarioId, siteId = selectedSite) => {
    const scenario = mockScenarios[scenarioId] ?? mockScenarios['all-healthy']
    const list = normalizeSites(scenario.sites)

    setSites(list)
    setAlerts(buildAlerts(list))
    setLastUpdated(new Date().toLocaleTimeString())
    setDataMode('demo')

    if (siteId) {
      setSiteDetail(scenario.details[siteId] ?? null)
    }
  }, [selectedSite])

  const fetchSites = useCallback(async () => {
    try {
      const res = await fetch(apiUrl('/api/sites'))
      if (!res.ok) {
        throw new Error(`Site list request failed with ${res.status}`)
      }

      const data = await res.json()
      const list = normalizeSites(data)
      setSites(list)
      setAlerts(buildAlerts(list))
      setDataMode('live')
      setLastUpdated(new Date().toLocaleTimeString())
    } catch (err) {
      console.error('Failed to fetch sites:', err)
      applyDemoScenario(demoScenarioId)
    }
  }, [applyDemoScenario, demoScenarioId])

  const fetchSiteDetail = useCallback(async siteId => {
    try {
      const res = await fetch(apiUrl(`/api/sites/${siteId}`))
      if (!res.ok) {
        throw new Error(`Site detail request failed with ${res.status}`)
      }

      const data = await res.json()
      setSiteDetail(data)
      setDataMode('live')
    } catch (err) {
      console.error('Failed to fetch site detail:', err)
      setSiteDetail(
        mockScenarios[demoScenarioId]?.details[siteId] ?? mockSiteDetails[siteId] ?? {
          site_id: siteId,
          location: siteId,
          summary: {},
          latest: { devices: {} },
        },
      )
      setDataMode('demo')
    }
  }, [demoScenarioId])

  const fetchTestConfig = useCallback(async () => {
    try {
      const res = await fetch(apiUrl('/api/test-config'))
      if (!res.ok) {
        throw new Error(`Test config request failed with ${res.status}`)
      }

      await res.json()
      setDataMode('live')
    } catch (err) {
      console.error('Failed to fetch test config:', err)
      void mockTestConfig
      setDataMode('demo')
    }
  }, [])

  useEffect(() => {
    fetchTestConfig()
    fetchSites()
    if (selectedSite) {
      fetchSiteDetail(selectedSite)
    }

    const id = setInterval(() => {
      fetchTestConfig()
      fetchSites()
      if (selectedSite) {
        fetchSiteDetail(selectedSite)
      }
    }, POLL_INTERVAL_MS)

    return () => clearInterval(id)
  }, [fetchSites, fetchSiteDetail, fetchTestConfig, selectedSite])

  const handleSiteClick = siteId => {
    setSelectedSite(siteId)
    setSiteDetail(null)
    fetchSiteDetail(siteId)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleBack = () => {
    setSelectedSite(null)
    setSiteDetail(null)
  }

  const handleDemoScenarioChange = scenarioId => {
    setDemoScenarioId(scenarioId)
    applyDemoScenario(scenarioId)
  }

  const hasAlerts = alerts.length > 0
  const contentTopPad = hasAlerts ? '124px' : '88px'
  const normalizedSearchQuery = deferredSearchQuery.trim().toLowerCase()
  const visibleSites = normalizedSearchQuery
    ? sites.filter(site => {
        const searchableText = [
          site.location,
          site.type,
          site.site_id,
          site.status,
        ]
          .filter(Boolean)
          .join(' ')
          .toLowerCase()

        return searchableText.includes(normalizedSearchQuery)
      })
    : sites

  return (
    <div className="app-layout">
      <Nav
        onLogoClick={handleBack}
        siteCount={sites.length}
        unitCount={sites.reduce((sum, site) => sum + (site.device_count ?? 0), 0)}
        dataMode={dataMode}
      />
      <AlertBanner alerts={alerts} />

      <main className="page-content" style={{ paddingTop: contentTopPad }}>
        {selectedSite ? (
          <SiteDetail
            data={siteDetail}
            onBack={handleBack}
          />
        ) : (
          <SitesGrid
            sites={visibleSites}
            onSiteClick={handleSiteClick}
            lastUpdated={lastUpdated}
            searchQuery={searchQuery}
            onSearchQueryChange={setSearchQuery}
            dataMode={dataMode}
            demoModes={demoModes}
            activeDemoMode={demoScenarioId}
            onDemoModeChange={handleDemoScenarioChange}
          />
        )}
      </main>
    </div>
  )
}
