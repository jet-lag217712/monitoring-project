export function formatBytes(bytes) {
  if (bytes == null || Number.isNaN(bytes)) return '—'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / 1024 ** i
  return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

export function formatUptime(days) {
  const value = Number(days)
  if (days == null || Number.isNaN(value)) return '—'
  return `${value.toFixed(2)} days`
}

export function formatSysUpTime(centiseconds) {
  if (centiseconds == null || Number.isNaN(centiseconds)) return '—'
  const totalSeconds = Math.floor(centiseconds / 100)
  const days = Math.floor(totalSeconds / 86400)
  const remainder = totalSeconds % 86400
  const hours = Math.floor(remainder / 3600)
  const minutes = Math.floor((remainder % 3600) / 60)
  const seconds = remainder % 60
  const pad = n => String(n).padStart(2, '0')
  return `${days} days, ${hours}:${pad(minutes)}:${pad(seconds)}`
}

export function formatTimestamp(iso) {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return '—'
  }
}

export function formatNumber(n) {
  if (n == null || Number.isNaN(n)) return '—'
  return n.toLocaleString()
}

/** Format a utilization percentage for display (bar width still uses the raw value). */
export function formatPercent(value, decimals = 2) {
  const n = Number(value)
  if (value == null || Number.isNaN(n)) return '—'
  return n.toFixed(decimals)
}
