import StatCard from './StatCard.jsx'

export default function OverviewStats({ stats }) {
  return (
    <div className="stat-strip">
      <StatCard label="Total sites" value={stats.total} />
      <StatCard label="Network devices" value={stats.totalDevices} />
      <StatCard label="Critical" value={stats.alertCount} tone={stats.alertCount > 0 ? 'alert' : ''} />
      <StatCard label="Caution" value={stats.cautionCount} tone={stats.cautionCount > 0 ? 'caution' : ''} />
      <StatCard
        label="Unknown impact"
        value={stats.unknownCount ?? 0}
        tone={(stats.unknownCount ?? 0) > 0 ? 'caution' : ''}
      />
    </div>
  )
}
