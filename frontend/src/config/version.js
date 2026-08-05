// Mirrors equate CLI buildVersion (ldflags -X main.buildVersion=...).
// Injected as VITE_APP_VERSION at build time; defaults to "unknown" like equate.
export const APP_VERSION = String(import.meta.env.VITE_APP_VERSION ?? 'unknown').replace(/^v/, '')
