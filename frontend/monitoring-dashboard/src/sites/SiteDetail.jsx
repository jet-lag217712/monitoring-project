import BackButton from '../common/BackButton.jsx'
import LoadingSkeleton from '../common/LoadingSkeleton.jsx'
import { DeviceStatusBadge } from '../common/StatusBadge.jsx'
import DevicesTable from '../tables/DevicesTable.jsx'

export default function SiteDetail({ data, onBack }) {
  if (!data) {
    return (
      <div>
        <BackButton onClick={onBack} />
        <LoadingSkeleton />
      </div>
    )
  }

  const { location, summary, latest } = data
  const devices = latest?.devices ?? {}

  return (
    <div>
      <BackButton onClick={onBack} />

      <div className="site-detail-header">
        <div>
          <div className="page-eyebrow">
            <span className="eyebrow-dot" />
            Site Detail
          </div>
          <h1 className="page-title">{location}</h1>
          <p className="page-sub">
            {summary?.total_devices ?? '—'} devices · {summary?.online_count ?? '—'} online
          </p>
        </div>
        {summary?.active_alerts > 0 && (
          <span className="status-badge alert" style={{ height: 'fit-content' }}>
            <span className="badge-dot" /> Critical Alerts Active
          </span>
        )}
      </div>

      <DevicesTable
        devices={devices}
        renderStatus={status => <DeviceStatusBadge status={status} />}
      />
    </div>
  )
}
