/** Map API device + interfaces (+ optional metrics) into DeviceDetail UI shape. */

function formatSpeedBps(speedBps) {
  if (speedBps == null || Number.isNaN(Number(speedBps))) return null
  const bps = Number(speedBps)
  if (bps >= 1_000_000_000) return `${bps / 1_000_000_000} Gbps`
  if (bps >= 1_000_000) return `${bps / 1_000_000} Mbps`
  if (bps >= 1_000) return `${bps / 1_000} Kbps`
  return `${bps} bps`
}

function mapHistorySeries(points) {
  if (!Array.isArray(points)) return []
  return points.map(p => ({
    ts: p.ts,
    value: p.value,
  }))
}

export function adaptInterface(iface) {
  const traffic = Array.isArray(iface.traffic_history)
    ? mapHistorySeries(iface.traffic_history)
    : []

  return {
    ...iface,
    if_index: iface.if_index,
    name: iface.name || `if${iface.if_index}`,
    admin_status: iface.admin_status || '—',
    oper_status: iface.oper_status || '—',
    speed: formatSpeedBps(iface.speed_bps) ?? '—',
    duplex: iface.duplex ?? '—',
    utilization_pct: iface.utilization_pct ?? null,
    bytes_in: iface.in_octets ?? iface.bytes_in ?? null,
    bytes_out: iface.out_octets ?? iface.bytes_out ?? null,
    packets_in: iface.packets_in ?? null,
    packets_out: iface.packets_out ?? null,
    errors_in: iface.in_errors ?? iface.errors_in ?? null,
    errors_out: iface.out_errors ?? iface.errors_out ?? null,
    last_status_change: iface.last_status_change ?? null,
    description: iface.description || iface.if_alias || '—',
    if_alias: iface.if_alias ?? null,
    if_type: iface.if_type ?? null,
    traffic_history: traffic,
  }
}

export function metricsToSeries(points) {
  return mapHistorySeries(points)
}

function pickHistory(deviceHistory, metrics, key, metricKey) {
  if (Array.isArray(deviceHistory?.[key]) && deviceHistory[key].length > 0) {
    return mapHistorySeries(deviceHistory[key])
  }
  if (Array.isArray(metrics?.[metricKey])) {
    return metrics[metricKey]
  }
  return []
}

/**
 * @param {object} device - GET /api/devices/{id} body
 * @param {object[]} interfaces - GET .../interfaces body
 * @param {{ uptime?: object[], cpu?: object[], memory?: object[], temperature?: object[] }} [metrics]
 */
export function adaptDeviceDetail(device, interfaces = [], metrics = {}) {
  const adaptedInterfaces = interfaces.map(adaptInterface)
  const activeInterfaceCount = adaptedInterfaces.filter(
    iface => String(iface.oper_status).toLowerCase() === 'up',
  ).length

  const status = device.status
  const isReachable = status === 1 || status === 2
  const deviceHistory = device.history ?? {}

  const snmp = device.snmp
    ? {
        sysName: device.snmp.sysName ?? device.hostname ?? '—',
        sysDescr: device.snmp.sysDescr ?? '—',
        sysObjectID: device.snmp.sysObjectID ?? null,
        sysUpTime: device.snmp.sysUpTime ?? null,
        sysContact: device.snmp.sysContact ?? '—',
        sysLocation: device.snmp.sysLocation ?? '—',
      }
    : {
        sysName: device.hostname ?? '—',
        sysDescr: [device.vendor, device.model].filter(Boolean).join(' ') || '—',
        sysObjectID: null,
        sysUpTime: null,
        sysContact: '—',
        sysLocation: '—',
      }

  return {
    ...device,
    name: device.hostname,
    hostname: device.hostname,
    role: device.role || '',
    status,
    status_reason: device.status_reason ?? null,
    failure_count: device.failure_count ?? null,
    upstream_device_ids: device.upstream_device_ids ?? [],
    unavailable_upstream_device_ids: device.unavailable_upstream_device_ids ?? [],
    root_cause_device_ids: device.root_cause_device_ids ?? [],
    serial_number: device.serial_number ?? null,
    profile: device.profile ?? null,
    capabilities: device.capabilities ?? [],
    cpu_pct: device.cpu_pct ?? null,
    memory_pct: device.memory_pct ?? null,
    temperature_c: device.temperature_c ?? device.primary_temperature_c ?? null,
    uptime_days: device.uptime_days ?? null,
    latency_ms: device.latency_ms ?? null,
    temperature_components: device.temperature_components ?? [],
    power_components: device.power_components ?? [],
    interface_count: adaptedInterfaces.length,
    active_interface_count: activeInterfaceCount,
    admin_status: isReachable ? 'up' : 'down',
    oper_status: isReachable ? 'up' : 'down',
    snmp,
    history: {
      cpu: pickHistory(deviceHistory, metrics, 'cpu', 'cpu'),
      memory: pickHistory(deviceHistory, metrics, 'memory', 'memory'),
      temperature: pickHistory(deviceHistory, metrics, 'temperature', 'temperature'),
      uptime: pickHistory(deviceHistory, metrics, 'uptime', 'uptime'),
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
