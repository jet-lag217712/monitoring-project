import { describe, expect, it } from 'vitest'
import { endGoogleSession, exchangeGoogleCredential, getGoogleSessionConfig, restoreGoogleSession, sessionUser } from './googleSession.js'

function response(status, body = null) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  }
}

describe('appliance Google session flow', () => {
  it('loads the managed Google client configuration at runtime', async () => {
    const method = await getGoogleSessionConfig(async (url, options) => {
      expect(url).toContain('/api/auth/method')
      expect(options.credentials).toBe('include')
      return response(200, { provider: 'google_session', client_id: 'equate.apps.googleusercontent.com' })
    })
    expect(method).toEqual({ provider: 'google_session', client_id: 'equate.apps.googleusercontent.com' })
  })

  it('exchanges the GIS credential without retaining a browser token', async () => {
    const account = await exchangeGoogleCredential('google-id-token', async (url, options) => {
      expect(url).toContain('/api/auth/google/login')
      expect(options.credentials).toBe('include')
      expect(options.method).toBe('POST')
      expect(options.headers).toEqual({ 'Content-Type': 'application/json' })
      expect(options.body).toBe(JSON.stringify({ credential: 'google-id-token' }))
      return response(200, { subject: 'operator', email: 'operator@example.com', name: 'Operator' })
    })
    expect(sessionUser(account)).toEqual({ sub: 'operator', email: 'operator@example.com', name: 'Operator' })
  })

  it('restores and ends only the server session', async () => {
    const request = async (url, options) => {
      expect(options.credentials).toBe('include')
      if (url.includes('/api/auth/me')) return response(200, { subject: 'operator', email: 'operator@example.com', name: 'Operator' })
      expect(url).toContain('/api/auth/logout')
      expect(options.method).toBe('POST')
      return response(204)
    }
    await expect(restoreGoogleSession(request)).resolves.toMatchObject({ subject: 'operator' })
    await expect(endGoogleSession(request)).resolves.toBeNull()
  })
})
