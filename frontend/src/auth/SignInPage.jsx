import { useEffect, useRef, useState } from 'react'

export default function SignInPage({ auth }) {
  const buttonRef = useRef(null)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!auth.isAppliance && buttonRef.current && auth.status === 'signed_out') {
      auth.renderButton(buttonRef.current)
    }
  }, [auth])

  async function handleLocalSubmit(event) {
    event.preventDefault()
    if (!auth.signIn || submitting) return
    setSubmitting(true)
    try {
      await auth.signIn(username, password)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="sign-in-page">
      <div className="sign-in-panel">
        <div className="page-eyebrow">
          <span className="eyebrow-dot" />
          Equate Monitoring
        </div>
        <h1 className="page-title">Sign in</h1>
        <p className="page-sub">
          {auth.isAppliance
            ? 'Use your appliance username and password to access the local monitoring dashboard.'
            : 'Use your Google account to access the live monitoring dashboard.'}
        </p>

        {auth.status === 'loading' && (
          <p className="sign-in-status">Loading sign-in…</p>
        )}

        {!auth.isAppliance && auth.status === 'unconfigured' && (
          <p className="sign-in-status sign-in-error">
            Google Client ID is not configured. Set <code>VITE_GOOGLE_CLIENT_ID</code>.
          </p>
        )}

        {auth.status === 'error' && (
          <p className="sign-in-status sign-in-error">{auth.error}</p>
        )}

        {auth.isAppliance && auth.status === 'signed_out' && (
          <form className="sign-in-form" onSubmit={handleLocalSubmit}>
            <label className="sign-in-field">
              <span>Username</span>
              <input
                type="text"
                name="username"
                autoComplete="username"
                value={username}
                onChange={event => setUsername(event.target.value)}
                disabled={submitting}
                required
              />
            </label>
            <label className="sign-in-field">
              <span>Password</span>
              <input
                type="password"
                name="password"
                autoComplete="current-password"
                value={password}
                onChange={event => setPassword(event.target.value)}
                disabled={submitting}
                required
              />
            </label>
            <button type="submit" className="sign-in-submit" disabled={submitting}>
              {submitting ? 'Signing in…' : 'Sign in'}
            </button>
          </form>
        )}

        {!auth.isAppliance && auth.status === 'signed_out' && (
          <div ref={buttonRef} className="sign-in-button-host" />
        )}

        {auth.error && auth.status === 'signed_out' && (
          <p className="sign-in-status sign-in-error">{auth.error}</p>
        )}
      </div>
    </div>
  )
}
