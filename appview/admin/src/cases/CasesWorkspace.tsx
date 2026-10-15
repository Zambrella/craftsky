import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError, request, type Operator } from '../api'
import { base, date, effectLabels, label, selectedReference, type CaseItem, type CaseDetail, type Page, type Preview, type Provenance } from './model'
import { CaseActions } from './CaseActions'

export function CasesWorkspace({ operator, onSessionExpired, onLocked }: { operator: Operator; onSessionExpired: () => void; onLocked?: (locked: boolean) => void }) {
  const [locked, setLocked] = useState(false)
  const lock = useRef(false)
  const changeLock = useCallback((value: boolean) => { lock.current = value; setLocked(value); onLocked?.(value) }, [onLocked])
  const [state, setState] = useState('open')
  const [subject, setSubject] = useState('')
  const [appeal, setAppeal] = useState('')
  const [page, setPage] = useState<Page<CaseItem>>({ items: [] })
  const [queueLoading, setQueueLoading] = useState(false)
  const [queueError, setQueueError] = useState('')
  const [reference, setReference] = useState(selectedReference)
  const [detail, setDetail] = useState<CaseDetail | null>(null)
  const [preview, setPreview] = useState<Preview | null>(null)
  const [provenance, setProvenance] = useState<Page<Provenance>>({ items: [] })
  const [provenanceLoading, setProvenanceLoading] = useState(false)
  const [detailError, setDetailError] = useState('')
  const [detailLoading, setDetailLoading] = useState(false)
  const queueGeneration = useRef(0)
  const detailGeneration = useRef(0)
  const canRead = !!operator.permissions['moderation.read']
  const denied = useCallback((error: unknown) => {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { onSessionExpired(); return true }
    return false
  }, [onSessionExpired])
  async function loadQueue(cursor?: string) {
    const generation = ++queueGeneration.current
    setQueueLoading(true); setQueueError('')
    const query = new URLSearchParams({ limit: '50' })
    if (state) query.set('state', state)
    if (subject) query.set('subjectType', subject)
    if (appeal) query.set('appealStatus', appeal)
    if (cursor) query.set('cursor', cursor)
    try {
      const next = await request<Page<CaseItem>>(`${base}?${query}`)
      if (generation === queueGeneration.current) setPage(prior => ({ ...next, items: cursor ? [...prior.items, ...next.items] : next.items }))
    } catch (error) { if (generation === queueGeneration.current && !denied(error)) setQueueError('The case queue could not be loaded. Try refreshing it.') }
    finally { if (generation === queueGeneration.current) setQueueLoading(false) }
  }
  const loadDetail = useCallback(async () => {
    if (!reference) return
    const generation = ++detailGeneration.current
    setDetailLoading(true); setDetailError('')
    try {
      const [next, content, sources] = await Promise.all([
        request<CaseDetail>(`${base}/${reference}`), request<Preview>(`${base}/${reference}/preview`),
        request<Page<Provenance>>(`${base}/${reference}/provenance?limit=50`),
      ])
      if (generation === detailGeneration.current) { setDetail(next); setPreview(content); setProvenance(sources) }
    } catch (error) {
      if (generation === detailGeneration.current && !denied(error)) setDetailError('Case data could not be refreshed. Refresh before taking another action.')
      throw error
    } finally { if (generation === detailGeneration.current) setDetailLoading(false) }
  }, [reference, denied])
  async function olderProvenance() {
    if (!provenance.cursor || provenanceLoading) return
    const generation = detailGeneration.current
    setProvenanceLoading(true)
    try {
      const next = await request<Page<Provenance>>(`${base}/${reference}/provenance?limit=50&cursor=${encodeURIComponent(provenance.cursor)}`)
      if (generation === detailGeneration.current) setProvenance(prior => ({ ...next, items: [...prior.items, ...next.items] }))
    } catch (error) { if (generation === detailGeneration.current && !denied(error)) setDetailError('Older provenance could not be loaded. Refresh the case before continuing.') }
    finally { if (generation === detailGeneration.current) setProvenanceLoading(false) }
  }
  useEffect(() => {
    const change = () => {
      if (lock.current) { window.history.replaceState(null, '', '#/cases/' + reference); return }
      setReference(selectedReference())
    }
    window.addEventListener('hashchange', change)
    return () => window.removeEventListener('hashchange', change)
  }, [reference])
  useEffect(() => { if (canRead) { setPage({ items: [] }); void loadQueue() } return () => { queueGeneration.current++ } }, [state, subject, appeal, canRead]) // filters own the queue
  useEffect(() => {
    setDetail(null); setPreview(null); setProvenance({ items: [] }); setProvenanceLoading(false); setDetailError('')
    if (canRead && reference) void loadDetail().catch(() => {})
    return () => { detailGeneration.current++ }
  }, [reference, canRead, loadDetail])
  if (!canRead) return <section className="card"><h2>Assigned safety work</h2><p>Your access is limited to assigned incidents. The specialist workspace will be added in the next stage.</p></section>
  return <div className="cases-workspace">
    <aside className="queue card" aria-label="Case queue">
      <div className="card-title"><h2>Cases</h2><button className="secondary" disabled={queueLoading || locked} onClick={() => void loadQueue()}>{queueLoading && <span className="spinner"/>}Refresh queue</button></div>
      <fieldset disabled={locked} className="filters action-fields"><label>Case status<select value={state} onChange={e => setState(e.target.value)}><option value="open">Open</option><option value="resolved">Resolved</option><option value="">All statuses</option></select></label><label>Subject type<select value={subject} onChange={e => setSubject(e.target.value)}><option value="">All subjects</option>{['post', 'account', 'event'].map(s => <option key={s} value={s}>{label(s)}</option>)}</select></label><label>Appeal status<select value={appeal} onChange={e => setAppeal(e.target.value)}><option value="">All appeals</option>{['none', 'pending', 'upheld', 'changed'].map(s => <option key={s} value={s}>{label(s)}</option>)}</select></label></fieldset>
      {queueError && <p role="alert">{queueError}</p>}
      {queueLoading && <p role="status">Loading cases…</p>}
      {!queueLoading && !queueError && !page.items.length && <p className="empty">No cases match these filters.</p>}
      <ol className="case-list">{page.items.map(item => <li key={item.caseReference}><a href={`#/cases/${item.caseReference}`} aria-label={`Review ${item.caseReference}`} aria-current={reference === item.caseReference ? 'page' : undefined} aria-disabled={locked} onClick={e => { if (lock.current) e.preventDefault(); else setReference(item.caseReference) }}><div><span className="badge">{label(item.subjectType)}</span><span>{item.reportCount} {item.reportCount === 1 ? 'report' : 'reports'}</span></div><strong>{item.caseReference}</strong><span className="did">{item.subjectDid}</span><small>{label(item.origin)} · {date(item.createdAt)}</small></a></li>)}</ol>
      {page.cursor && <button className="secondary" disabled={queueLoading || locked} onClick={() => void loadQueue(page.cursor)}>Load more cases</button>}
    </aside>
    <section className="case-pane" aria-label="Case review">
      {!reference ? <div className="card empty"><h2>Make room for a fair review.</h2><p>Choose a case to review its reports, content and account standing.</p><p>Reports are allegations. Review the evidence before deciding.</p></div> : <>
        <div className="case-toolbar"><span className="hint">Reports are allegations. Review evidence before deciding.</span><button className="secondary" disabled={detailLoading || locked} onClick={() => void loadDetail().catch(() => {})}>{detailLoading && <span className="spinner"/>}Refresh case</button></div>
        {detailLoading && <p role="status">Loading case…</p>}
        {detailError && <div className="notice" role="alert">{detailError}</div>}
        {detail && <><div className="case-record"><CaseRead detail={detail} preview={preview} provenance={provenance.items}/>{provenance.cursor && <button className="secondary" disabled={provenanceLoading || detailLoading || locked} onClick={() => void olderProvenance()}>{provenanceLoading && <span className="spinner"/>}Load older provenance</button>}</div>{operator.permissions['moderation.manage'] ? <CaseActions key={reference} detail={detail} disabled={detailLoading || provenanceLoading || !!detailError} refresh={async () => { await loadDetail(); void loadQueue() }} onSessionExpired={onSessionExpired} onLocked={changeLock}/> : <p className="notice">Your access permits review only.</p>}</>}
      </>}
    </section>
  </div>
}

