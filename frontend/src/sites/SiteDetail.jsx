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
              </p>
            </div>
            {data.summary?.active_alerts > 0 && (
              <span className="status-badge alert" style={{ height: 'fit-content' }}>
                <span className="badge-dot" /> Critical Alerts Active
              </span>
            )}
          </div>

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
