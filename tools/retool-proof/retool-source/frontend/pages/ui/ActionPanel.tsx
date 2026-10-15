import type { ActionUiState } from '../../hooks/useModerationActions'
import type { CaseDetail } from '../../utils/craftskyTypes'
import { ConfirmDialog } from './ConfirmDialog'
import { AlertTriangle, RefreshCw, Undo2, Zap } from 'lucide-react'

type Kind = 'apply' | 'reverse'

type Props = {
  detail: CaseDetail | null
  applyState: ActionUiState
  reverseState: ActionUiState
  anyRunning: boolean
  onBegin: (kind: Kind) => void
  onCancel: (kind: Kind) => void
  onConfirm: (kind: Kind) => void
  onDismiss: (kind: Kind) => void
  onConflictRefresh: (kind: Kind) => void
}

/** Compact panel for the two synthetic actions, with confirm, replay-retry and conflict states. */
export function ActionPanel(props: Props) {
  const { detail, applyState: a, reverseState: r, anyRunning } = props
  const ready = detail !== null
  const button = (kind: Kind, label: string, icon: React.ReactNode, state: ActionUiState) => {
    const locked = !ready || anyRunning || state.phase === 'conflict'
    const retrying = state.phase === 'error' && state.command !== null
    return (
      <button
        type="button"
        onClick={() => props.onBegin(kind)}
        disabled={locked}
        title={
          state.phase === 'conflict'
            ? 'Refresh the case and confirm again'
            : !ready
              ? 'Load a case first'
              : undefined
        }
        className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:cursor-not-allowed disabled:opacity-50"
      >
        {state.phase === 'running' ? (
          <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true" />
        ) : (
          icon
        )}
        {state.phase === 'running' ? 'Working…' : retrying ? 'Retry (same key + revision)' : label}
      </button>
    )
  }
  return (
    <section aria-labelledby="actions-heading" className="rounded-lg border border-border bg-card p-5 shadow-retool-sm">
      <h2 id="actions-heading" className="text-base font-semibold">Synthetic actions</h2>
      <p className="mt-1 text-sm text-muted-foreground">
        Commands target the isolated synthetic fixture. A retry always resends the stored command unchanged — same idempotency key, same expectedRevision.
      </p>
      <div className="mt-4 flex flex-wrap gap-3">
        {button('apply', 'Apply synthetic hide and strike', <Zap className="h-4 w-4" aria-hidden="true" />, a)}
        {button('reverse', 'Reverse synthetic effects', <Undo2 className="h-4 w-4" aria-hidden="true" />, r)}
      </div>
      {detail !== null ? (
        <p className="mt-3 text-xs text-muted-foreground">
          Currently displayed revision: <span className="font-mono font-semibold">{detail.revision}</span> · case state: {detail.state}
        </p>
      ) : null}
      <Outcome kind="apply" state={a} {...props} />
      <Outcome kind="reverse" state={r} {...props} />
      <ConfirmDialog open={a.phase === 'confirm'} kind="apply" detail={detail} retryCommand={a.command} onConfirm={() => props.onConfirm('apply')} onCancel={() => props.onCancel('apply')} />
      <ConfirmDialog open={r.phase === 'confirm'} kind="reverse" detail={detail} retryCommand={r.command} onConfirm={() => props.onConfirm('reverse')} onCancel={() => props.onCancel('reverse')} />
    </section>
  )
}

function Outcome({
  kind,
  state,
  onDismiss,
  onConflictRefresh,
}: Props & { kind: Kind; state: ActionUiState }) {
  const label = kind === 'apply' ? 'Apply' : 'Reversal'
  if (state.phase === 'conflict' && state.failure) {
    const f = state.failure
    return (
      <div role="alert" className="mt-4 rounded-md border border-warning/50 bg-warning/10 p-3">
        <p className="flex items-center gap-2 text-sm font-medium">
          <AlertTriangle className="h-4 w-4" aria-hidden="true" />
          Revision conflict on {label.toLowerCase()} — displayed revision no longer matches.
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {f.message}
          {f.code ? ` · code: ${f.code}` : ''}{f.requestId ? ` · requestId: ${f.requestId}` : ''}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">The stored command was retired, not retried; refresh and review, then confirm anew.</p>
        <div className="mt-2 flex gap-2">
          <button type="button" onClick={() => onConflictRefresh(kind)} className="inline-flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium hover:bg-accent">
            <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" /> Refresh case
          </button>
          <button type="button" onClick={() => onDismiss(kind)} className="rounded-md border border-border px-3 py-1.5 text-xs font-medium hover:bg-accent">Dismiss</button>
        </div>
      </div>
    )
  }
  if (state.phase === 'error' && state.failure) {
    const f = state.failure
    return (
      <div role="alert" className="mt-4 rounded-md border border-destructive/40 bg-destructive/10 p-3">
        <p className="text-sm font-medium text-destructive">{label} failed: {f.message}</p>
        <p className="mt-1 text-xs text-muted-foreground">
          {f.status !== undefined ? `status: ${f.status} · ` : ''}{f.code ? `code: ${f.code} · ` : ''}{f.requestId ? `requestId: ${f.requestId} · ` : ''}
          stored command kept unchanged: retry resends key <span className="font-mono">{state.command?.idempotencyKey.slice(0, 8) ?? '—'}</span> at expectedRevision <span className="font-mono">{state.command?.expectedRevision ?? '—'}</span>
        </p>
        <p className="mt-1 text-xs text-muted-foreground">A thrown timeout can occur after the backend committed; the unchanged replay resolves that uncertainty.</p>
        <button type="button" onClick={() => onDismiss(kind)} className="mt-2 rounded-md border border-border px-3 py-1.5 text-xs font-medium hover:bg-accent">Dismiss</button>
      </div>
    )
  }
  if (state.phase === 'success') {
    return (
      <div role="status" className="mt-4 rounded-md border border-border bg-accent/50 p-3 text-sm">
        <p className="font-medium text-accent-foreground">
          {label === 'Apply' ? 'Synthetic decision applied' : 'Synthetic effects reversed'}
          {state.replayed === true ? ' (idempotent replay)' : ''} — revision now <span className="font-mono">{state.successRevision ?? '—'}</span>.
        </p>
        <p className="mt-1 text-xs text-muted-foreground">A fresh command (new key) is generated for the next confirmation.</p>
        <button type="button" onClick={() => onDismiss(kind)} className="mt-2 rounded-md border border-border px-3 py-1.5 text-xs font-medium hover:bg-accent">Dismiss</button>
      </div>
    )
  }
  return null
}
