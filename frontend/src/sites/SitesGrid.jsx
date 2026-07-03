import LastUpdatedLabel from '../dashboard/LastUpdatedLabel.jsx'
import OverviewStats from '../dashboard/OverviewStats.jsx'
import PageHeader from '../dashboard/PageHeader.jsx'
import PageNavStack from '../common/PageNavStack.jsx'
import SearchBar from '../common/SearchBar.jsx'
import SiteCard from './SiteCard.jsx'
import { getSiteStats } from '../utils/siteData.js'

export default function SitesGrid({
  sites,
  onSiteClick,
  lastUpdated,
  searchQuery,
  onSearchQueryChange,
  dataMode,
}) {
  const stats = getSiteStats(sites)

  return (
    <div>
      <PageNavStack breadcrumbItems={[{ label: 'All Sites' }]} />

      <PageHeader
        eyebrow={dataMode === 'live' ? 'Network Dashboard' : 'Network Dashboard'}
        title="All Sites"
        rightContent={<LastUpdatedLabel lastUpdated={lastUpdated} />}
      />

      <OverviewStats stats={stats} />

      <SearchBar
        value={searchQuery}
        onChange={onSearchQueryChange}
      />

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
