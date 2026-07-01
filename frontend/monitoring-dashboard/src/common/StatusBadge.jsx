const SITE_STATUS_LABELS = {
  ok: 'OK',
  caution: 'Caution',
  alert: 'Alert',
}

const DEVICE_STATUS_LABELS = {
  1: ['ok', 'Healthy'],
  2: ['caution', 'Warning'],
  3: ['alert', 'Critical'],
}

export function SiteStatusBadge({ status }) {
  return (
    <span className={`status-badge ${status}`}>
      <span className="badge-dot" />
      {SITE_STATUS_LABELS[status] ?? status}
    </span>
  )
}

export function DeviceStatusBadge({ status }) {
  const [className, label] = DEVICE_STATUS_LABELS[status] ?? ['ok', 'Unknown']

  return (
    <span className={`status-badge ${className}`}>
      <span className="badge-dot" />
      {label}
    </span>
  )
}
