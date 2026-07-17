import LoadingSkeleton from '../common/LoadingSkeleton.jsx'
import PageNavStack from '../common/PageNavStack.jsx'
import { DeviceStatusBadge } from '../common/StatusBadge.jsx'
import { resolveSelectedInterface } from '../utils/deviceData.js'
import DeviceInfoCard from './DeviceInfoCard.jsx'
import DeviceMetricsCharts from './DeviceMetricsCharts.jsx'
import InterfaceDetailPanel from './InterfaceDetailPanel.jsx'
import InterfacesTable from '../tables/InterfacesTable.jsx'

const DEVICE_STATUS_LABELS = { 0: 'Unknown', 1: 'Healthy', 2: 'Warning', 3: 'Critical' }

export default function DeviceDetail({
  site,
  device,
  deviceIp,
  loading,
  error,
  selectedInterfaceKey,
  onInterfaceSelect,
  onNavigateAllSites,
  onNavigateSite,
}) {
  const siteLabel = site?.location ?? 'Site'
  const displayName = device?.name ?? device?.hostname ?? deviceIp
  const selectedInterface = device ? resolveSelectedInterface(device, selectedInterfaceKey) : null
  const statusLabel = DEVICE_STATUS_LABELS[device?.status] ?? 'Unknown'
  const reasonSuffix = device?.status_reason ? ` · ${device.status_reason}` : ''
  const rootCause =
    Array.isArray(device?.root_cause_device_ids) && device.root_cause_device_ids.length > 0
      ? ` · root cause: ${device.root_cause_device_ids.join(', ')}`
      : ''

  return (
    <div className="device-detail-page">
      <PageNavStack
        breadcrumbItems={[
          { label: 'All Sites', onClick: onNavigateAllSites },
          { label: siteLabel, onClick: onNavigateSite },
          { label: displayName },
        ]}
        onBack={onNavigateSite}
      />

      {error && !device ? (
        <p className="page-sub" style={{ color: 'var(--status-alert)' }}>
          {error}
        </p>
      ) : loading || !device || !site ? (
        <LoadingSkeleton />
      ) : (
        <>
          <div className="site-detail-header">
            <div>
              <div className="page-eyebrow">
                <span className="eyebrow-dot" />
                Device Detail
              </div>
              <h1 className="page-title">{displayName}</h1>
              <p className="page-sub">
                {device.hostname ?? '—'} · {deviceIp} · {statusLabel}
                {reasonSuffix}
                {rootCause}
              </p>
            </div>
            <DeviceStatusBadge status={device.status} />
          </div>

          <DeviceMetricsCharts history={device.history} />

          <DeviceInfoCard device={device} ip={deviceIp} />

          <div className="interface-section-header">
            <div className="chart-card-label">Interface Telemetry</div>
          </div>

          <div className="interface-split">
            <div className="interface-split-table">
              <InterfacesTable
                interfaces={device.interfaces}
                selectedKey={selectedInterfaceKey ?? selectedInterface?.name}
                onSelect={key => onInterfaceSelect(deviceIp, key)}
              />
            </div>
            <div className="interface-split-detail">
              <InterfaceDetailPanel iface={selectedInterface} />
            </div>
          </div>
        </>
      )}
    </div>
  )
}