function CaseRead({ detail: d, preview, provenance }: { detail: CaseDetail; preview: Preview | null; provenance: Provenance[] }) {
  return <>
    <section className="card"><div className="card-title"><h2 className="case-reference">{d.caseReference}</h2><span className="badge">{label(d.state)} · revision {d.revision}</span></div><p className="did">{d.subjectUri || d.subjectDid}</p><p className="hint">{label(d.origin)} · Opened {date(d.createdAt)}{d.appealStatus && ` · Appeal ${d.appealStatus}`}</p></section>
    <section className="card"><h2>Content preview</h2>{preview?.state === 'available' ? <><h3>{preview.title}</h3><p className="preserve">{preview.text || 'No text accompanies this subject.'}</p><p className="hint">Text only. Media is not displayed in this workspace.</p></> : <p>{preview?.state === 'restricted' ? 'Restricted content. Use the specialist review workflow.' : 'Content is unavailable. It may have changed or may not have a safe captured version.'}</p>}</section>
    <section className="card"><h2>Reports <span className="badge">{d.reportCount}</span></h2><p className="hint">Multiple reports do not create multiple strikes.</p>{d.reports.map(r => <article className="entry" key={r.id}><strong>{label(r.reason)}</strong><p className="preserve">{r.details || 'No additional details.'}</p><small className="did">{r.reporterDid} · {date(r.createdAt)}</small></article>)}{!d.reports.length && <p>No user reports are attached.</p>}</section>
    <section className="card"><h2>Account standing</h2><dl><dt>Account</dt><dd className="did">{d.ownerDid}</dd><dt>Active strikes</dt><dd>{d.standing.activeStrikeCount} of {d.standing.strikeThreshold}</dd><dt>Account access</dt><dd>{d.standing.suspended ? 'Suspended' : 'Active'}</dd><dt>Suspension bases</dt><dd>{[d.standing.thresholdSuspended && 'Strike threshold', d.standing.severeSuspended && 'Severe suspension'].filter(Boolean).join(', ') || 'None'}</dd>{d.strikeContext && <><dt>This case’s strike</dt><dd>{d.strikeContext.active ? `Active until ${date(d.strikeContext.dueAt)}` : `Inactive${d.strikeContext.overturnedAt ? ' · Overturned ' + date(d.strikeContext.overturnedAt) : ''}`}</dd></>}</dl><p className="hint">Standing includes all cases. Lifting one effect may leave other restrictions in place.</p></section>
    {d.appealStatus && <section className="card"><h2>Appeal review</h2><p>Status: {label(d.appealStatus)}</p>{d.appealStatus !== 'pending' && <><h3>Internal review rationale</h3><p className="preserve">{d.appealRationale || 'No rationale was retained for this earlier appeal.'}</p></>}</section>}
    {!!d.decisions.length && <section className="card"><h2>Decision history</h2>{d.decisions.map((decision, i) => <article className="entry" key={i}><h3>{label(decision.disposition)}{decision.reason && ` · ${label(decision.reason)}`}</h3><small>{date(decision.createdAt)}</small>{decision.legalClassification && <p>Legal classification: {label(decision.legalClassification)}</p>}<h4>Internal evidence notes</h4><p className="preserve">{decision.evidenceNotes || 'None recorded.'}</p><h4>Explanation for the account owner</h4><p className="preserve">{decision.userSafeDetail || 'None recorded.'}</p>{decision.severityRationale && <><h4>Severe suspension rationale</h4><p className="preserve">{decision.severityRationale}</p></>}</article>)}</section>}
    {!!d.effects.length && <section className="card"><h2>Consequences</h2><h3>Currently active in this case</h3><ul>{[...new Set(d.effects.filter(e => e.active).map(e => e.type))].map(type => <li key={type}>{effectLabels[type]}</li>)}</ul>{!d.effects.some(e => e.active) && <p>No active consequences.</p>}<h3>Effect history</h3>{d.effects.map((effect, i) => <article className="entry" key={i}><strong>{label(effect.action)} · {effectLabels[effect.type]}</strong><p className="preserve">{effect.rationale}</p><small>{date(effect.createdAt)}</small></article>)}</section>}
    <section className="card"><h2>Case timeline</h2>{!d.events.length && <p>No decisions have been recorded.</p>}{d.events.map((e, i) => <article className="entry" key={i}><strong>{label(e.type)}</strong><p className="did">{e.actorId}</p><small>{e.sourceSystem} · revision {e.revision} · {date(e.createdAt)}</small></article>)}</section>
    {!!provenance.length && <section className="card"><h2>Case provenance</h2>{provenance.map(p => <article className="entry" key={p.id}><strong>{p.reason}</strong><p className="did">{p.actorId}</p><small>{p.sourceSystem} · {date(p.createdAt)}</small></article>)}</section>}
  </>
}
