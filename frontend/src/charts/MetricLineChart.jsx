import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { getUtilizationClass } from '../common/UtilizationBar.jsx'

const TICK_STYLE = {
  fontFamily: "'JetBrains Mono', monospace",
  fontSize: '0.58rem',
  fill: 'var(--ink-muted)',
}

function formatTime(ts) {
  try {
    return new Date(ts).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ''
  }
}

function getStrokeColor(metric, value) {
  if (metric === 'temperature') {
    return value > 65 ? 'var(--status-caution)' : 'var(--salmon)'
  }
  const cls = getUtilizationClass(value ?? 0)
  if (cls === 'critical') return 'var(--status-alert)'
  if (cls === 'low') return 'var(--status-caution)'
  return 'var(--status-ok)'
}

export default function MetricLineChart({ title, data, metric = 'cpu', unit = '%', yDomain }) {
  const latest = data?.length ? data[data.length - 1].value : 0
  const stroke = getStrokeColor(metric, latest)

  return (
    <div className="chart-card">
      <div className="chart-card-label">{title}</div>
      <div className="chart-card-body">
        <ResponsiveContainer width="100%" height={140}>
          <LineChart data={data} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
            <CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="ts"
              tickFormatter={formatTime}
              tick={TICK_STYLE}
              axisLine={false}
              tickLine={false}
              interval="preserveStartEnd"
            />
            <YAxis
              domain={yDomain ?? [0, 100]}
              tick={TICK_STYLE}
              axisLine={false}
              tickLine={false}
              width={36}
              tickFormatter={v => `${v}${unit === '%' ? '' : ''}`}
            />
            <Tooltip
              contentStyle={{
                background: 'var(--bg)',
                border: '1px solid var(--border)',
                borderRadius: 'var(--radius)',
                fontFamily: "'JetBrains Mono', monospace",
                fontSize: '0.68rem',
              }}
              labelFormatter={formatTime}
              formatter={value => [`${value}${unit}`, title]}
            />
            <Line
              type="monotone"
              dataKey="value"
              stroke={stroke}
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
