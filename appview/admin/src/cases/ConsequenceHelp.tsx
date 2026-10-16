import { useId, useState } from 'react'
import { effectLabels, type Effect } from './model'

const descriptions: Record<Effect, string> = {
  formalWarning: 'Record an official warning to the account owner in their moderation history. Public visibility is unchanged; no strike is added.',
  visibilityWarn: 'Show a warning banner to viewers of the affected post or profile. An account warning also appears on its posts. Content stays accessible; no strike is added.',
  visibilityHide: 'Hide the affected post, event or account from other people in CraftSky. Records remain on the owner’s PDS; strikes and suspension are separate consequences.',
  visibilityTakedown: 'Record a takedown and make the affected post, event or account unavailable to other people in CraftSky. This does not delete records from the owner’s PDS.',
  strike: 'Count one strike for this case toward account suspension. Three active strikes suspend the account. Each case contributes at most one strike, due to expire after one year.',
  severeSuspension: 'Suspend the account for a qualifying severe violation, regardless of strike count. This suspension remains until explicitly restored. Other suspension reasons can still apply afterward.',
}

export function ConsequenceHelp({ effect }: { effect: Effect }) {
  const id = useId()
  const [hovered, setHovered] = useState(false)
  const [focused, setFocused] = useState(false)
  const [dismissed, setDismissed] = useState(false)
  const open = (hovered || focused) && !dismissed
  return <span className="consequence-help" onMouseEnter={() => { setHovered(true); setDismissed(false) }} onMouseLeave={() => setHovered(false)}>
    <button type="button" className="consequence-help-trigger" aria-label={`${effectLabels[effect]}: purpose and effect`} aria-describedby={id}
      onFocus={() => { setFocused(true); setDismissed(false) }} onBlur={() => setFocused(false)} onClick={() => setDismissed(false)}
      onKeyDown={event => { if (event.key === 'Escape' && open) { event.preventDefault(); event.stopPropagation(); setDismissed(true) } }}><span aria-hidden="true">?</span></button>
    <span id={id} role="tooltip" className="consequence-tooltip" hidden={!open}>{descriptions[effect]}</span>
  </span>
}
