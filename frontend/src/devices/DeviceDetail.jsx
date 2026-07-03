import BackButton from '../common/BackButton.jsx'
import Breadcrumb from '../common/Breadcrumb.jsx'
import LoadingSkeleton from '../common/LoadingSkeleton.jsx'
import { DeviceStatusBadge } from '../common/StatusBadge.jsx'
import { resolveSelectedInterface } from '../utils/deviceData.js'
import DeviceInfoCard from './DeviceInfoCard.jsx'
import DeviceMetricsCharts from './DeviceMetricsCharts.jsx'
import InterfaceDetailPanel from './InterfaceDetailPanel.jsx'
import InterfacesTable from '../tables/InterfacesTable.jsx'

const DEVICE_STATUS_LABELS = { 1: 'Healthy', 2: 'Warning', 3: 'Critical' }

export default function DeviceDetail({
  site,
  device,
  deviceIp,
  selectedInterfaceKey,
  onInterfaceSelect,
  onNavigateAllSites,
  onNavigateSite,
}) {
  if (!device || !site) {
    return <LoadingSkeleton />
  }

  const displayName = device.name ?? device.hostname ?? deviceIp
  const selectedInterface = resolveSelectedInterface(device, selectedInterfaceKey)

  const statusLabel = DEVICE_STATUS_LABELS[device.status] ?? 'Unknown'

  return (
    <div className="device-detail-page">
      <div className="page-nav-row">
        <BackButton onClick={onNavigateSite}>← Back</BackButton>
        <Breadcrumb
          items={[
            { label: 'All Sites', onClick: onNavigateAllSites },
            { label: site.location ?? 'Site', onClick: onNavigateSite },
            { label: displayName },
          ]}
        />
      </div>

      <div className="site-detail-header">
        <div>
          <div className="page-eyebrow">
            <span className="eyebrow-dot" />
            Device Detail
          </div>
          <h1 className="page-title">{displayName}</h1>
          <p className="page-sub">
            {device.hostname ?? '—'} · {deviceIp} · {statusLabel}
          </p>
        </div>
        <DeviceStatusBadge status={device.status} />
      </div>

      <DeviceMetricsCharts history={device.history} />

      <DeviceInfoCard device={device} ip={deviceIp} />

      <div className="interface-section-header">
        <div className="chart-card-label">Interface Management</div>
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
    </div>
  )
}
