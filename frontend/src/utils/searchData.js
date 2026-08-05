export function filterDevicesFromSiteDetails(siteDetailsById, searchQuery) {
  const needle = searchQuery.trim().toLowerCase()
  if (!needle) return []

  const hits = []
  for (const [siteId, detail] of Object.entries(siteDetailsById ?? {})) {
    const devices = detail?.latest?.devices ?? {}
    for (const [mapKey, device] of Object.entries(devices)) {
      const hay = [
        mapKey,
        device?.hostname,
        device?.ip_address,
        device?.device_id,
        device?.role,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
      if (!hay.includes(needle)) continue
      hits.push({
        kind: 'device',
        site_id: siteId,
        location: detail?.location || siteId,
        device_id: device?.device_id,
        hostname: device?.hostname,
        ip_address: device?.ip_address && device.ip_address !== '0.0.0.0' ? device.ip_address : mapKey,
        map_key: mapKey,
        role: device?.role,
        status: device?.status,
        administratively_ignored: Boolean(device?.administratively_ignored),
      })
    }
  }
  return hits
}

export function mergeDeviceSearchHits(primary, secondary) {
  const seen = new Set()
  const out = []
  for (const hit of [...(primary ?? []), ...(secondary ?? [])]) {
    const key = `${hit.site_id}:${hit.map_key || hit.hostname || hit.ip_address}`
    if (seen.has(key)) continue
    seen.add(key)
    out.push(hit)
  }
  return out
}
