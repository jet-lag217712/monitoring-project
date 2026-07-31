import LoadingSkeleton from '../common/LoadingSkeleton.jsx'
import PageNavStack from '../common/PageNavStack.jsx'
import { DeviceStatusBadge } from '../common/StatusBadge.jsx'
import DevicesTable from '../tables/DevicesTable.jsx'

export default function SiteDetail({ data, siteId, onBack, onDeviceClick }) {
  const siteLabel = data?.location ?? siteId ?? 'Site'

  return (
    <div>
      <PageNavStack
        breadcrumbItems={[
          { label: 'All Sites', onClick: onBack },
          { label: siteLabel },
        ]}
        onBack={onBack}
      />

      {!data ? (
        <LoadingSkeleton />
      ) : (
        <>
          <div className="site-detail-header">
            <div>
              <div className="page-eyebrow">
                <span className="eyebrow-dot" />
                Site Detail
              </div>
              <h1 className="page-title">{data.location}</h1>
              <p className="page-sub">
                {data.summary?.total_devices ?? '—'} devices · {data.summary?.online_count ?? '—'} online
                {(data.summary?.warning_count ?? 0) > 0 && ` · ${data.summary.warning_count} warning`}
                {(data.summary?.critical_count ?? 0) > 0 && ` · ${data.summary.critical_count} critical`}
                {(data.summary?.unknown_count ?? 0) > 0 && ` · ${data.summary.unknown_count} unknown`}
                {(data.summary?.dependency_impacted_count ?? 0) > 0 &&
                  ` · ${data.summary.dependency_impacted_count} dependency-impacted`}
              </p>
            </div>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', height: 'fit-content' }}>
              {((data.summary?.critical_count ?? 0) > 0 || data.summary?.active_alerts > 0) && (
                <span className="status-badge alert">
                  <span className="badge-dot" /> Critical Alerts Active
                </span>
              )}
              {(data.summary?.unknown_count ?? 0) > 0 && (
                <span className="status-badge unknown">
                  <span className="badge-dot" /> Dependency Impact
                </span>
              )}
              {data.site_dependency_impacted && (data.root_cause_site_ids?.length ?? 0) > 0 && (
                <span className="status-badge unknown">
                  <span className="badge-dot" /> Upstream: {data.root_cause_site_ids.join(', ')}
                </span>
              )}
            </div>
          </div>

          {data.site_dependency_impacted && (data.unavailable_upstream_site_ids?.length ?? 0) > 0 && (
            <div className="site-dependency-banner" style={{ marginBottom: '1rem', color: 'var(--status-unknown)', fontSize: '0.82rem' }}>
              Site dependency impact: upstream {data.unavailable_upstream_site_ids.join(', ')} unavailable
              {(data.root_cause_site_ids?.length ?? 0) > 0 && ` · root cause ${data.root_cause_site_ids.join(', ')}`}
            </div>
          )}

          <DevicesTable
            devices={data.latest?.devices ?? {}}
            onDeviceClick={onDeviceClick}
            renderStatus={status => <DeviceStatusBadge status={status} />}
          />
        </>
      )}
    </div>
  )
}
