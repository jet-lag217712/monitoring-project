export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:8000'
export const POLL_INTERVAL_MS = 5000
export const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID ?? ''
export const DEMO_ENABLED = String(import.meta.env.VITE_DEMO_ENABLED ?? 'false').toLowerCase() === 'true'

export function apiUrl(path) {
  return new URL(path, API_BASE_URL).toString()
}
