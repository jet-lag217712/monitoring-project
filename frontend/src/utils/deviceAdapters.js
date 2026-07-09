/** Map API device + interfaces (+ optional metrics) into DeviceDetail UI shape. */

function formatSpeedBps(speedBps) {
  if (speedBps == null || Number.isNaN(Number(speedBps))) return null
  const bps = Number(speedBps)
  if (bps >= 1_000_000_000) return `${bps / 1_000_000_000} Gbps`
  if (bps >= 1_000_000) return `${bps / 1_000_000} Mbps`
  if (bps >= 1_000) return `${bps / 1_000} Kbps`
  return `${bps} bps`
}

export function adaptInterface(iface) {
  return {
    ...iface,
    if_index: iface.if_index,
    name: iface.name || `if${iface.if_index}`,
    admin_status: iface.admin_status || '—',
    oper_status: iface.oper_status || '—',
    speed: formatSpeedBps(iface.speed_bps) ?? '—',
    duplex: '—',
    utilization_pct: null,
    bytes_in: null,
    bytes_out: null,
    packets_in: null,
    packets_out: null,
    errors_in: null,
    errors_out: null,
    last_status_change: null,
    description: iface.description || '—',
    traffic_history: [],
  }
}

export function metricsToSeries(points) {
  if (!Array.isArray(points)) return []
  return points.map(p => ({
    ts: p.ts,
    value: p.value,
  }))
}

/**
 * @param {object} device - GET /api/devices/{id} body
 * @param {object[]} interfaces - GET .../interfaces body
 * @param {{ uptime?: object }} [metrics] - optional series keyed by chart name
 */
export function adaptDeviceDetail(device, interfaces = [], metrics = {}) {
  const adaptedInterfaces = interfaces.map(adaptInterface)
  const activeInterfaceCount = adaptedInterfaces.filter(
    iface => String(iface.oper_status).toLowerCase() === 'up',
  ).length
  const isOnline = device.status === 1
  const uptimeDays = device.uptime_days
  const sysUpTimeCs =
    uptimeDays != null && !Number.isNaN(Number(uptimeDays))
      ? Math.round(Number(uptimeDays) * 24 * 60 * 60 * 100)
      : null

  return {
    ...device,
    name: device.hostname,
    hostname: device.hostname,
    role: device.role || '',
    status: device.status,
    cpu_pct: device.cpu_pct ?? 0,
    memory_pct: device.memory_pct ?? 0,
    uptime_days: uptimeDays,
    latency_ms: device.latency_ms,
    temperature_c: null,
    interface_count: adaptedInterfaces.length,
    active_interface_count: activeInterfaceCount,
    admin_status: isOnline ? 'up' : 'down',
    oper_status: isOnline ? 'up' : 'down',
    snmp: {
      sysName: device.hostname,
      sysDescr: [device.vendor, device.model].filter(Boolean).join(' ') || '—',
      sysUpTime: sysUpTimeCs,
      sysContact: '—',
      sysLocation: '—',
    },
    history: {
      cpu: metrics.cpu ?? [],
      memory: metrics.memory ?? [],
      temperature: metrics.temperature ?? [],
      uptime: metrics.uptime ?? [],
    },
    interfaces: adaptedInterfaces,
  }
}

export function adaptApiAlerts(alerts) {
  if (!Array.isArray(alerts)) return []
  return alerts.map(a => ({
    site: a.alert_type || 'Alert',
    message: a.message || a.severity || 'Active alert',
  }))
}
