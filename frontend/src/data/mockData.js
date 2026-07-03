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

const DEVICE_META = {
  'Core Switch': { vendor: 'Cisco', model: 'Catalyst 9300-48P', portCount: 12, prefix: 'Gi1/0' },
  'Firewall': { vendor: 'Palo Alto', model: 'PA-3220', portCount: 8, prefix: 'ethernet1/' },
  'Wireless Controller': { vendor: 'Aruba', model: 'Mobility Master', portCount: 4, prefix: 'Port' },
  'Access Switch': { vendor: 'Cisco', model: 'Catalyst 9200-24P', portCount: 12, prefix: 'Gi1/0' },
  'Distribution Switch': { vendor: 'Cisco', model: 'Catalyst 9500-24Y4C', portCount: 12, prefix: 'Te1/0' },
}

function hashSeed(str) {
  let h = 0
  for (let i = 0; i < str.length; i++) h = (h * 31 + str.charCodeAt(i)) | 0
  return Math.abs(h)
}

function createRng(seed) {
  let s = seed
  return () => {
    s = (s * 1103515245 + 12345) & 0x7fffffff
    return s / 0x7fffffff
  }
}

function buildHistory(seed, baseValue, variance = 12, points = 24) {
  const rng = createRng(seed)
  const now = Date.now()
  const hourMs = 60 * 60 * 1000
  return Array.from({ length: points }, (_, i) => {
    const offset = (points - 1 - i) * hourMs
    const jitter = (rng() - 0.5) * variance * 2
    const value = Math.max(0, Math.min(100, Math.round(baseValue + jitter)))
    return { ts: new Date(now - offset).toISOString(), value }
  })
}

function buildTrafficHistory(seed, baseIn, baseOut, points = 24) {
  const rng = createRng(seed + 99)
  const now = Date.now()
  const hourMs = 60 * 60 * 1000
  return Array.from({ length: points }, (_, i) => {
    const offset = (points - 1 - i) * hourMs
    const inJitter = (rng() - 0.5) * baseIn * 0.4
    const outJitter = (rng() - 0.5) * baseOut * 0.4
    return {
      ts: new Date(now - offset).toISOString(),
      in_mbps: Math.max(0, Math.round((baseIn + inJitter) * 10) / 10),
      out_mbps: Math.max(0, Math.round((baseOut + outJitter) * 10) / 10),
    }
  })
}

function buildInterfaces(ip, device) {
  const meta = DEVICE_META[device.role] ?? DEVICE_META['Access Switch']
  const rng = createRng(hashSeed(`${ip}-${device.hostname}`))
  const descriptions = [
    'Uplink to core',
    'IDF access layer',
    'AP trunk VLAN 110',
    'Security camera VLAN 120',
    'VoIP phone VLAN 130',
    'Guest Wi-Fi uplink',
    'Building management system',
    'Spare — unused',
  ]

  return Array.from({ length: meta.portCount }, (_, i) => {
    const ifIndex = i + 1
    const name = `${meta.prefix}${ifIndex}`
    const isUp = rng() > 0.12
    const utilization = isUp ? Math.round(rng() * 85) : 0
    const speed = device.role === 'Distribution Switch' && i < 2 ? '10 Gbps' : '1 Gbps'
    const bytesBase = isUp ? Math.round(rng() * 8e9 + 5e7) : 0
    const seed = hashSeed(`${ip}-${name}`)

    return {
      if_index: ifIndex,
      name,
      status: isUp ? 'up' : 'down',
      utilization_pct: utilization,
      speed,
      admin_status: isUp || rng() > 0.3 ? 'up' : 'down',
      oper_status: isUp ? 'up' : 'down',
      duplex: isUp ? 'full' : '—',
      bytes_in: bytesBase,
      bytes_out: Math.round(bytesBase * (0.3 + rng() * 0.5)),
      packets_in: Math.round(bytesBase / 900),
      packets_out: Math.round(bytesBase / 1100),
      errors_in: isUp && rng() > 0.85 ? Math.round(rng() * 12) : 0,
      errors_out: isUp && rng() > 0.9 ? Math.round(rng() * 8) : 0,
      last_status_change: new Date(Date.now() - Math.round(rng() * 30) * 86400000).toISOString(),
      description: descriptions[i % descriptions.length],
      traffic_history: buildTrafficHistory(seed, 20 + utilization * 0.8, 10 + utilization * 0.5),
    }
  })
}

function buildSysDescr(role, meta) {
  switch (role) {
    case 'Core Switch':
      return `Cisco IOS XE Software, ${meta.model}`
    case 'Firewall':
      return `Palo Alto PAN-OS, ${meta.model}`
    case 'Wireless Controller':
      return `Aruba AOS, ${meta.model}`
    case 'Distribution Switch':
      return `Cisco IOS XE Software, ${meta.model}`
    default:
      return `Cisco IOS Software, ${meta.model}`
  }
}

function enrichDevice(ip, device, siteLocation) {
  const meta = DEVICE_META[device.role] ?? DEVICE_META['Access Switch']
  const seed = hashSeed(ip)
  const rng = createRng(seed)
  const serialSuffix = (seed % 900000 + 100000).toString(16).toUpperCase()
  const interfaces = buildInterfaces(ip, device)
  const activeInterfaceCount = interfaces.filter(iface => iface.oper_status === 'up').length
  const isOnline = device.status === 1
  const psu1_v = Math.round((11.8 + rng() * 0.6) * 10) / 10
  const psu2_v = Math.round((11.8 + rng() * 0.6) * 10) / 10
  const voltagesInRange = psu1_v >= 11.5 && psu1_v <= 12.5 && psu2_v >= 11.5 && psu2_v <= 12.5

  return {
    ...device,
    name: device.hostname,
    vendor: meta.vendor,
    model: meta.model,
    serial_number: `SN-${serialSuffix}`,
    temperature_c: Math.round(38 + rng() * 18 + (device.cpu_pct ?? 0) * 0.1),
    power_supply: {
      psu1_v,
      psu2_v,
    },
    power_supply_status: voltagesInRange ? 'Normal' : 'Warning',
    snmp: {
      sysName: device.hostname,
      sysDescr: buildSysDescr(device.role, meta),
      sysUpTime: Math.round((device.uptime_days ?? 0) * 24 * 60 * 60 * 100),
      sysContact: 'netops@district.edu',
      sysLocation: siteLocation ?? '—',
    },
    interface_count: interfaces.length,
    active_interface_count: activeInterfaceCount,
    admin_status: isOnline ? 'up' : 'down',
    oper_status: isOnline ? 'up' : 'down',
    history: {
      cpu: buildHistory(seed, device.cpu_pct ?? 30),
      memory: buildHistory(seed + 1, device.memory_pct ?? 40),
      temperature: buildHistory(seed + 2, 42 + (device.cpu_pct ?? 0) * 0.15, 4),
    },
    interfaces,
  }
}

function enrichDetails(details) {
  for (const detail of Object.values(details)) {
    const devices = detail.latest?.devices ?? {}
    for (const [ip, device] of Object.entries(devices)) {
      detail.latest.devices[ip] = enrichDevice(ip, device, detail.location)
    }
  }
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
  enrichDetails(details)

  const sites = Object.fromEntries(
    Object.entries(details).map(([siteId, detail]) => [siteId, deriveSiteSummary(detail)]),
  )

  return { sites, details }
}

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

export const mockTestConfig = {
  mode: 'demo',
  polling_enabled: true,
}
