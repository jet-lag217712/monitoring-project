import { useCallback, useEffect, useRef, useState } from 'react'
import { API_BASE_URL, GOOGLE_CLIENT_ID, apiUrl } from '../config/api.js'
import { endGoogleSession, exchangeGoogleCredential, getGoogleSessionConfig, restoreGoogleSession, sessionUser } from '../auth/googleSession.js'
import { setAuthTokenProvider } from '../services/sitesApi.js'

const GIS_SRC = 'https://accounts.google.com/gsi/client'
const TOKEN_STORAGE_KEY = 'ogsd_google_id_token'

function loadGisScript() {
  if (window.google?.accounts?.id) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[src="${GIS_SRC}"]`)
    if (existing) {
      existing.addEventListener('load', resolve, { once: true })
      existing.addEventListener('error', () => reject(new Error('Failed to load Google Identity Services')), { once: true })
      return
    }
    const script = document.createElement('script')
    script.src = GIS_SRC
    script.async = true
    script.defer = true
    script.onload = resolve
    script.onerror = () => reject(new Error('Failed to load Google Identity Services'))
    document.head.appendChild(script)
  })
}

function decodeJwtPayload(token) {
  try {
    const [, payload] = token.split('.')
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

function tokenStillValid(token) {
  const claims = decodeJwtPayload(token)
  return Boolean(claims?.exp && claims.exp * 1000 > Date.now() + 60_000)
}

function profileFromToken(token) {
  const claims = decodeJwtPayload(token)
  if (!claims) return null
  return { email: claims.email ?? '', name: claims.name ?? claims.email ?? 'Signed in', picture: claims.picture ?? null, sub: claims.sub ?? '' }
}

// useAuth keeps Google ID tokens only for the legacy bearer mode. Appliance
// Google sign-in exchanges the GIS credential for a secure HTTP-only session.
export function useAuth() {
  const [status, setStatus] = useState('loading')
  const [user, setUser] = useState(null)
  const [error, setError] = useState(null)
  const [provider, setProvider] = useState(null)
  const [googleClientID, setGoogleClientID] = useState('')
  const tokenRef = useRef(null)

  const clearLegacyGoogle = useCallback(() => {
    tokenRef.current = null
    sessionStorage.removeItem(TOKEN_STORAGE_KEY)
  }, [])

  const startGoogle = useCallback(async (clientID, sessionBacked) => {
    if (!clientID) {
      setStatus('unconfigured')
      setError('Google Client ID is not configured.')
      return
    }
    if (!sessionBacked) {
      const cached = sessionStorage.getItem(TOKEN_STORAGE_KEY)
      if (cached && tokenStillValid(cached)) {
        tokenRef.current = cached
        setUser(profileFromToken(cached))
        setStatus('signed_in')
      }
    }
    try {
      await loadGisScript()
      window.google.accounts.id.initialize({
        client_id: clientID,
        callback: async response => {
          if (!response?.credential || !tokenStillValid(response.credential)) {
            setError('Google sign-in failed')
            setStatus('signed_out')
            return
          }
          if (!sessionBacked) {
            tokenRef.current = response.credential
            sessionStorage.setItem(TOKEN_STORAGE_KEY, response.credential)
            setUser(profileFromToken(response.credential))
            setStatus('signed_in')
            setError(null)
            return
          }
          try {
            const account = await exchangeGoogleCredential(response.credential)
            setUser(sessionUser(account))
            setStatus('signed_in')
            setError(null)
          } catch (err) {
            setError(err.status === 401 ? 'Your Google Workspace account is not allowed.' : 'Google sign-in is unavailable.')
            setStatus('signed_out')
          }
        },
        auto_select: false,
        cancel_on_tap_outside: true,
      })
      if (sessionBacked || !tokenRef.current) setStatus('signed_out')
    } catch (err) {
      setError(err.message ?? 'Google sign-in failed to initialize')
      setStatus('error')
    }
  }, [])

  useEffect(() => {
    setAuthTokenProvider(() => (provider === 'google_bearer' && tokenRef.current && tokenStillValid(tokenRef.current) ? tokenRef.current : null))
    return () => setAuthTokenProvider(() => null)
  }, [provider])

  useEffect(() => {
    let cancelled = false
    async function restoreSession() {
      try {
        const account = await restoreGoogleSession()
        if (!cancelled) {
          setUser(sessionUser(account))
          setStatus('signed_in')
        }
      } catch (err) {
        if (!cancelled && err.status === 401) setStatus('signed_out')
        else if (!cancelled) throw err
      }
    }
    async function initialize() {
      try {
        const method = await getGoogleSessionConfig()
        if (cancelled) return
        setProvider(method.provider)
        if (method.provider === 'local' || method.provider === 'oidc') {
          await restoreSession()
          return
        }
        if (method.provider === 'google_session') {
          const clientID = method.client_id ?? ''
          setGoogleClientID(clientID)
          if (!clientID) {
            setStatus('unconfigured')
            setError('Google Client ID is not configured.')
            return
          }
          try {
            const account = await restoreGoogleSession()
            if (!cancelled) {
              setUser(sessionUser(account))
              setStatus('signed_in')
            }
            return
          } catch (err) {
            if (err.status !== 401) throw err
          }
          if (!cancelled) await startGoogle(clientID, true)
          return
        }
        if (method.provider === 'setup_required') {
          setStatus('unconfigured')
          return
        }
      } catch {
        // Legacy cloud deployments expose Google bearer auth without a public
        // authentication-method endpoint.
      }
      if (!cancelled) {
        setProvider('google_bearer')
        setGoogleClientID(GOOGLE_CLIENT_ID)
        await startGoogle(GOOGLE_CLIENT_ID, false)
      }
    }
    void initialize()
    return () => { cancelled = true }
  }, [startGoogle])

  const signInLocal = useCallback(async (username, password) => {
    setError(null)
    setStatus('loading')
    try {
      const response = await fetch(apiUrl('/api/auth/local/login'), {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!response.ok) {
        const err = new Error(`Request failed with ${response.status}`)
        err.status = response.status
        throw err
      }
      const account = await response.json()
      setUser(sessionUser(account))
      setStatus('signed_in')
    } catch (err) {
      setError(err.status === 401 ? 'Invalid username or password.' : 'Sign-in is unavailable.')
      setStatus('signed_out')
    }
  }, [])

  const startOIDC = useCallback(() => {
    window.location.assign(apiUrl('/api/auth/oidc/start'))
  }, [])

  const renderButton = useCallback(element => {
    const googleProvider = provider === 'google_session' || provider === 'google_bearer'
    if (!element || !googleProvider || !window.google?.accounts?.id || !googleClientID) return
    element.innerHTML = ''
    window.google.accounts.id.renderButton(element, { type: 'standard', theme: 'outline', size: 'large', text: 'signin_with', shape: 'rectangular', width: 280 })
  }, [googleClientID, provider])

  const signOut = useCallback(() => {
    if (provider === 'local' || provider === 'google_session' || provider === 'oidc') {
      void endGoogleSession()
    }
    clearLegacyGoogle()
    if (window.google?.accounts?.id) window.google.accounts.id.disableAutoSelect()
    setUser(null)
    setStatus('signed_out')
  }, [clearLegacyGoogle, provider])

  return {
    error,
    provider,
    renderButton,
    signInLocal,
    startOIDC,
    signOut,
    status,
    user,
    isAuthenticated: status === 'signed_in',
    isConfigured: provider !== null,
    apiBaseUrl: API_BASE_URL,
  }
}
