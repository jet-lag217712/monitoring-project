let lastSiteTopoSig = ''

export function normalizeSites(data) {
  const list = Object.entries(data).map(([siteId, value]) => ({
    site_id: siteId,
    location: value.location,
    type: value.type,
    status: value.status?.toLowerCase() ?? 'ok',
    upstream_site_ids: value.upstream_site_ids ?? [],
    unavailable_upstream_site_ids: value.unavailable_upstream_site_ids ?? [],
    root_cause_site_ids: value.root_cause_site_ids ?? [],
    site_dependency_impacted: value.site_dependency_impacted ?? false,
    ...value.latest?.summary,
  }))
  // #region agent log
  const snapshot = list.map(site => ({
    site_id: site.site_id,
    location: site.location,
    status: site.status,
    device_count: site.device_count ?? 0,
    critical_count: site.critical_count ?? 0,
    unknown_count: site.unknown_count ?? 0,
    dependency_impacted_count: site.dependency_impacted_count ?? 0,
    upstream_site_ids: site.upstream_site_ids ?? [],
    unavailable_upstream_site_ids: site.unavailable_upstream_site_ids ?? [],
    root_cause_site_ids: site.root_cause_site_ids ?? [],
    site_dependency_impacted: !!site.site_dependency_impacted,
  }))
  const sig = JSON.stringify(snapshot)
  if (sig !== lastSiteTopoSig) {
    lastSiteTopoSig = sig
    fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'f7c9cd'},body:JSON.stringify({sessionId:'f7c9cd',runId:'post-fix',hypothesisId:'A',location:'siteData.js:normalizeSites',message:'site overview topology snapshot',data:{sites:snapshot},timestamp:Date.now()})}).catch(()=>{});
  }
  // #endregion
  return list
}

export function buildAlerts(list) {
  return list
    .filter(site => site.status === 'alert' || (site.critical_count ?? 0) > 0)
    .map(site => ({
      site: site.location,
      message:
        (site.critical_count ?? 0) > 0
          ? `${site.critical_count} critical device${site.critical_count === 1 ? '' : 's'}`
          : site.status === 'alert'
            ? 'Critical device issue'
            : 'Performance warning',
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
    alertCount: sites.filter(site => site.status === 'alert' || (site.critical_count ?? 0) > 0).length,
    cautionCount: sites.filter(site => site.status === 'caution').length,
    unknownCount: sites.reduce((count, site) => count + (site.unknown_count ?? 0), 0),
    dependencyImpactedCount: sites.reduce(
      (count, site) => count + (site.dependency_impacted_count ?? 0),
      0,
    ),
    totalDevices: sites.reduce((count, site) => count + (site.device_count ?? 0), 0),
    totalIdfs: sites.reduce((count, site) => count + (site.idf_count ?? 0), 0),
  }
}
