import { useEffect, useRef } from 'react'

export default function SignInPage({ auth }) {
  const buttonRef = useRef(null)

  useEffect(() => {
    if (buttonRef.current && auth.status === 'signed_out') {
      auth.renderButton(buttonRef.current)
    }
  }, [auth])

  return (
    <div className="sign-in-page">
      <div className="sign-in-panel">
        <div className="page-eyebrow">
          <span className="eyebrow-dot" />
          Equate Monitoring
        </div>
        <h1 className="page-title">Sign in</h1>
        <p className="page-sub">
          Use your Google account to access the live monitoring dashboard.
        </p>

        {auth.status === 'loading' && (
          <p className="sign-in-status">Loading Google sign-in…</p>
        )}

        {auth.status === 'unconfigured' && (
          <p className="sign-in-status sign-in-error">
            Google Client ID is not configured. Set <code>VITE_GOOGLE_CLIENT_ID</code>.
          </p>
        )}

        {auth.status === 'error' && (
          <p className="sign-in-status sign-in-error">{auth.error}</p>
        )}

        {auth.status === 'signed_out' && <div ref={buttonRef} className="sign-in-button-host" />}

        {auth.error && auth.status === 'signed_out' && (
          <p className="sign-in-status sign-in-error">{auth.error}</p>
        )}
      </div>
    </div>
  )
}
