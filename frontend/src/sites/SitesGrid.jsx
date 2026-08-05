import LastUpdatedLabel from '../dashboard/LastUpdatedLabel.jsx'
import OverviewStats from '../dashboard/OverviewStats.jsx'
import PageHeader from '../dashboard/PageHeader.jsx'
import PageNavStack from '../common/PageNavStack.jsx'
import SiteCard from './SiteCard.jsx'
import { getSiteStats } from '../utils/siteData.js'

export default function SitesGrid({
  sites,
  onSiteClick,
  lastUpdated,
  dataMode,
  loadError,
}) {
  const stats = getSiteStats(sites)

  return (
    <div>
      <PageNavStack breadcrumbItems={[{ label: 'All Sites' }]} />

      <PageHeader
        eyebrow={dataMode === 'live' ? 'Live Network Dashboard' : dataMode === 'demo' ? 'Demo Network Dashboard' : 'Network Dashboard'}
        title="All Sites"
        rightContent={<LastUpdatedLabel lastUpdated={lastUpdated} />}
      />

      {loadError && (
        <p className="page-sub" style={{ color: 'var(--status-alert)', marginBottom: 16 }}>
          {loadError}
        </p>
      )}

      <OverviewStats stats={stats} />

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
          <div className="empty-state-title">No sites yet</div>
          <p className="empty-state-copy">
            {loadError
              ? 'Unable to load sites from the live API.'
              : 'Sites appear here once collectors begin reporting.'}
          </p>
        </div>
      )}
    </div>
  )
}
