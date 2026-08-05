import { useEffect } from 'react'
import { DeviceStatusBadge } from '../common/StatusBadge.jsx'

export default function SearchPopover({
  query,
  onClose,
  siteHits,
  deviceHits,
  loading,
  onSelectSite,
  onSelectDevice,
}) {
  const hasQuery = query.trim().length > 0
  const empty = hasQuery && !loading && siteHits.length === 0 && deviceHits.length === 0

  useEffect(() => {
    const onKeyDown = event => {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  return (
    <>
      <div className="search-popover-backdrop" onClick={onClose} aria-hidden="true" />
      <div className="search-popover" role="search">
        <div className="search-popover-toolbar">
          <button
            type="button"
            className="search-popover-close"
            onClick={onClose}
            aria-label="Close search"
          >
            ×
          </button>
        </div>

        {!hasQuery ? (
          <div className="empty-state search-popover-empty">
            <div className="empty-state-title">Start typing to search</div>
          </div>
        ) : null}

        {loading ? <p className="page-sub search-popover-status">Searching…</p> : null}

        {empty ? (
          <div className="empty-state search-popover-empty">
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
    </>
  )
}
