import { useEffect, useRef, useState, type FormEvent } from 'react'
import { ApiError, browserToken, request } from '../api'
import { base, effectLabels, label, reasons, type CaseDetail, type Effect, type Review } from './model'
import { Confirmation } from './Confirmation'
import { FollowUp } from './FollowUp'
import { EffectChoices } from './EffectChoices'

type Command = { path: string; body: string; key: string; title: string }
export function CaseActions({ detail, disabled, refresh, onSessionExpired, onLocked }: { detail: CaseDetail; disabled: boolean; refresh: () => Promise<void>; onSessionExpired: () => void; onLocked: (locked: boolean) => void }) {
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
      await request(next.path, { method: 'POST', body: next.body, signal: AbortSignal.timeout(30_000), headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': await browserToken(), 'Idempotency-Key': next.key } })
      setCommand(null); setNotice('Command committed. The case is being refreshed.'); setSettling(true)
      // A read failure cannot change the known outcome of the committed write.
      try { await refresh(); setNotice('Command committed. Case data refreshed.') }
      catch { setNotice('Command committed. Refresh the case before taking another action.') }
    } catch (error) {
      if (error instanceof ApiError && error.status >= 400 && error.status < 500 && ![408, 429].includes(error.status)) {
        setCommand(null)
        if (error.status === 401 || error.status === 403) { onSessionExpired(); return }
        const support = error.requestId ? ` Request reference: ${error.requestId}.` : ''
        if (error.status === 409) {
          setNotice('Case changed or command conflicted. Review the refreshed case and confirm a new command.' + support)
          setSettling(true); await refresh().catch(() => {})
        } else setNotice('The command was not accepted. Check the case state, policy reason and selected consequences before reviewing again.' + support)
      } else setNotice('The outcome is uncertain. Keep this page open and retry the exact command to safely recover its result.')
    } finally { inFlight.current = false; setRunning(false); setSettling(false) }
  }
  function confirm() {
    if (!review || inFlight.current) return
    const next = { path: `${base}/${detail.caseReference}/${review.endpoint}`, body: JSON.stringify({ ...review.body, expectedRevision: detail.revision }), key: crypto.randomUUID(), title: review.title }
    setReview(null); setCommand(next); void execute(next)
  }
  const unavailable = disabled || locked
  return <section className="card action-card" aria-labelledby="actions-title">
    <h2 id="actions-title">{detail.state === 'open' ? 'Record a decision' : 'Review and follow up'}</h2>
    {notice && <p className="notice preserve" role="status">{notice}</p>}
    {command && <div className="notice"><button className="primary" disabled={running || settling} onClick={() => void execute(command)}>{running && <span className="spinner"/>}{running ? 'Submitting…' : 'Retry exact command'}</button><p className="hint">Other actions and case navigation are paused until this result is known.</p></div>}
    <fieldset disabled={unavailable} className="action-fields">
      {detail.state === 'open' ? <DecisionForm review={openReview}/> : <FollowUp key={detail.revision} detail={detail} review={openReview}/>}
    </fieldset>
    {review && <Confirmation review={review} reference={detail.caseReference} revision={detail.revision} returnFocus={returnFocus.current} onCancel={() => setReview(null)} onConfirm={confirm}/>}
  </section>
}

function DecisionForm({ review }: { review: (r: Review) => void }) {
  const [disposition, setDisposition] = useState('noAction')
  const [reason, setReason] = useState('spam')
  const [evidenceNotes, setEvidence] = useState('')
  const [userSafeDetail, setExplanation] = useState('')
  const [severityRationale, setSeverity] = useState('')
  const [legalClassification, setLegal] = useState('unassessed')
  const [consequences, setEffects] = useState<Effect[]>([])
  const violation = disposition === 'violation'
  function submit(event: FormEvent) {
    event.preventDefault()
    const body = violation ? { disposition, reason, evidenceNotes: evidenceNotes.trim(), userSafeDetail: userSafeDetail.trim(), legalClassification, consequences, ...(consequences.includes('severeSuspension') ? { severityRationale: severityRationale.trim() } : {}) } : { disposition, ...(evidenceNotes.trim() ? { evidenceNotes: evidenceNotes.trim() } : {}) }
    review({ title: 'Decision', endpoint: 'decisions', body, summary: [violation ? `Violation: ${label(reason)}` : 'No action. Resolve without consequences.', ...consequences.filter(() => violation).map(e => effectLabels[e]), `Internal notes: ${evidenceNotes || 'None'}`, ...(violation ? [`Explanation for owner: ${userSafeDetail || 'No additional explanation'}`, `Legal classification: ${label(legalClassification)}`, ...(consequences.includes('severeSuspension') ? [`Severe suspension rationale: ${severityRationale}`] : [])] : [])] })
  }
  return <form onSubmit={submit} className="case-form"><label>Decision<select value={disposition} onChange={e => setDisposition(e.target.value)}><option value="noAction">No action</option><option value="violation">Policy violation</option></select></label>
    {violation && <label>Policy reason<select value={reason} onChange={e => setReason(e.target.value)}>{reasons.map(r => <option value={r} key={r}>{label(r)}</option>)}</select></label>}
    <label>Internal evidence notes<textarea value={evidenceNotes} required={violation} maxLength={10000} onChange={e => setEvidence(e.target.value)}/></label><p className="hint">For operators only. Keep evidence factual and relevant.</p>
    {violation && <><label>Explanation for the account owner<textarea value={userSafeDetail} maxLength={10000} onChange={e => setExplanation(e.target.value)}/></label><p className="hint">Visible to the account owner. Do not include reporter identities, private evidence or internal notes.</p>
      <EffectChoices legend="Consequences" choices={Object.keys(effectLabels) as Effect[]} value={consequences} onChange={setEffects}/>
      {consequences.includes('severeSuspension') && <label>Severe suspension rationale<textarea required value={severityRationale} maxLength={10000} onChange={e => setSeverity(e.target.value)}/></label>}
      <label>Legal classification<select value={legalClassification} onChange={e => setLegal(e.target.value)}>{['unassessed', 'childSexualExploitation', 'intimateImageAbuse', 'credibleThreat', 'otherIllegalContent'].map(v => <option value={v} key={v}>{label(v)}</option>)}</select></label>
      <p className="hint">Choose at most one content visibility consequence. Strikes and severe suspensions require a qualifying policy reason. The server validates each decision.</p>
    </>}
    <button className="primary" disabled={violation && (!evidenceNotes.trim() || !consequences.length)}>Review decision</button>
  </form>
}
