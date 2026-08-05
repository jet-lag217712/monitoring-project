import { DeviceStatusBadge } from '../common/StatusBadge.jsx'

export default function SearchTakeover({
  query,
  onQueryChange,
  onClose,
  siteHits,
  deviceHits,
  loading,
  onSelectSite,
  onSelectDevice,
}) {
  const hasQuery = query.trim().length > 0
  const empty = hasQuery && !loading && siteHits.length === 0 && deviceHits.length === 0

  return (
    <div className="search-takeover" role="search">
      <div className="search-takeover-header">
        <div>
          <p className="page-eyebrow">Search</p>
          <h1 className="page-title">Find sites and devices</h1>
          <p className="page-sub">Match site name, hostname, or IP address.</p>
        </div>
        <button type="button" className="search-takeover-close" onClick={onClose}>
          Close
        </button>
      </div>

      <label className="searchbar searchbar-takeover" htmlFor="takeover-search">
        <span className="searchbar-icon" aria-hidden="true">⌕</span>
        <input
          id="takeover-search"
          type="search"
          value={query}
          onChange={event => onQueryChange(event.target.value)}
          onKeyDown={event => {
            if (event.key === 'Escape') onClose()
          }}
          placeholder="Search by site, hostname, or IP…"
          autoComplete="off"
          autoFocus
        />
      </label>

      {!hasQuery ? (
        <div className="empty-state">
          <div className="empty-state-title">Start typing to search</div>
          <p className="empty-state-copy">
            Results cover sites plus device hostname and IP address across the appliance.
          </p>
        </div>
      ) : null}

      {loading ? <p className="page-sub">Searching…</p> : null}

      {empty ? (
        <div className="empty-state">
          <div className="empty-state-title">No matches</div>
          <p className="empty-state-copy">Try a different site name, hostname, or IP address.</p>
        </div>
      ) : null}

      {siteHits.length > 0 ? (
        <section className="search-results-section">
          <h2 className="search-results-heading">Sites</h2>
          <ul className="search-results-list">
            {siteHits.map(site => (
              <li key={site.site_id}>
                <button
                  type="button"
                  className="search-result-row"
                  onClick={() => onSelectSite(site.site_id)}
                >
                  <span className="search-result-title">{site.location || site.site_id}</span>
                  <span className="search-result-meta">{site.site_id}</span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {deviceHits.length > 0 ? (
        <section className="search-results-section">
          <h2 className="search-results-heading">Devices</h2>
          <ul className="search-results-list">
            {deviceHits.map(device => (
              <li key={`${device.site_id}:${device.map_key || device.hostname}`}>
                <button
                  type="button"
                  className="search-result-row"
                  onClick={() => onSelectDevice(device)}
                >
                  <span className="search-result-title">
                    {device.hostname || device.map_key}
                    {device.administratively_ignored ? (
                      <span className="ignored-pill">Ignored</span>
                    ) : null}
                  </span>
                  <span className="search-result-meta">
                    {[device.ip_address, device.site_id, device.role]
                      .filter(Boolean)
                      .join(' · ')}
                  </span>
                  {typeof device.status === 'number' ? (
                    <DeviceStatusBadge status={device.status} />
                  ) : null}
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  )
}
