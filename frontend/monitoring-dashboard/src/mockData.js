const baseSites = [
  {
    site_id: 'district-office',
    location: 'District Office',
    type: 'Core Campus',
    idf_count: 1,
    device_count: 14,
    online_count: 14,
    avg_cpu: 31,
    avg_memory: 46,
    active_alerts: 0,
  },
  {
    site_id: 'school-a',
    location: 'School A',
    type: 'Campus',
    idf_count: 2,
    device_count: 11,
    online_count: 11,
    avg_cpu: 29,
    avg_memory: 43,
    active_alerts: 0,
  },
  {
    site_id: 'school-b',
    location: 'School B',
    type: 'Campus',
    idf_count: 1,
    device_count: 8,
    online_count: 8,
    avg_cpu: 34,
    avg_memory: 48,
    active_alerts: 0,
  },
  {
    site_id: 'school-c',
    location: 'School C',
    type: 'Campus',
    idf_count: 3,
    device_count: 17,
    online_count: 17,
    avg_cpu: 37,
    avg_memory: 51,
    active_alerts: 0,
  },
]

const baseDetails = {
  'district-office': {
    site_id: 'district-office',
    location: 'District Office',
    summary: {
      total_devices: 14,
      online_count: 14,
      idf_count: 1,
      active_alerts: 0,
    },
    latest: {
      devices: {
        '10.10.0.1': { hostname: 'dist-core-01', role: 'Core Switch', status: 1, cpu_pct: 28, memory_pct: 44, uptime_days: 221, latency_ms: 1.2 },
        '10.10.0.2': { hostname: 'dist-edge-01', role: 'Firewall', status: 1, cpu_pct: 34, memory_pct: 52, uptime_days: 145, latency_ms: 1.6 },
        '10.10.0.3': { hostname: 'dist-ap-ctrl', role: 'Wireless Controller', status: 1, cpu_pct: 31, memory_pct: 49, uptime_days: 89, latency_ms: 2.3 },
      },
    },
  },
  'school-a': {
    site_id: 'school-a',
    location: 'School A',
    summary: {
      total_devices: 11,
      online_count: 11,
      idf_count: 2,
      active_alerts: 0,
    },
    latest: {
      devices: {
        '10.20.0.1': { hostname: 'sch-a-core-01', role: 'Core Switch', status: 1, cpu_pct: 26, memory_pct: 41, uptime_days: 173, latency_ms: 1.7 },
        '10.20.10.2': { hostname: 'sch-a-idf1-sw1', role: 'Access Switch', status: 1, cpu_pct: 21, memory_pct: 39, uptime_days: 112, latency_ms: 2.1 },
        '10.20.20.2': { hostname: 'sch-a-idf2-sw1', role: 'Access Switch', status: 1, cpu_pct: 33, memory_pct: 47, uptime_days: 98, latency_ms: 2.2 },
      },
    },
  },
  'school-b': {
    site_id: 'school-b',
    location: 'School B',
    summary: {
      total_devices: 8,
      online_count: 8,
      idf_count: 1,
      active_alerts: 0,
    },
    latest: {
      devices: {
        '10.30.0.1': { hostname: 'sch-b-core-01', role: 'Core Switch', status: 1, cpu_pct: 36, memory_pct: 46, uptime_days: 204, latency_ms: 1.9 },
        '10.30.10.2': { hostname: 'sch-b-idf1-sw1', role: 'Access Switch', status: 1, cpu_pct: 29, memory_pct: 42, uptime_days: 77, latency_ms: 2.6 },
      },
    },
  },
  'school-c': {
    site_id: 'school-c',
    location: 'School C',
    summary: {
      total_devices: 17,
      online_count: 17,
      idf_count: 3,
      active_alerts: 0,
    },
    latest: {
      devices: {
        '10.40.0.1': { hostname: 'sch-c-core-01', role: 'Core Switch', status: 1, cpu_pct: 39, memory_pct: 53, uptime_days: 154, latency_ms: 1.8 },
        '10.40.10.2': { hostname: 'sch-c-idf1-sw1', role: 'Access Switch', status: 1, cpu_pct: 35, memory_pct: 48, uptime_days: 133, latency_ms: 2.4 },
        '10.40.20.2': { hostname: 'sch-c-idf2-sw1', role: 'Access Switch', status: 1, cpu_pct: 41, memory_pct: 55, uptime_days: 116, latency_ms: 2.7 },
        '10.40.30.2': { hostname: 'sch-c-idf3-sw1', role: 'Distribution Switch', status: 1, cpu_pct: 32, memory_pct: 50, uptime_days: 97, latency_ms: 2.5 },
      },
    },
  },
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function deriveSiteSummary(detail) {
  const devices = Object.values(detail.latest.devices)
  const onlineCount = devices.filter(device => device.status === 1).length
  const cautionCount = devices.filter(device => device.status === 2).length
  const alertCount = devices.filter(device => device.status === 3).length
  const avgCpu = Math.round(devices.reduce((sum, device) => sum + device.cpu_pct, 0) / devices.length)
  const avgMemory = Math.round(devices.reduce((sum, device) => sum + device.memory_pct, 0) / devices.length)

  detail.summary.total_devices = devices.length
  detail.summary.online_count = onlineCount
  detail.summary.active_alerts = alertCount

  return {
    location: detail.location,
    type: detail.summary.idf_count > 1 ? 'Multi-IDF Campus' : 'Single-IDF Campus',
    status: alertCount > 0 ? 'alert' : cautionCount > 0 ? 'caution' : 'ok',
    latest: {
      summary: {
        idf_count: detail.summary.idf_count,
        device_count: devices.length,
        online_count: onlineCount,
        avg_cpu: avgCpu,
        avg_memory: avgMemory,
        active_alerts: alertCount,
      },
    },
  }
}

function buildScenario(mutator) {
  const details = clone(baseDetails)
  mutator(details)

  const sites = Object.fromEntries(
    Object.entries(details).map(([siteId, detail]) => [siteId, deriveSiteSummary(detail)]),
  )

  return { sites, details }
}

export const demoModes = [
  { id: 'all-healthy', label: 'All Healthy' },
  { id: 'two-caution', label: '2 Caution' },
  { id: 'one-red', label: '1 Red' },
]

export const mockScenarios = {
  'all-healthy': buildScenario(() => {}),
  'two-caution': buildScenario(details => {
    details['school-a'].latest.devices['10.20.10.2'].status = 2
    details['school-a'].latest.devices['10.20.10.2'].cpu_pct = 74
    details['school-a'].latest.devices['10.20.10.2'].memory_pct = 78
    details['school-a'].latest.devices['10.20.10.2'].latency_ms = 18.4

    details['school-c'].latest.devices['10.40.30.2'].status = 2
    details['school-c'].latest.devices['10.40.30.2'].cpu_pct = 79
    details['school-c'].latest.devices['10.40.30.2'].memory_pct = 81
    details['school-c'].latest.devices['10.40.30.2'].latency_ms = 21.1
  }),
  'one-red': buildScenario(details => {
    details['school-b'].latest.devices['10.30.0.1'].status = 3
    details['school-b'].latest.devices['10.30.0.1'].cpu_pct = 96
    details['school-b'].latest.devices['10.30.0.1'].memory_pct = 93
    details['school-b'].latest.devices['10.30.0.1'].latency_ms = 126.5
    details['school-b'].latest.devices['10.30.10.2'].latency_ms = 48.3
  }),
}

export const mockSitesResponse = mockScenarios['all-healthy'].sites
export const mockSiteDetails = mockScenarios['all-healthy'].details

export const mockTestConfig = {
  mode: 'demo',
  polling_enabled: true,
}
