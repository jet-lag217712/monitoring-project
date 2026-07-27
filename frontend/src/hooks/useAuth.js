import { useCallback, useEffect, useRef, useState } from 'react'
import { apiUrl, GOOGLE_CLIENT_ID, isApplianceAuth } from '../config/api.js'
import { setAuthTokenProvider, setCsrfTokenProvider } from '../services/sitesApi.js'

const GIS_SRC = 'https://accounts.google.com/gsi/client'
const TOKEN_STORAGE_KEY = 'ogsd_google_id_token'
const APPLIANCE_AUTH = isApplianceAuth()

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

function profileFromApplianceUser(username) {
  return {
    email: '',
    name: username,
    picture: null,
    sub: username,
  }
}

async function fetchApplianceSession() {
  let sessionUrl
  try {
    sessionUrl = apiUrl('/auth/me')
  } catch (err) {
    // #region agent log
    fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'56b40f'},body:JSON.stringify({sessionId:'56b40f',location:'useAuth.js:fetchApplianceSession',message:'apiUrl threw',data:{error:err?.message},timestamp:Date.now(),hypothesisId:'A'})}).catch(()=>{});
    // #endregion
    throw err
  }
  // #region agent log
  fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'56b40f'},body:JSON.stringify({sessionId:'56b40f',location:'useAuth.js:fetchApplianceSession',message:'fetching session',data:{sessionUrl},timestamp:Date.now(),hypothesisId:'B'})}).catch(()=>{});
  // #endregion
  const res = await fetch(sessionUrl, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  if (res.status === 401) {
    // #region agent log
    fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'56b40f'},body:JSON.stringify({sessionId:'56b40f',location:'useAuth.js:fetchApplianceSession',message:'session not authenticated',data:{status:res.status},timestamp:Date.now(),hypothesisId:'B'})}).catch(()=>{});
    // #endregion
    return null
  }
  if (!res.ok) {
    // #region agent log
    fetch('http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'56b40f'},body:JSON.stringify({sessionId:'56b40f',location:'useAuth.js:fetchApplianceSession',message:'session fetch failed',data:{status:res.status},timestamp:Date.now(),hypothesisId:'B'})}).catch(()=>{});
    // #endregion
    throw new Error('Failed to load session')
  }
  const data = await res.json()
  return {
    user: profileFromApplianceUser(data.username),
    csrfToken: data.csrf_token,
  }
}

/**
 * Authentication hook for Google OIDC or appliance-local cookie sessions.
 */
export function useAuth() {
  const [status, setStatus] = useState(() => {
    if (APPLIANCE_AUTH) return 'loading'
    return GOOGLE_CLIENT_ID ? 'loading' : 'unconfigured'
  })
  const [user, setUser] = useState(null)
  const [error, setError] = useState(null)
  const tokenRef = useRef(null)
  const csrfRef = useRef(null)
  const buttonHostRef = useRef(null)

  const clearSession = useCallback(() => {
    tokenRef.current = null
    csrfRef.current = null
    sessionStorage.removeItem(TOKEN_STORAGE_KEY)
    setCsrfTokenProvider(() => null)
    setUser(null)
    if (APPLIANCE_AUTH) {
      setStatus('signed_out')
      return
    }
    setStatus(GOOGLE_CLIENT_ID ? 'signed_out' : 'unconfigured')
  }, [])

  const applyApplianceSession = useCallback(session => {
    if (!session?.user) {
      clearSession()
      return false
    }
    csrfRef.current = session.csrfToken
    setCsrfTokenProvider(() => csrfRef.current)
    setUser(session.user)
    setStatus('signed_in')
    setError(null)
    return true
  }, [clearSession])

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
    if (APPLIANCE_AUTH) {
      setAuthTokenProvider(() => null)
      let cancelled = false

      async function initAppliance() {
        try {
          const session = await fetchApplianceSession()
          if (cancelled) return
          if (session) {
            applyApplianceSession(session)
          } else {
            setStatus('signed_out')
          }
        } catch (err) {
          if (!cancelled) {
            setError(err.message ?? 'Failed to initialize sign-in')
            setStatus('error')
          }
        }
      }

      initAppliance()
      return () => {
        cancelled = true
        setCsrfTokenProvider(() => null)
      }
    }

    setAuthTokenProvider(() => {
      const token = tokenRef.current
      if (token && tokenStillValid(token)) return token
      return null
    })
    return () => setAuthTokenProvider(() => null)
  }, [applyApplianceSession])

  useEffect(() => {
    if (APPLIANCE_AUTH || !GOOGLE_CLIENT_ID) {
      if (!APPLIANCE_AUTH && !GOOGLE_CLIENT_ID) {
        setStatus('unconfigured')
      }
      return undefined
    }

    let cancelled = false

    async function initGoogle() {
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

    initGoogle()
    return () => {
      cancelled = true
    }
  }, [applyToken])

  const renderButton = useCallback(element => {
    if (APPLIANCE_AUTH) return
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

  const signIn = useCallback(async (username, password) => {
    if (!APPLIANCE_AUTH) return false
    setError(null)
    const res = await fetch(apiUrl('/auth/login'), {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ username, password }),
    })
    if (res.status === 401 || res.status === 429) {
      const payload = await res.json().catch(() => null)
      const message = payload?.error?.message ?? 'Invalid username or password'
      setError(message)
      setStatus('signed_out')
      return false
    }
    if (!res.ok) {
      setError('Sign-in is temporarily unavailable')
      setStatus('error')
      return false
    }
    const data = await res.json()
    return applyApplianceSession({
      user: profileFromApplianceUser(data.username),
      csrfToken: data.csrf_token,
    })
  }, [applyApplianceSession])

  const signOut = useCallback(async () => {
    if (APPLIANCE_AUTH) {
      const csrf = csrfRef.current
      try {
        await fetch(apiUrl('/auth/logout'), {
          method: 'POST',
          credentials: 'include',
          headers: {
            Accept: 'application/json',
            ...(csrf ? { 'X-CSRF-Token': csrf } : {}),
          },
        })
      } catch {
        // Best-effort logout; clear local state regardless.
      }
      clearSession()
      return
    }

    clearSession()
    if (window.google?.accounts?.id) {
      window.google.accounts.id.disableAutoSelect()
    }
  }, [clearSession])

  return {
    error,
    renderButton,
    signIn,
    signOut,
    status,
    user,
    isAuthenticated: status === 'signed_in',
    isConfigured: APPLIANCE_AUTH ? true : Boolean(GOOGLE_CLIENT_ID),
    isAppliance: APPLIANCE_AUTH,
  }
}
