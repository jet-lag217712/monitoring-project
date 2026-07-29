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
  // #region agent log
  if (interfaceName === 'Gi0/0' || data?.length) {
    fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'269a95'},body:JSON.stringify({sessionId:'269a95',location:'InterfaceTrafficChart.jsx:render',message:'chart data received',data:{interfaceName,dataCount:data?.length??0,firstPoint:data?.[0]??null,hasInMbps:data?.[0]?.in_mbps!=null,hasOutMbps:data?.[0]?.out_mbps!=null},timestamp:Date.now(),runId:'post-fix',hypothesisId:'A'})}).catch(()=>{});
  }
  // #endregion

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
