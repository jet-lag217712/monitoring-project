import { useCallback, useEffect, useRef, useState } from 'react'
import { GOOGLE_CLIENT_ID } from '../config/api.js'
import { setAuthTokenProvider } from '../services/sitesApi.js'

const GIS_SRC = 'https://accounts.google.com/gsi/client'
const TOKEN_STORAGE_KEY = 'ogsd_google_id_token'

function loadGisScript() {
  if (typeof window === 'undefined') {
    return Promise.reject(new Error('window unavailable'))
  }
  if (window.google?.accounts?.id) {
    return Promise.resolve()
  }

  const existing = document.querySelector(`script[src="${GIS_SRC}"]`)
  if (existing) {
    return new Promise((resolve, reject) => {
      existing.addEventListener('load', () => resolve())
      existing.addEventListener('error', () => reject(new Error('Failed to load Google Identity Services')))
    })
  }

  return new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = GIS_SRC
    script.async = true
    script.defer = true
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('Failed to load Google Identity Services'))
    document.head.appendChild(script)
  })
}

function decodeJwtPayload(token) {
  try {
    const [, payload] = token.split('.')
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/')
    const json = atob(normalized)
    return JSON.parse(json)
  } catch {
    return null
  }
}

function tokenStillValid(token) {
  const claims = decodeJwtPayload(token)
  if (!claims?.exp) return false
  // Refresh a minute early.
  return claims.exp * 1000 > Date.now() + 60_000
}

function profileFromToken(token) {
  const claims = decodeJwtPayload(token)
  if (!claims) return null
  return {
    email: claims.email ?? '',
    name: claims.name ?? claims.email ?? 'Signed in',
    picture: claims.picture ?? null,
    sub: claims.sub ?? '',
  }
}

/**
 * Google Identity Services (GIS) sign-in. ID token is kept in memory and
 * sessionStorage so a refresh does not force an immediate re-login during MVP.
 */
export function useAuth() {
  const [status, setStatus] = useState(() => (GOOGLE_CLIENT_ID ? 'loading' : 'unconfigured'))
  const [user, setUser] = useState(null)
  const [error, setError] = useState(null)
  const tokenRef = useRef(null)
  const buttonHostRef = useRef(null)

  const clearSession = useCallback(() => {
    tokenRef.current = null
    sessionStorage.removeItem(TOKEN_STORAGE_KEY)
    setUser(null)
    setStatus(GOOGLE_CLIENT_ID ? 'signed_out' : 'unconfigured')
  }, [])

  const applyToken = useCallback(credential => {
    if (!credential || !tokenStillValid(credential)) {
      clearSession()
      return false
    }
    tokenRef.current = credential
    sessionStorage.setItem(TOKEN_STORAGE_KEY, credential)
    setUser(profileFromToken(credential))
    setStatus('signed_in')
    setError(null)
    return true
  }, [clearSession])

  useEffect(() => {
    setAuthTokenProvider(() => {
      const token = tokenRef.current
      if (token && tokenStillValid(token)) return token
      return null
    })
    return () => setAuthTokenProvider(() => null)
  }, [])

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) {
      setStatus('unconfigured')
      return undefined
    }

    let cancelled = false

    async function init() {
      const cached = sessionStorage.getItem(TOKEN_STORAGE_KEY)
      if (cached && applyToken(cached)) {
        // Still initialize GIS for One Tap / button re-render after sign-out.
      }

      try {
        await loadGisScript()
        if (cancelled) return

        window.google.accounts.id.initialize({
          client_id: GOOGLE_CLIENT_ID,
          callback: response => {
            if (!response?.credential) {
              setError('Google sign-in returned no credential')
              return
            }
            applyToken(response.credential)
          },
          auto_select: false,
          cancel_on_tap_outside: true,
        })

        if (!tokenRef.current) {
          setStatus('signed_out')
        }
      } catch (err) {
        if (!cancelled) {
          setError(err.message ?? 'Google sign-in failed to initialize')
          setStatus('error')
        }
      }
    }

    init()
    return () => {
      cancelled = true
    }
  }, [applyToken])

  const renderButton = useCallback(element => {
    buttonHostRef.current = element
    if (!element || !window.google?.accounts?.id || !GOOGLE_CLIENT_ID) return
    element.innerHTML = ''
    window.google.accounts.id.renderButton(element, {
      type: 'standard',
      theme: 'outline',
      size: 'large',
      text: 'signin_with',
      shape: 'rectangular',
      width: 280,
    })
  }, [])

  const signOut = useCallback(() => {
    clearSession()
    if (window.google?.accounts?.id) {
      window.google.accounts.id.disableAutoSelect()
    }
  }, [clearSession])

  return {
    error,
    renderButton,
    signOut,
    status,
    user,
    isAuthenticated: status === 'signed_in',
    isConfigured: Boolean(GOOGLE_CLIENT_ID),
  }
}
