import MetricLineChart from '../charts/MetricLineChart.jsx'

export default function DeviceMetricsCharts({ history }) {
  return (
    <div className="device-metrics-grid">
      <MetricLineChart title="CPU Utilization" data={history?.cpu ?? []} metric="cpu" unit="%" />
      <MetricLineChart title="RAM Utilization" data={history?.memory ?? []} metric="memory" unit="%" />
      <MetricLineChart
        title="Device Temperature"
        data={history?.temperature ?? []}
        metric="temperature"
        unit="°C"
        yDomain={['auto', 'auto']}
      />
    </div>
  )
}
