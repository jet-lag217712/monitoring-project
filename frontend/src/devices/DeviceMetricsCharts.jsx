import MetricLineChart from '../charts/MetricLineChart.jsx'

function EmptyChart({ title, message }) {
  return (
    <div className="chart-card">
      <div className="chart-card-label">{title}</div>
      <div className="chart-card-body chart-card-empty">
        <p>{message}</p>
      </div>
    </div>
  )
}

export default function DeviceMetricsCharts({ history }) {
  const cpu = history?.cpu ?? []
  const memory = history?.memory ?? []
  const temperature = history?.temperature ?? []
  const uptime = history?.uptime ?? []

  return (
    <div className="device-metrics-grid">
      {cpu.length > 0 ? (
        <MetricLineChart title="CPU Utilization" data={cpu} metric="cpu" unit="%" />
      ) : (
        <EmptyChart title="CPU Utilization" message="No CPU telemetry for this device yet." />
      )}
      {memory.length > 0 ? (
        <MetricLineChart title="RAM Utilization" data={memory} metric="memory" unit="%" />
      ) : (
        <EmptyChart title="RAM Utilization" message="No memory telemetry for this device yet." />
      )}
      {temperature.length > 0 ? (
        <MetricLineChart
          title="Device Temperature"
          data={temperature}
          metric="temperature"
          unit="°C"
          yDomain={['auto', 'auto']}
        />
      ) : uptime.length > 0 ? (
        <MetricLineChart
          title="Uptime (seconds)"
          data={uptime}
          metric="uptime"
          unit="s"
          yDomain={['auto', 'auto']}
        />
      ) : (
        <EmptyChart title="Device Temperature" message="No temperature telemetry for this device yet." />
      )}
    </div>
  )
}
