import { execSync } from 'node:child_process'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Same source as equate's buildVersion (Makefile VERSION / release --version).
// Release builds pass VITE_APP_VERSION explicitly; local/dev falls back to git describe.
function resolveAppVersion() {
  const fromEnv = process.env.VITE_APP_VERSION?.trim()
  if (fromEnv) return fromEnv.replace(/^v/, '')
  try {
    return execSync('git describe --tags --always --dirty', {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore'],
    })
      .trim()
      .replace(/^v/, '')
  } catch {
    return 'unknown'
  }
}

const appVersion = resolveAppVersion()
process.env.VITE_APP_VERSION = appVersion

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'node',
    include: ['src/**/*.test.js'],
  },
})
