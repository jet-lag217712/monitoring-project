/** Map API device + interfaces (+ optional metrics) into DeviceDetail UI shape. */

function normalizeInterfaceStatus(value) {
  const normalized = String(value ?? '').trim().toLowerCase()
  return normalized || null
}

function deriveInterfaceStatus(iface) {
  return (
    normalizeInterfaceStatus(iface.status) ??
    normalizeInterfaceStatus(iface.oper_status) ??
    'down'
  )
}

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

function octetDeltaToMbps(deltaOctets, dtSec) {
  if (dtSec <= 0 || deltaOctets < 0) return null
  return Math.round(((deltaOctets * 8) / (dtSec * 1_000_000)) * 100) / 100
}

function mapTrafficHistory(points) {
  if (!Array.isArray(points) || points.length === 0) return []

  if (points[0].in_mbps != null || points[0].out_mbps != null) {
    return points.map(p => ({
      ts: p.ts,
      in_mbps: Number(p.in_mbps ?? 0),
      out_mbps: Number(p.out_mbps ?? 0),
    }))
  }

  const sorted = [...points].sort((a, b) => new Date(a.ts) - new Date(b.ts))
  const result = []

  for (let i = 1; i < sorted.length; i++) {
    const prev = sorted[i - 1]
    const curr = sorted[i]
    const dtSec = (new Date(curr.ts) - new Date(prev.ts)) / 1000
    if (dtSec <= 0) continue

    const currIn = curr.in_octets ?? curr.value
    const currOut = curr.out_octets
    const prevIn = prev.in_octets ?? prev.value
    const prevOut = prev.out_octets

    if (currIn == null && currOut == null) continue

    const inDelta = currIn != null && prevIn != null ? currIn - prevIn : null
    const outDelta = currOut != null && prevOut != null ? currOut - prevOut : null

    result.push({
      ts: curr.ts,
      in_mbps: inDelta != null ? octetDeltaToMbps(inDelta, dtSec) : null,
      out_mbps: outDelta != null ? octetDeltaToMbps(outDelta, dtSec) : null,
    })
  }

  return result
}

function deriveUtilizationPct(trafficHistory, speedBps) {
  if (speedBps == null || !trafficHistory.length) return null
  const latest = trafficHistory[trafficHistory.length - 1]
  const inMbps = latest.in_mbps ?? 0
  const outMbps = latest.out_mbps ?? 0
  const speedMbps = speedBps / 1_000_000
  if (speedMbps <= 0) return null
  return Math.min(100, Math.round(((inMbps + outMbps) / speedMbps) * 10000) / 100)
}

export function adaptInterface(iface) {
  const traffic = mapTrafficHistory(iface.traffic_history)
  const utilizationPct =
    iface.utilization_pct ?? deriveUtilizationPct(traffic, iface.speed_bps)

  // #region agent log
  if (iface?.name === 'Gi0/0' || iface?.if_index === 1) {
    const raw = Array.isArray(iface.traffic_history) ? iface.traffic_history : []
    fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'269a95'},body:JSON.stringify({sessionId:'269a95',location:'deviceAdapters.js:adaptInterface',message:'traffic history mapping',data:{name:iface.name,rawCount:raw.length,rawFirst:raw[0]??null,mappedCount:traffic.length,mappedFirst:traffic[0]??null,utilizationPct},timestamp:Date.now(),runId:'post-fix',hypothesisId:'A'})}).catch(()=>{});
  }
  // #endregion

  return {
    ...iface,
    if_index: iface.if_index,
    name: iface.name || `if${iface.if_index}`,
    status: deriveInterfaceStatus(iface),
    admin_status: iface.admin_status || '—',
    oper_status: iface.oper_status || '—',
    speed: formatSpeedBps(iface.speed_bps) ?? '—',
    duplex: iface.duplex ?? '—',
    utilization_pct: utilizationPct,
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
