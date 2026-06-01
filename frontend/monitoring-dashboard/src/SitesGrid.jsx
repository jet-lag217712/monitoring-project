import SiteCard from './SiteCard.jsx'

function StatCard({ label, value, tone }) {
  return (
    <div className="stat-card">
      <div className="stat-label">{label}</div>
      <div className={`stat-value ${tone ?? ''}`}>{value}</div>
    </div>
  )
}

export default function SitesGrid({
  sites,
  onSiteClick,
  lastUpdated,
  searchQuery,
  onSearchQueryChange,
  dataMode,
  demoModes,
  activeDemoMode,
  onDemoModeChange,
}) {
  const total = sites.length
  const alertCount = sites.filter(s => s.status === 'alert').length
  const cautionCount = sites.filter(s => s.status === 'caution').length
  const totalDevices = sites.reduce((n, s) => n + (s.device_count ?? 0), 0)
  const totalIdfs = sites.reduce((n, s) => n + (s.idf_count ?? 0), 0)

  return (
    <div>
      <div className="page-header">
        <div className="page-header-left">
          <div className="page-eyebrow">
            <span className="eyebrow-dot" />
            {dataMode === 'live' ? 'Network Dashboard' : 'Network Dashboard Demo'}
          </div>
          <h1 className="page-title">All Sites</h1>
          <p className="page-sub">
            District network overview — {total} sites, {totalDevices} monitored devices across {totalIdfs} IDFs
          </p>
        </div>
        {lastUpdated && (
          <span style={{
            fontFamily: "'JetBrains Mono', monospace",
            fontSize: '0.68rem',
            color: 'var(--ink-muted)',
          }}>
            Updated {lastUpdated}
          </span>
        )}
      </div>

      <div className="mode-strip">
        {demoModes.map(mode => (
          <button
            key={mode.id}
            type="button"
            className={`mode-chip ${mode.id === activeDemoMode ? 'active' : ''}`}
            onClick={() => onDemoModeChange(mode.id)}
          >
            {mode.label}
          </button>
        ))}
      </div>

      <div className="stat-strip">
        <StatCard label="Total sites" value={total} />
        <StatCard label="Network devices" value={totalDevices} />
        <StatCard label="Critical" value={alertCount} tone={alertCount > 0 ? 'alert' : ''} />
        <StatCard label="Caution"         value={cautionCount} tone={cautionCount > 0 ? 'caution' : ''} />
      </div>

      <div className="searchbar-row">
        <label className="searchbar" htmlFor="site-search">
          <span className="searchbar-icon" aria-hidden="true">⌕</span>
          <input
            id="site-search"
            type="search"
            value={searchQuery}
            onChange={event => onSearchQueryChange(event.target.value)}
            placeholder="Search by site name, type, status, or ID"
            autoComplete="off"
          />
        </label>
      </div>

      {sites.length > 0 ? (
        <div className="sites-grid">
          {sites.map(site => (
            <SiteCard
              key={site.site_id}
              site={site}
              onClick={() => onSiteClick(site.site_id)}
            />
          ))}
        </div>
      ) : (
        <div className="empty-state">
          <div className="empty-state-title">No matching sites</div>
          <p className="empty-state-copy">
            Try a different search term to find a campus, status, or site ID.
          </p>
        </div>
      )}
    </div>
  )
}
