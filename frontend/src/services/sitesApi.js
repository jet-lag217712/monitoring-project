import { apiUrl } from '../config/api.js'

let authTokenProvider = () => null

/** Register a function that returns the current Google ID token (or null). */
export function setAuthTokenProvider(provider) {
  authTokenProvider = typeof provider === 'function' ? provider : () => null
}

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function fetchJson(path, errorMessage) {
  const headers = { Accept: 'application/json' }
  const token = authTokenProvider()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const res = await fetch(apiUrl(path), { headers })

  if (res.status === 401) {
    throw new ApiError('Unauthorized', 401)
  }

  if (!res.ok) {
    throw new ApiError(`${errorMessage} with ${res.status}`, res.status)
  }

  return res.json()
}

export function fetchSitesFromApi() {
  return fetchJson('/api/sites', 'Site list request failed')
}

export function fetchSiteDetailFromApi(siteId) {
  return fetchJson(`/api/sites/${encodeURIComponent(siteId)}`, 'Site detail request failed')
}

export function fetchTestConfigFromApi() {
  return fetchJson('/api/test-config', 'Test config request failed')
}

export function fetchDeviceFromApi(deviceId, siteId) {
  const qs = siteId ? `?siteId=${encodeURIComponent(siteId)}` : ''
  return fetchJson(
    `/api/devices/${encodeURIComponent(deviceId)}${qs}`,
    'Device request failed',
  )
}

export function fetchDeviceInterfacesFromApi(deviceId, siteId) {
  const qs = siteId ? `?siteId=${encodeURIComponent(siteId)}` : ''
  return fetchJson(
    `/api/devices/${encodeURIComponent(deviceId)}/interfaces${qs}`,
    'Interfaces request failed',
  )
}

export function fetchDeviceMetricsFromApi(deviceId, { siteId, metric, start, end } = {}) {
  const params = new URLSearchParams()
  if (siteId) params.set('siteId', siteId)
  if (metric) params.set('metric', metric)
  if (start) params.set('start', start)
  if (end) params.set('end', end)
  const qs = params.toString() ? `?${params}` : ''
  return fetchJson(
    `/api/devices/${encodeURIComponent(deviceId)}/metrics${qs}`,
    'Metrics request failed',
  )
}

export function fetchAlertsFromApi() {
  return fetchJson('/api/alerts', 'Alerts request failed')
}
