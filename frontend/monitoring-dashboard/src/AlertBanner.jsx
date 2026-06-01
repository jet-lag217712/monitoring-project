export default function AlertBanner({ alerts }) {
  if (!alerts || alerts.length === 0) return null

  const count = alerts.length
  const label = count === 1
    ? `${alerts[0].site} — ${alerts[0].message}`
    : `${count} sites need attention`

  return (
    <div className="alert-banner" style={{ top: '60px' }}>
      <span className="alert-dot" />
      <strong>Alert:</strong>&nbsp;{label}
    </div>
  )
}
