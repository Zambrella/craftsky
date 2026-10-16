import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { createRoot } from 'react-dom/client'
import { ApiError, browserToken, identity, login, logout, type Operator } from './api'
import './style.css'
import { OperatorWorkspace } from './workspaces/OperatorWorkspace'

function App() {
  const [operator, setOperator] = useState<Operator | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<'login' | 'logout' | null>(null)
  const [identifier, setIdentifier] = useState('')
  const [error, setError] = useState('')
  const [workspaceLocked, setWorkspaceLocked] = useState(false)
  const sessionExpired = useCallback(() => { setOperator(null); setWorkspaceLocked(false); setError('Your access changed or your session expired. Sign in again. If an action was interrupted, review case history before submitting another command.') }, [])
  async function refresh(background = false) {
    if (!background) { setLoading(true); setError('') }
    try { setOperator(await identity()) }
    catch (err) {
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) setOperator(null)
      else { if (!background) setOperator(null); setError('The console is unavailable. Please try again.') }
    } finally { setLoading(false) }
  }
  useEffect(() => { void refresh() }, [])
  useEffect(() => {
    const onFocus = () => { if (!busy && !workspaceLocked) void refresh(true) }
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [busy, workspaceLocked])
  async function signIn(event: FormEvent) {
    event.preventDefault(); if (busy) return
    setBusy('login'); setError('')
    try {
      const result = await login(identifier.trim(), await browserToken())
      const target = new URL(result.authorizationUrl)
      if (target.protocol !== 'https:') throw new Error('Invalid provider URL')
      window.location.assign(target.href)
    } catch (err) {
      setError(err instanceof ApiError && err.code === 'operator_access_denied' ? 'This account does not have access to the moderation console.' : 'Sign-in could not start. Please try again.')
      setBusy(null)
    }
  }
  async function signOut() {
    if (busy || workspaceLocked) return
    setBusy('logout'); setError('')
    try { await logout(await browserToken()); setOperator(null) }
    catch (err) { if (err instanceof ApiError && err.status === 401) setOperator(null); else setError('Sign-out could not be completed. Please try again.') }
    finally { setBusy(null) }
  }
  return <div className="app">
    <header><a className="brand" href="/admin/" aria-label="CraftSky moderation home" aria-disabled={workspaceLocked} onClick={event => { if (workspaceLocked) event.preventDefault() }}><span className="mark" aria-hidden="true">✳</span>CraftSky <span className="brand-detail">Moderation</span></a><span className="private">Operator console</span></header>
    <main className={operator ? 'operator-main' : ''}>
      {loading ? <div className="loading" role="status"><span className="spinner"/>Checking your session…</div> : operator ? <>
        <div className="eyebrow">YOUR WORKSPACE</div><div className="heading-row"><div><h1>Moderation console</h1><p className="intro">Review reports, record decisions and follow up on appeals.</p></div><button disabled={busy !== null || workspaceLocked} className="secondary" onClick={() => void signOut()}>{busy === 'logout' && <span className="spinner"/>}{busy === 'logout' ? 'Signing out…' : 'Sign out'}</button></div>
        <details className="access-details"><summary>Your access · {operator.permissionSet === 'mod' ? 'Moderator' : operator.permissionSet === 'admin' ? 'Administrator' : 'Helper'}</summary><dl><dt>Atmosphere account</dt><dd className="did">{operator.did}</dd><dt>Available permissions</dt><dd>{Object.entries(operator.permissions).filter(([, allowed]) => allowed).length}</dd>{operator.permissionSet === 'helper' && <><dt>Assigned incidents</dt><dd>{operator.assignedIncidentReferences.length}</dd></>}</dl></details>
        <OperatorWorkspace operator={operator} onSessionExpired={sessionExpired} onLocked={setWorkspaceLocked}/>
      </> : <section className="login-card"><div className="eyebrow">CRAFTSKY MODERATION</div><h1>A little care<br/>for the community.</h1><p className="intro">Sign in with your Atmosphere account to access the moderation workspace.</p><form onSubmit={event => void signIn(event)}><label htmlFor="identifier">Handle or DID</label><input id="identifier" autoComplete="username" autoCapitalize="none" spellCheck={false} required disabled={busy !== null} value={identifier} onChange={event => setIdentifier(event.target.value)} placeholder="you.bsky.social"/><button disabled={busy !== null || !identifier.trim()}>{busy === 'login' && <span className="spinner"/>}{busy === 'login' ? 'Connecting…' : 'Continue with Atmosphere'}<span aria-hidden="true">→</span></button></form><p className="hint">Access is limited to approved CraftSky operators. You’ll sign in with your account provider.</p></section>}
      {error && <div className="error" role="alert">{error}{!operator && !busy && <button className="text-button" onClick={() => void refresh()}>Retry connection</button>}</div>}
    </main><footer>CraftSky · A community made with care</footer>
  </div>
}
createRoot(document.getElementById('root')!).render(<App />)
