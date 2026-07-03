export default function LastUpdatedLabel({ lastUpdated }) {
  if (!lastUpdated) return null

  return (
    <span style={{
      fontFamily: "'JetBrains Mono', monospace",
      fontSize: '0.68rem',
      color: 'var(--ink-muted)',
    }}>
      Updated {lastUpdated}
    </span>
  )
}
