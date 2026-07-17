import { DEVICE_STATUS_LABELS, SITE_STATUS_LABELS } from './statusLabels.js'

export function SiteStatusBadge({ status }) {
  return (
    <span className={`status-badge ${status}`}>
      <span className="badge-dot" />
      {SITE_STATUS_LABELS[status] ?? status}
    </span>
  )
}

export function DeviceStatusBadge({ status }) {
  const [className, label] = DEVICE_STATUS_LABELS[status] ?? ['unknown', 'Unknown']

  return (
    <span className={`status-badge ${className}`}>
      <span className="badge-dot" />
      {label}
    </span>
  )
}

export { DEVICE_STATUS_LABELS, SITE_STATUS_LABELS }
