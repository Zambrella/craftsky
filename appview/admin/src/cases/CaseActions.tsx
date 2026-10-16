import { useState, type FormEvent } from 'react'
import { base, effectLabels, label, reasons, type CaseDetail, type Effect, type Review } from './model'
import { FollowUp } from './FollowUp'
import { EffectChoices } from './EffectChoices'
import { CommandPanel } from '../workspaces/CommandPanel'

export function CaseActions({detail, ...props}: {detail:CaseDetail;disabled:boolean;refresh:()=>Promise<void>;onSessionExpired:()=>void;onLocked:(v:boolean)=>void}) {
  return <CommandPanel {...props} basePath={`${base}/${detail.caseReference}`} reference={detail.caseReference} revision={detail.revision} title={detail.state==='open'?'Record a decision':'Review and follow up'}>{(review,epoch)=>detail.state==='open'?<DecisionForm key={epoch} review={review}/>:<FollowUp key={`${detail.revision}:${epoch}`} detail={detail} review={review}/>}</CommandPanel>
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
