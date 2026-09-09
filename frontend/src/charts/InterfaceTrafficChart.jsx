import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

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

export default function InterfaceTrafficChart({ data, interfaceName }) {
  if (!data?.length) {
    return (
      <div className="chart-card chart-card-compact">
        <div className="chart-card-label">Traffic — {interfaceName ?? '—'}</div>
        <div className="sparkline-placeholder" style={{ width: '100%' }}>No data</div>
      </div>
    )
  }

  return (
    <div className="chart-card chart-card-compact">
      <div className="chart-card-label">Traffic — {interfaceName}</div>
      <div className="chart-card-body">
        <ResponsiveContainer width="100%" height={120}>
          <LineChart data={data} margin={{ top: 8, right: 8, left: -12, bottom: 0 }}>
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
              tick={TICK_STYLE}
              axisLine={false}
              tickLine={false}
              width={32}
              tickFormatter={v => `${v}`}
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
            />
            <Legend
              wrapperStyle={{
                fontFamily: "'JetBrains Mono', monospace",
                fontSize: '0.6rem',
                textTransform: 'uppercase',
              }}
            />
            <Line
              type="monotone"
              dataKey="in_mbps"
              name="In (Mbps)"
              stroke="var(--salmon)"
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
            />
            <Line
              type="monotone"
              dataKey="out_mbps"
              name="Out (Mbps)"
              stroke="var(--ink-muted)"
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
