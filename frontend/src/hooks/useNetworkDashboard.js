import { useCallback, useDeferredValue, useEffect, useState } from 'react'
import { ACTIVE_DEMO } from '../config/demo.js'
import { POLL_INTERVAL_MS } from '../config/api.js'
import { mockScenarios, mockTestConfig } from '../data/mockData.js'
import { fetchSiteDetailFromApi, fetchSitesFromApi, fetchTestConfigFromApi } from '../services/sitesApi.js'
import { buildAlerts, filterSitesBySearch, normalizeSites } from '../utils/siteData.js'

function getActiveDemoScenario() {
  return mockScenarios[ACTIVE_DEMO] ?? mockScenarios['all-healthy']
}

function emptySiteDetail(siteId) {
  return {
    site_id: siteId,
    location: siteId,
    summary: {},
    latest: { devices: {} },
  }
}

export function useNetworkDashboard() {
  const activeDemoScenario = getActiveDemoScenario()
  const initialSites = normalizeSites(activeDemoScenario.sites)

  const [sites, setSites] = useState(initialSites)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedSite, setSelectedSite] = useState(null)
  const [selectedDevice, setSelectedDevice] = useState(null)
  const [selectedInterfaceByDevice, setSelectedInterfaceByDevice] = useState({})
  const [siteDetail, setSiteDetail] = useState(null)
  const [lastUpdated, setLastUpdated] = useState(null)
  const [alerts, setAlerts] = useState(() => buildAlerts(initialSites))
  const [dataMode, setDataMode] = useState('demo')
  const deferredSearchQuery = useDeferredValue(searchQuery)

  const applyActiveDemo = useCallback((siteId = selectedSite) => {
    const scenario = getActiveDemoScenario()
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
      const data = await fetchSitesFromApi()
      const list = normalizeSites(data)

      setSites(list)
      setAlerts(buildAlerts(list))
      setDataMode('live')
      setLastUpdated(new Date().toLocaleTimeString())
    } catch (err) {
      console.error('Failed to fetch sites:', err)
      applyActiveDemo()
    }
  }, [applyActiveDemo])

  const fetchSiteDetail = useCallback(async siteId => {
    try {
      const data = await fetchSiteDetailFromApi(siteId)

      setSiteDetail(data)
      setDataMode('live')
    } catch (err) {
      const scenario = getActiveDemoScenario()

      console.error('Failed to fetch site detail:', err)
      setSiteDetail(scenario.details[siteId] ?? emptySiteDetail(siteId))
      setDataMode('demo')
    }
  }, [])

  const fetchTestConfig = useCallback(async () => {
    try {
      await fetchTestConfigFromApi()
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
    setSelectedDevice(null)
    setSiteDetail(null)
    fetchSiteDetail(siteId)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleBack = () => {
    setSelectedSite(null)
    setSelectedDevice(null)
    setSelectedInterfaceByDevice({})
    setSiteDetail(null)
  }

  const handleDeviceClick = ip => {
    setSelectedDevice(ip)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleDeviceBack = () => {
    setSelectedDevice(null)
  }

  const handleInterfaceSelect = (deviceIp, interfaceKey) => {
    setSelectedInterfaceByDevice(prev => ({
      ...prev,
      [deviceIp]: interfaceKey,
    }))
  }

  const getSelectedInterfaceKey = deviceIp => selectedInterfaceByDevice[deviceIp] ?? null

  return {
    alerts,
    dataMode,
    getSelectedInterfaceKey,
    handleBack,
    handleDeviceBack,
    handleDeviceClick,
    handleInterfaceSelect,
    handleSiteClick,
    lastUpdated,
    searchQuery,
    selectedDevice,
    selectedSite,
    setSearchQuery,
    siteDetail,
    sites,
    visibleSites: filterSitesBySearch(sites, deferredSearchQuery),
  }
}
