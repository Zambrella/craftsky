import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { ApiError, browserToken, request } from '../api'
import type { Review } from '../cases/model'
import { Confirmation } from '../cases/Confirmation'

type CommandResult = {caseReference?: string; id?: string; operatorId?: string; correspondenceId?: string}
type Command = { path: string; body: string; key: string; title: string }
export function CommandPanel({ basePath, reference, revision, title, resourceLabel = 'Case', disabled, refresh, onSessionExpired, onLocked, children }: {
  basePath: string; reference: string; revision?: number; title: string; resourceLabel?: string;
  disabled: boolean; refresh: (result?: CommandResult) => Promise<void>; onSessionExpired: () => void; onLocked: (locked: boolean) => void;
  children: (review: (r: Review) => void, epoch: number) => ReactNode;
}) {
  const titleId = useId()
  const [epoch, setEpoch] = useState(0)
  const [resultReference, setResultReference] = useState('')
  const [resultLink, setResultLink] = useState('')
  const [review, setReview] = useState<Review | null>(null)
  const [command, setCommand] = useState<Command | null>(null)
  const [running, setRunning] = useState(false)
  const [settling, setSettling] = useState(false)
  const [notice, setNotice] = useState('')
  const inFlight = useRef(false)
  const returnFocus = useRef<HTMLElement | null>(null)
  function openReview(next: Review) {
    // Capture before disabling the form, which blurs its focused control.
    returnFocus.current = document.activeElement as HTMLElement | null
    setReview(next)
  }
  const locked = !!review || !!command || running || settling
  useEffect(() => { onLocked(locked); return () => onLocked(false) }, [locked, onLocked])
  useEffect(() => {
    if (!locked) return
    const warn = (event: BeforeUnloadEvent) => { event.preventDefault(); event.returnValue = '' }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [locked])
  async function execute(next: Command) {
    if (inFlight.current) return
    inFlight.current = true; setRunning(true); setNotice('')
    try {
      const result = await request<CommandResult>(next.path, { method: 'POST', body: next.body, signal: AbortSignal.timeout(30_000), headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': await browserToken(), 'Idempotency-Key': next.key } })
      const appealCase = next.path.endsWith('/appeal-link') ? JSON.parse(next.body).caseId as string : undefined
      const area = /\/incidents\/[^/]+\/evidence$/.test(next.path) ? 'evidence' : /\/incidents\/[^/]+\/holds$/.test(next.path) ? 'holds' : (result?.caseReference || appealCase) ? 'cases' : next.path === '/v1/admin/operators' ? 'operators' : next.path === '/v1/admin/safety/external-intakes' ? 'intakes' : /^\/v1\/admin\/safety\/workflows\/[^/]+$/.test(next.path) ? 'workflows' : ''
      const ref = result?.caseReference || (appealCase ? 'MOD-'+appealCase : '') || result?.operatorId || result?.id
      setResultLink(area && ref ? `#/${area}/${encodeURIComponent(ref)}` : '')
      setCommand(null); setEpoch(v => v + 1); setResultReference(result?.caseReference || result?.operatorId || result?.id || result?.correspondenceId || ''); setNotice(`Command committed. The ${resourceLabel.toLowerCase()} is being refreshed.`); setSettling(true)
      // A read failure cannot change the known outcome of the committed write.
      try { await refresh(result); setNotice(`Command committed. ${resourceLabel} data refreshed.`) }
      catch { setNotice(`Command committed. Refresh the ${resourceLabel.toLowerCase()} before taking another action.`) }
    } catch (error) {
      if (error instanceof ApiError && error.status >= 400 && error.status < 500 && ![408, 429].includes(error.status)) {
        setCommand(null)
        if (error.status === 401 || error.status === 403) { onSessionExpired(); return }
        const support = error.requestId ? ` Request reference: ${error.requestId}.` : ''
        if (error.status === 409) {
          setNotice(`${resourceLabel} changed or command conflicted. Review fresh data and confirm a new command.` + support)
          setSettling(true); await refresh().catch(() => {})
        } else setNotice((error.code === 'deadline_target_not_configured' ? 'An approved response target must be configured before this workflow can start. Ask a maintainer to configure it.' : error.code === 'last_admin_required' ? 'Keep another active administrator before disabling or demoting this account.' : 'The command was not accepted. Check the selected values and current resource state before reviewing again.') + support)
      } else setNotice('The outcome is uncertain. Keep this page open and retry the exact command to safely recover its result.')
    } finally { inFlight.current = false; setRunning(false); setSettling(false) }
  }
  function confirm() {
    if (!review || inFlight.current) return
    const version = review.expectedRevision === null ? undefined : review.expectedRevision ?? revision
    const next = { path: review.path || `${basePath}/${review.endpoint}`, body: JSON.stringify({ ...review.body, ...(version === undefined ? {} : {expectedRevision: version}) }), key: crypto.randomUUID(), title: review.title }
    setReview(null); setCommand(next); void execute(next)
  }
  const unavailable = disabled || locked
  return <section className="card action-card" aria-labelledby={titleId}>
    <h2 id={titleId}>{title}</h2>
    {resultReference && <p className="hint preserve">Recorded reference: {resultReference}</p>}
    {resultLink && !locked && <p><a href={resultLink}>Open recorded item</a></p>}
    {notice && <p className="notice preserve" role="status">{notice}</p>}
    {command && <div className="notice"><button className="primary" disabled={running || settling} onClick={() => void execute(command)}>{running && <span className="spinner"/>}{running ? 'Submitting…' : 'Retry exact command'}</button><p className="hint">Other actions and workspace navigation are paused until this result is known.</p></div>}
    <fieldset disabled={unavailable} className="action-fields">
      {children(openReview, epoch)}
    </fieldset>
    {review && <Confirmation review={review} reference={reference} revision={review.expectedRevision === null ? undefined : review.expectedRevision ?? revision} returnFocus={returnFocus.current} onCancel={() => setReview(null)} onConfirm={confirm}/>}
  </section>
}

