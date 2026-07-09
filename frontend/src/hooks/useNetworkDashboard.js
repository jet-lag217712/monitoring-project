import { useCallback, useDeferredValue, useEffect, useRef, useState } from 'react'
import { ACTIVE_DEMO } from '../config/demo.js'
import { DEMO_ENABLED, POLL_INTERVAL_MS } from '../config/api.js'
import { mockScenarios, mockTestConfig } from '../data/mockData.js'
import {
  ApiError,
  fetchAlertsFromApi,
  fetchDeviceFromApi,
  fetchDeviceInterfacesFromApi,
  fetchDeviceMetricsFromApi,
  fetchSiteDetailFromApi,
  fetchSitesFromApi,
  fetchTestConfigFromApi,
} from '../services/sitesApi.js'
import { adaptApiAlerts, adaptDeviceDetail, metricsToSeries } from '../utils/deviceAdapters.js'
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

function resolveCollectorDeviceId(deviceSummary, mapKey) {
  return deviceSummary?.hostname || mapKey
}

export function useNetworkDashboard({ enabled = true, onUnauthorized } = {}) {
  const activeDemoScenario = getActiveDemoScenario()
  const initialSites = DEMO_ENABLED ? normalizeSites(activeDemoScenario.sites) : []

  const [sites, setSites] = useState(initialSites)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedSite, setSelectedSite] = useState(null)
  const [selectedDevice, setSelectedDevice] = useState(null)
  const [selectedInterfaceByDevice, setSelectedInterfaceByDevice] = useState({})
  const [siteDetail, setSiteDetail] = useState(null)
  const [deviceDetail, setDeviceDetail] = useState(null)
  const [deviceLoading, setDeviceLoading] = useState(false)
  const [deviceError, setDeviceError] = useState(null)
  const [lastUpdated, setLastUpdated] = useState(null)
  const [alerts, setAlerts] = useState(() => (DEMO_ENABLED ? buildAlerts(initialSites) : []))
  const [dataMode, setDataMode] = useState(DEMO_ENABLED ? 'demo' : 'live')
  const [loadError, setLoadError] = useState(null)
  const deferredSearchQuery = useDeferredValue(searchQuery)
  const siteDetailRef = useRef(siteDetail)
  const dataModeRef = useRef(dataMode)
  siteDetailRef.current = siteDetail
  dataModeRef.current = dataMode

  const handleUnauthorized = useCallback(
    err => {
      if (err instanceof ApiError && err.status === 401 && onUnauthorized) {
        onUnauthorized()
        return true
      }
      return false
    },
    [onUnauthorized],
  )

  const applyActiveDemo = useCallback(
    (siteId = selectedSite) => {
      if (!DEMO_ENABLED) {
        setLoadError('Live API unavailable')
        setDataMode('error')
        return
      }

      const scenario = getActiveDemoScenario()
      const list = normalizeSites(scenario.sites)

      setSites(list)
      setAlerts(buildAlerts(list))
      setLastUpdated(new Date().toLocaleTimeString())
      setDataMode('demo')
      setLoadError(null)

      if (siteId) {
        setSiteDetail(scenario.details[siteId] ?? null)
      }
    },
    [selectedSite],
  )

  const fetchSites = useCallback(async () => {
    try {
      const data = await fetchSitesFromApi()
      const list = normalizeSites(data)

      setSites(list)
      setDataMode('live')
      setLastUpdated(new Date().toLocaleTimeString())
      setLoadError(null)

      try {
        const apiAlerts = await fetchAlertsFromApi()
        const adapted = adaptApiAlerts(apiAlerts)
        setAlerts(adapted.length > 0 ? adapted : buildAlerts(list))
      } catch {
        setAlerts(buildAlerts(list))
      }
    } catch (err) {
      console.error('Failed to fetch sites:', err)
      if (handleUnauthorized(err)) return
      applyActiveDemo()
    }
  }, [applyActiveDemo, handleUnauthorized])

  const fetchSiteDetail = useCallback(
    async siteId => {
      try {
        const data = await fetchSiteDetailFromApi(siteId)
        setSiteDetail(data)
        setDataMode('live')
        setLoadError(null)
      } catch (err) {
        console.error('Failed to fetch site detail:', err)
        if (handleUnauthorized(err)) return
        if (DEMO_ENABLED) {
          const scenario = getActiveDemoScenario()
          setSiteDetail(scenario.details[siteId] ?? emptySiteDetail(siteId))
          setDataMode('demo')
        } else {
          setSiteDetail(emptySiteDetail(siteId))
          setLoadError('Failed to load site detail')
          setDataMode('error')
        }
      }
    },
    [handleUnauthorized],
  )

  const fetchDeviceDetail = useCallback(
    async (siteId, deviceMapKey) => {
      if (!siteId || !deviceMapKey) return

      setDeviceLoading(true)
      setDeviceError(null)

      const summary = siteDetailRef.current?.latest?.devices?.[deviceMapKey]
      const collectorId = resolveCollectorDeviceId(summary, deviceMapKey)

      try {
        if (dataModeRef.current === 'demo' && DEMO_ENABLED) {
          const scenario = getActiveDemoScenario()
          const demoDevice = scenario.details[siteId]?.latest?.devices?.[deviceMapKey] ?? null
          setDeviceDetail(demoDevice)
          setDeviceLoading(false)
          return
        }

        const [device, interfaces, metrics] = await Promise.all([
          fetchDeviceFromApi(collectorId, siteId),
          fetchDeviceInterfacesFromApi(collectorId, siteId),
          fetchDeviceMetricsFromApi(collectorId, {
            siteId,
            metric: 'uptime_seconds',
            start: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
            end: new Date().toISOString(),
          }).catch(() => ({ points: [] })),
        ])

        setDeviceDetail(
          adaptDeviceDetail(device, interfaces, {
            uptime: metricsToSeries(metrics.points),
            cpu: [],
            memory: [],
            temperature: [],
          }),
        )
        setDataMode('live')
      } catch (err) {
        console.error('Failed to fetch device detail:', err)
        if (handleUnauthorized(err)) return
        if (DEMO_ENABLED) {
          const scenario = getActiveDemoScenario()
          setDeviceDetail(scenario.details[siteId]?.latest?.devices?.[deviceMapKey] ?? null)
          setDataMode('demo')
        } else {
          setDeviceDetail(null)
          setDeviceError(err.message ?? 'Failed to load device')
          setDataMode('error')
        }
      } finally {
        setDeviceLoading(false)
      }
    },
    [handleUnauthorized],
  )

  const fetchTestConfig = useCallback(async () => {
    try {
      await fetchTestConfigFromApi()
      setDataMode(prev => (prev === 'error' ? 'live' : prev === 'demo' ? prev : 'live'))
    } catch (err) {
      console.error('Failed to fetch test config:', err)
      if (handleUnauthorized(err)) return
      void mockTestConfig
      if (DEMO_ENABLED) {
        setDataMode('demo')
      }
    }
  }, [handleUnauthorized])

  useEffect(() => {
    if (!enabled) return undefined

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
  }, [enabled, fetchSites, fetchSiteDetail, fetchTestConfig, selectedSite])

  useEffect(() => {
    if (!enabled || !selectedSite || !selectedDevice) {
      setDeviceDetail(null)
      return undefined
    }

    fetchDeviceDetail(selectedSite, selectedDevice)

    const id = setInterval(() => {
      fetchDeviceDetail(selectedSite, selectedDevice)
    }, POLL_INTERVAL_MS)

    return () => clearInterval(id)
  }, [enabled, selectedSite, selectedDevice, fetchDeviceDetail])

  const handleSiteClick = siteId => {
    setSelectedSite(siteId)
    setSelectedDevice(null)
    setDeviceDetail(null)
    setSiteDetail(null)
    fetchSiteDetail(siteId)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleBack = () => {
    setSelectedSite(null)
    setSelectedDevice(null)
    setSelectedInterfaceByDevice({})
    setSiteDetail(null)
    setDeviceDetail(null)
  }

  const handleDeviceClick = ip => {
    setSelectedDevice(ip)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleDeviceBack = () => {
    setSelectedDevice(null)
    setDeviceDetail(null)
    setDeviceError(null)
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
    deviceDetail,
    deviceError,
    deviceLoading,
    getSelectedInterfaceKey,
    handleBack,
    handleDeviceBack,
    handleDeviceClick,
    handleInterfaceSelect,
    handleSiteClick,
    lastUpdated,
    loadError,
    searchQuery,
    selectedDevice,
    selectedSite,
    setSearchQuery,
    siteDetail,
    sites,
    visibleSites: filterSitesBySearch(sites, deferredSearchQuery),
  }
}
