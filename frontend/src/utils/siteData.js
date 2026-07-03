export function normalizeSites(data) {
  return Object.entries(data).map(([siteId, value]) => ({
    site_id: siteId,
    location: value.location,
    type: value.type,
    status: value.status?.toLowerCase() ?? 'ok',
    ...value.latest?.summary,
  }))
}

export function buildAlerts(list) {
  return list
    .filter(site => site.status !== 'ok')
    .map(site => ({
      site: site.location,
      message: site.status === 'alert' ? 'Critical device issue' : 'Performance warning',
    }))
}

export function filterSitesBySearch(sites, searchQuery) {
  const normalizedSearchQuery = searchQuery.trim().toLowerCase()

  if (!normalizedSearchQuery) {
    return sites
  }

  return sites.filter(site => {
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
}

export function getSiteStats(sites) {
  return {
    total: sites.length,
    alertCount: sites.filter(site => site.status === 'alert').length,
    cautionCount: sites.filter(site => site.status === 'caution').length,
    totalDevices: sites.reduce((count, site) => count + (site.device_count ?? 0), 0),
    totalIdfs: sites.reduce((count, site) => count + (site.idf_count ?? 0), 0),
  }
}
