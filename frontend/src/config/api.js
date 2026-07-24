// The appliance is served through its nginx front door, so its default API
// origin is the browser origin. Development can still point at a standalone
// API with VITE_API_BASE_URL.
const browserOrigin = typeof window === 'undefined' ? 'http://127.0.0.1:8000' : window.location.origin
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? browserOrigin
export const POLL_INTERVAL_MS = 5000
export const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID ?? ''
export const DEMO_ENABLED = String(import.meta.env.VITE_DEMO_ENABLED ?? 'false').toLowerCase() === 'true'

export function apiUrl(path) {
  return new URL(path, API_BASE_URL).toString()
}
