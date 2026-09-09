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
  fetchSearchFromApi,
  fetchSiteDetailFromApi,
  fetchSitesFromApi,
  fetchTestConfigFromApi,
  updateSiteLocation,
} from '../services/sitesApi.js'
import { adaptApiAlerts, adaptDeviceDetail, metricsToSeries } from '../utils/deviceAdapters.js'
import { buildAlerts, filterSitesBySearch, normalizeSites } from '../utils/siteData.js'
import { filterDevicesFromSiteDetails, mergeDeviceSearchHits } from '../utils/searchData.js'

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
  return deviceSummary?.device_id || deviceSummary?.hostname || mapKey
}

export function useNetworkDashboard({ enabled = true, onUnauthorized } = {}) {
  const activeDemoScenario = getActiveDemoScenario()
  const initialSites = DEMO_ENABLED ? normalizeSites(activeDemoScenario.sites) : []

  const [sites, setSites] = useState(initialSites)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchDeviceHits, setSearchDeviceHits] = useState([])
  const [searchLoading, setSearchLoading] = useState(false)
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
  const siteFetchSeqRef = useRef(0)
  const deviceFetchSeqRef = useRef(0)
  const searchSeqRef = useRef(0)
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
      const seq = siteFetchSeqRef.current
      const isCurrentFetch = () => seq === siteFetchSeqRef.current

      try {
        const data = await fetchSiteDetailFromApi(siteId)
        if (!isCurrentFetch()) return
        setSiteDetail(data)
        setDataMode('live')
        setLoadError(null)
      } catch (err) {
        console.error('Failed to fetch site detail:', err)
        if (handleUnauthorized(err)) return
        if (!isCurrentFetch()) return
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

      const seq = deviceFetchSeqRef.current
      setDeviceLoading(true)
      setDeviceError(null)

      const summary = siteDetailRef.current?.latest?.devices?.[deviceMapKey]
      const collectorId = resolveCollectorDeviceId(summary, deviceMapKey)

      const isCurrentFetch = () => seq === deviceFetchSeqRef.current

      try {
        if (dataModeRef.current === 'demo' && DEMO_ENABLED) {
          const scenario = getActiveDemoScenario()
          const demoDevice = scenario.details[siteId]?.latest?.devices?.[deviceMapKey] ?? null
          if (!isCurrentFetch()) return
          setDeviceDetail(demoDevice)
          return
        }

        const historyStart = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString()
        const historyEnd = new Date().toISOString()
        const metricOpts = { siteId, start: historyStart, end: historyEnd }

        const [device, interfaces, uptimeMetrics, cpuMetrics, memMetrics, tempMetrics] =
          await Promise.all([
            fetchDeviceFromApi(collectorId, siteId),
            fetchDeviceInterfacesFromApi(collectorId, siteId),
            fetchDeviceMetricsFromApi(collectorId, {
              ...metricOpts,
              metric: 'uptime_seconds',
            }).catch(() => ({ points: [] })),
            fetchDeviceMetricsFromApi(collectorId, {
              ...metricOpts,
              metric: 'cpu_utilization_pct',
            }).catch(() => ({ points: [] })),
            fetchDeviceMetricsFromApi(collectorId, {
              ...metricOpts,
              metric: 'memory_utilization_pct',
            }).catch(() => ({ points: [] })),
            fetchDeviceMetricsFromApi(collectorId, {
              ...metricOpts,
              metric: 'primary_temperature_c',
            }).catch(() => ({ points: [] })),
          ])

        if (!isCurrentFetch()) return

        setDeviceDetail(
          adaptDeviceDetail(device, interfaces, {
            uptime: metricsToSeries(uptimeMetrics.points),
            cpu: metricsToSeries(cpuMetrics.points),
            memory: metricsToSeries(memMetrics.points),
            temperature: metricsToSeries(tempMetrics.points),
          }),
        )
        setDataMode('live')
      } catch (err) {
        console.error('Failed to fetch device detail:', err)
        if (handleUnauthorized(err)) return
        if (!isCurrentFetch()) return
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
        if (isCurrentFetch()) {
          setDeviceLoading(false)
        }
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

    const id = setInterval(() => {
      fetchTestConfig()
      fetchSites()
    }, POLL_INTERVAL_MS)

    return () => clearInterval(id)
  }, [enabled, fetchSites, fetchTestConfig])

  useEffect(() => {
    if (!enabled || !selectedSite) {
      siteFetchSeqRef.current += 1
      setSiteDetail(null)
      return undefined
    }

    siteFetchSeqRef.current += 1
    setSiteDetail(null)
    fetchSiteDetail(selectedSite)

    const id = setInterval(() => {
      fetchSiteDetail(selectedSite)
    }, POLL_INTERVAL_MS)

    return () => clearInterval(id)
  }, [enabled, selectedSite, fetchSiteDetail])

  useEffect(() => {
    if (!enabled || !selectedSite || !selectedDevice) {
      deviceFetchSeqRef.current += 1
      setDeviceDetail(null)
      setDeviceError(null)
      setDeviceLoading(false)
      return undefined
    }

    deviceFetchSeqRef.current += 1
    setDeviceDetail(null)
    setDeviceError(null)

    fetchDeviceDetail(selectedSite, selectedDevice)

    const id = setInterval(() => {
      fetchDeviceDetail(selectedSite, selectedDevice)
    }, POLL_INTERVAL_MS)

    return () => clearInterval(id)
  }, [enabled, selectedSite, selectedDevice, fetchDeviceDetail])

  useEffect(() => {
    if (!enabled || !searchOpen) {
      searchSeqRef.current += 1
      setSearchDeviceHits([])
      setSearchLoading(false)
      return undefined
    }

    const needle = deferredSearchQuery.trim()
    if (!needle) {
      searchSeqRef.current += 1
      setSearchDeviceHits([])
      setSearchLoading(false)
      return undefined
    }

    const seq = ++searchSeqRef.current
    setSearchLoading(true)

    const run = async () => {
      const localDetails = {}
      if (siteDetailRef.current?.site_id) {
        localDetails[siteDetailRef.current.site_id] = siteDetailRef.current
      }
      if (DEMO_ENABLED || dataModeRef.current === 'demo') {
        const scenario = getActiveDemoScenario()
        Object.assign(localDetails, scenario.details ?? {})
      }
      const localHits = filterDevicesFromSiteDetails(localDetails, needle)

      try {
        if (dataModeRef.current === 'demo' && DEMO_ENABLED) {
          if (seq !== searchSeqRef.current) return
          setSearchDeviceHits(localHits)
          return
        }
        const apiResult = await fetchSearchFromApi(needle)
        if (seq !== searchSeqRef.current) return
        setSearchDeviceHits(mergeDeviceSearchHits(apiResult.devices, localHits))
      } catch (err) {
        console.error('Search failed:', err)
        if (handleUnauthorized(err)) return
        if (seq !== searchSeqRef.current) return
        setSearchDeviceHits(localHits)
      } finally {
        if (seq === searchSeqRef.current) {
          setSearchLoading(false)
        }
      }
    }

    void run()
    return undefined
  }, [enabled, searchOpen, deferredSearchQuery, handleUnauthorized])

  const openSearch = () => {
    setSearchOpen(true)
  }

  const closeSearch = () => {
    setSearchOpen(false)
    setSearchQuery('')
    setSearchDeviceHits([])
    setSearchLoading(false)
  }

  const handleSiteClick = siteId => {
    setSearchOpen(false)
    setSearchQuery('')
    setSelectedSite(siteId)
    setSelectedDevice(null)
    setDeviceDetail(null)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleBack = () => {
    setSearchOpen(false)
    setSearchQuery('')
    setSelectedSite(null)
    setSelectedDevice(null)
    setSelectedInterfaceByDevice({})
    setSiteDetail(null)
    setDeviceDetail(null)
  }

  const handleDeviceClick = ip => {
    setSearchOpen(false)
    setSearchQuery('')
    setDeviceDetail(null)
    setDeviceError(null)
    setSelectedDevice(ip)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleSearchDeviceSelect = hit => {
    if (!hit?.site_id) return
    setSearchOpen(false)
    setSearchQuery('')
    setSelectedSite(hit.site_id)
    setSelectedDevice(hit.map_key || hit.hostname || hit.ip_address)
    setDeviceDetail(null)
    setDeviceError(null)
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

  const handleRenameLocation = useCallback(
    async (siteId, location) => {
      const nextLabel = String(location ?? '').trim()

      if (dataModeRef.current === 'demo' && DEMO_ENABLED) {
        const display = nextLabel || siteId
        setSites(prev => {
          const next = prev.map(site => (site.site_id === siteId ? { ...site, location: display } : site))
          setAlerts(buildAlerts(next))
          return next
        })
        setSiteDetail(prev => (prev && prev.site_id === siteId ? { ...prev, location: display } : prev))
        return { site_id: siteId, location: display }
      }

      try {
        const updated = await updateSiteLocation(siteId, nextLabel)
        setSiteDetail(prev =>
          prev && prev.site_id === siteId ? { ...prev, location: updated.location } : prev,
        )
        await fetchSites()
        return updated
      } catch (err) {
        handleUnauthorized(err)
        throw err
      }
    },
    [fetchSites, handleUnauthorized],
  )

  const siteHits = filterSitesBySearch(sites, deferredSearchQuery).map(site => ({
    kind: 'site',
    site_id: site.site_id,
    location: site.location,
  }))

  return {
    alerts,
    closeSearch,
    dataMode,
    deviceDetail,
    deviceError,
    deviceLoading,
    getSelectedInterfaceKey,
    handleBack,
    handleDeviceBack,
    handleDeviceClick,
    handleInterfaceSelect,
    handleRenameLocation,
    handleSearchDeviceSelect,
    handleSiteClick,
    lastUpdated,
    loadError,
    openSearch,
    searchDeviceHits,
    searchLoading,
    searchOpen,
    searchQuery,
    searchSiteHits: siteHits,
    selectedDevice,
    selectedSite,
    setSearchQuery,
    siteDetail,
    sites,
    visibleSites: sites,
  }
}
