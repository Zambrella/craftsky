import { useEffect, useRef } from 'react'
import type { Review } from './model'

export function Confirmation({ review, reference, revision, returnFocus, onCancel, onConfirm }: { review: Review; reference: string; revision: number; returnFocus: HTMLElement | null; onCancel: () => void; onConfirm: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null)
  const cancel = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    const element = dialog.current!
    if (typeof element.showModal === 'function') element.showModal()
    else element.setAttribute('open', '')
    cancel.current?.focus()
    return () => { element.close?.(); returnFocus?.focus() }
  }, [])
  return <dialog ref={dialog} aria-labelledby="confirm-title" onCancel={e => { e.preventDefault(); onCancel() }} onKeyDown={event => {
    if (event.key !== 'Tab') return
    const buttons = Array.from(dialog.current!.querySelectorAll<HTMLButtonElement>('button:not(:disabled)'))
    const first = buttons[0], last = buttons.at(-1)
    if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus() }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus() }
  }}>
    <div className="eyebrow">REVIEW BEFORE SAVING</div><h2 id="confirm-title">{review.title}</h2>
    <p className="did">{reference} · revision {revision}</p>
    <ul>{review.summary.map((line, i) => <li key={i} className="preserve">{line}</li>)}</ul>
    <p>Actions affect CraftSky. Records on the account owner’s PDS remain theirs.</p>
    <p>One strike per case. Three active strikes suspend the account. A severe suspension remains until explicitly restored.</p>
    <div className="button-row"><button ref={cancel} className="secondary" onClick={onCancel}>Back to review</button><button className="primary" onClick={onConfirm}>Confirm {review.title.toLowerCase()}</button></div>
  </dialog>
}
