import { apiUrl } from '../config/api.js'

async function fetchJson(path, errorMessage) {
  const res = await fetch(apiUrl(path))

  if (!res.ok) {
    throw new Error(`${errorMessage} with ${res.status}`)
  }

  return res.json()
}

export function fetchSitesFromApi() {
  return fetchJson('/api/sites', 'Site list request failed')
}

export function fetchSiteDetailFromApi(siteId) {
  return fetchJson(`/api/sites/${siteId}`, 'Site detail request failed')
}

export function fetchTestConfigFromApi() {
  return fetchJson('/api/test-config', 'Test config request failed')
}
