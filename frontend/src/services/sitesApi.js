import { apiUrl, isApplianceAuth } from '../config/api.js'

let authTokenProvider = () => null
let csrfTokenProvider = () => null

/** Register a function that returns the current Google ID token (or null). */
export function setAuthTokenProvider(provider) {
  authTokenProvider = typeof provider === 'function' ? provider : () => null
}

/** Register a function that returns the current appliance CSRF token (or null). */
export function setCsrfTokenProvider(provider) {
  csrfTokenProvider = typeof provider === 'function' ? provider : () => null
}

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function fetchJson(path, errorMessage, options = {}) {
  const headers = { Accept: 'application/json', ...(options.headers ?? {}) }
  const fetchOptions = { ...options, headers }

  if (isApplianceAuth()) {
    fetchOptions.credentials = 'include'
    const csrf = csrfTokenProvider()
    if (csrf) {
      headers['X-CSRF-Token'] = csrf
    }
  } else {
    const token = authTokenProvider()
    if (token) {
      headers.Authorization = `Bearer ${token}`
    }
  }

  const url = apiUrl(path)
  const res = await fetch(url, fetchOptions)

  if (res.status === 401) {
    // #region agent log
    fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'56b40f'},body:JSON.stringify({sessionId:'56b40f',location:'sitesApi.js:fetchJson',message:'API 401 unauthorized',data:{path,url},timestamp:Date.now(),hypothesisId:'D'})}).catch(()=>{});
    // #endregion
    throw new ApiError('Unauthorized', 401)
  }

  if (!res.ok) {
    // #region agent log
    fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'56b40f'},body:JSON.stringify({sessionId:'56b40f',location:'sitesApi.js:fetchJson',message:'API error response',data:{path,url,status:res.status},timestamp:Date.now(),hypothesisId:'E'})}).catch(()=>{});
    // #endregion
    throw new ApiError(`${errorMessage} with ${res.status}`, res.status)
  }

  const json = await res.json()
  // #region agent log
  fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'56b40f'},body:JSON.stringify({sessionId:'56b40f',location:'sitesApi.js:fetchJson',message:'API success',data:{path,url,topLevelKeys:json&&typeof json==='object'?Object.keys(json).slice(0,10):[],deviceCount:path.includes('/sites/')&&json?.latest?.devices?Object.keys(json.latest.devices).length:undefined,siteCount:path==='/api/sites'&&json?Object.keys(json).length:undefined},timestamp:Date.now(),hypothesisId:'F'})}).catch(()=>{});
  // #endregion
  return json
}

export function fetchSitesFromApi() {
  return fetchJson('/api/sites', 'Site list request failed')
}

export function fetchSiteDetailFromApi(siteId) {
  return fetchJson(`/api/sites/${encodeURIComponent(siteId)}`, 'Site detail request failed')
}

export function updateSiteLocation(siteId, location) {
  return fetchJson(`/api/sites/${encodeURIComponent(siteId)}`, 'Site rename request failed', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ location }),
  })
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
