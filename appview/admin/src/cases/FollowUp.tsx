import { useState, type FormEvent } from 'react'
import { effectLabels, type CaseDetail, type Effect, type Review } from './model'
import { EffectChoices } from './EffectChoices'

export function FollowUp({ detail, review }: { detail: CaseDetail; review: (r: Review) => void }) {
  const active = [...new Set(detail.effects.filter(e => e.active).map(e => e.type))]
  const historical = [...new Set(detail.effects.map(e => e.type))]
  const reversible = active.filter(e => e !== 'severeSuspension')
  const reapplicable = historical.filter(e => e !== 'severeSuspension' && !active.includes(e))
  const severe = detail.effects.find(e => e.active && e.type === 'severeSuspension')
  const [correspondence, setCorrespondence] = useState('')
  const [outcome, setOutcome] = useState('upheld')
  const [appealEffects, setAppealEffects] = useState<Effect[]>([])
  const [appealRationale, setAppealRationale] = useState('')
  const [negate, setNegate] = useState<Effect[]>([])
  const [apply, setApply] = useState<Effect[]>([])
  const [effectRationale, setEffectRationale] = useState('')
  const [restorationRationale, setRestorationRationale] = useState('')
  function confirmAppeal(event: FormEvent) {
    event.preventDefault()
    review({ title: 'Appeal confirmation', endpoint: 'appeal-confirmations', body: correspondence.trim() ? { correspondenceReference: correspondence.trim() } : {}, summary: ['Record a confirmed appeal from the account owner. Consequences remain in place during review.', `Correspondence reference: ${correspondence.trim() || 'Not linked'}`] })
  }
  function resolveAppeal(event: FormEvent) {
    event.preventDefault()
    const reversedEffects = outcome === 'changed' ? appealEffects : []
    review({ title: 'Appeal resolution', endpoint: 'appeal-resolutions', body: { outcome, reversedEffects, rationale: appealRationale.trim() }, summary: [`Appeal ${outcome}.`, ...reversedEffects.map(e => `Reverse: ${effectLabels[e]}`), `Rationale: ${appealRationale}`] })
  }
  function changeEffects(event: FormEvent) {
    event.preventDefault()
    review({ title: 'Effect change', endpoint: 'effect-changes', body: { negateEffects: negate, applyEffects: apply, rationale: effectRationale.trim() }, summary: [...negate.map(e => `Reverse: ${effectLabels[e]}`), ...apply.map(e => `Reapply: ${effectLabels[e]}`), `Rationale: ${effectRationale}`] })
  }
  return <>
    {!detail.appealStatus && detail.decisions.some(d => d.disposition === 'violation') && !!historical.length && <form className="case-form" onSubmit={confirmAppeal}><h3>Confirm an appeal</h3><p>Confirm only after receiving an appeal from the account owner.</p><label>Correspondence reference (optional)<input value={correspondence} pattern="[0-9a-fA-F-]{36}" onChange={e => setCorrespondence(e.target.value)}/></label><p className="hint">Use the reference of correspondence already attached to this case.</p><button className="primary">Review appeal confirmation</button></form>}
    {detail.appealStatus === 'pending' && <form className="case-form" onSubmit={resolveAppeal}><h3>Resolve the appeal</h3><label>Appeal outcome<select value={outcome} onChange={e => setOutcome(e.target.value)}><option value="upheld">Uphold the decision</option><option value="changed">Change the decision</option></select></label>{outcome === 'changed' && <EffectChoices legend="Effects to reverse on appeal" choices={reversible} value={appealEffects} onChange={setAppealEffects}/>}<label>Appeal rationale<textarea required maxLength={10000} value={appealRationale} onChange={e => setAppealRationale(e.target.value)}/></label><p className="hint">Review rationale becomes part of case history. Severe suspension restoration is a separate action.</p><button className="primary" disabled={!appealRationale.trim() || (outcome === 'changed' && !appealEffects.length)}>Review appeal resolution</button></form>}
    {!!(reversible.length || reapplicable.length) && <form className="case-form" onSubmit={changeEffects}><h3>Change consequences</h3><p>Reverse active effects or reapply a consequence previously issued in this case.</p>{!!reversible.length && <EffectChoices legend="Reverse active consequences" prefix="Reverse " choices={reversible} value={negate} onChange={setNegate}/>} {!!reapplicable.length && <EffectChoices legend="Reapply prior consequences" prefix="Reapply " choices={reapplicable} value={apply} onChange={setApply}/>}<label>Effect change rationale<textarea required maxLength={10000} value={effectRationale} onChange={e => setEffectRationale(e.target.value)}/></label><button className="primary" disabled={!effectRationale.trim() || !(negate.length + apply.length)}>Review effect change</button></form>}
    {severe && <form className="case-form restoration" onSubmit={event => { event.preventDefault(); review({ title: 'Severe suspension restoration', endpoint: 'restorations', body: { effectId: severe.effectId, rationale: restorationRationale.trim() }, summary: ['Restore the severe suspension issued by this case.', `Effect reference: ${severe.effectId}`, `Rationale: ${restorationRationale}`, 'Other severe suspensions or strike threshold restrictions may still suspend the account.'] }) }}><h3>Restore a severe suspension</h3><p>This lifts this case’s severe suspension only. It does not overturn strikes or change content visibility.</p><label>Restoration rationale<textarea required maxLength={10000} value={restorationRationale} onChange={e => setRestorationRationale(e.target.value)}/></label><button className="primary" disabled={!restorationRationale.trim()}>Review severe suspension restoration</button></form>}
    {!historical.length && <p>No consequences were issued in this case.</p>}
  </>
}
