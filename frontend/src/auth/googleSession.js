import { apiUrl } from '../config/api.js'

async function requestJSON(path, options = {}, request = fetch) {
  const response = await request(apiUrl(path), { credentials: 'include', ...options })
  if (!response.ok) {
    const error = new Error(`Request failed with ${response.status}`)
    error.status = response.status
    throw error
  }
  return response.status === 204 ? null : response.json()
}

// getGoogleSessionConfig reads the non-secret client ID from the appliance API
// at runtime. It deliberately never relies on a browser build-time secret.
export async function getGoogleSessionConfig(request = fetch) {
  return requestJSON('/api/auth/method', {}, request)
}

// exchangeGoogleCredential hands the one-time GIS credential directly to the
// appliance API. The caller receives only the server session user profile.
export async function exchangeGoogleCredential(credential, request = fetch) {
  return requestJSON('/api/auth/google/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ credential }),
  }, request)
}

export async function restoreGoogleSession(request = fetch) {
  return requestJSON('/api/auth/me', {}, request)
}

export async function endGoogleSession(request = fetch) {
  return requestJSON('/api/auth/logout', { method: 'POST' }, request)
}

export function sessionUser(account) {
  return {
    email: account.email ?? '',
    name: account.name ?? account.username ?? account.email ?? 'Signed in',
    sub: account.subject ?? '',
  }
}
