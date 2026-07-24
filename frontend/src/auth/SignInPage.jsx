import { useEffect, useRef, useState } from 'react'

export default function SignInPage({ auth }) {
  const buttonRef = useRef(null)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  useEffect(() => {
    if (buttonRef.current && auth.status === 'signed_out') auth.renderButton(buttonRef.current)
  }, [auth])

  const submit = event => {
    event.preventDefault()
    void auth.signInLocal(username, password)
  }

  const local = auth.provider === 'local'
  return (
    <div className="sign-in-page">
      <div className="sign-in-panel">
        <div className="page-eyebrow"><span className="eyebrow-dot" />Equate Monitoring</div>
        <h1 className="page-title">Sign in</h1>
        <p className="page-sub">{local ? 'Use the administrator account configured on the appliance.' : 'Use the configured identity provider to access the dashboard.'}</p>

        {auth.status === 'loading' && <p className="sign-in-status">Loading sign-in…</p>}
        {local && auth.status !== 'loading' && (
          <form onSubmit={submit} className="sign-in-form">
            <label>Username<input autoComplete="username" value={username} onChange={event => setUsername(event.target.value)} required /></label>
            <label>Password<input type="password" autoComplete="current-password" value={password} onChange={event => setPassword(event.target.value)} required /></label>
            <button type="submit">Sign in</button>
          </form>
        )}
        {auth.provider === 'oidc' && auth.status === 'signed_out' && <button type="button" onClick={auth.startOIDC}>Continue with identity provider</button>}
        {(auth.provider === 'google_session' || auth.provider === 'google_bearer') && auth.status === 'signed_out' && <div ref={buttonRef} className="sign-in-button-host" />}
        {auth.provider === 'setup_required' && <p className="sign-in-status sign-in-error">Complete local SNMP setup before dashboard access.</p>}
        {auth.status === 'unconfigured' && auth.provider !== 'setup_required' && <p className="sign-in-status sign-in-error">Authentication is not configured.</p>}
        {auth.error && <p className="sign-in-status sign-in-error">{auth.error}</p>}
      </div>
    </div>
  )
}
