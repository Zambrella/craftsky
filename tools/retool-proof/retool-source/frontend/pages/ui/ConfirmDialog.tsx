import type { StoredCommand } from '../../hooks/useModerationActions'
import type { CaseDetail } from '../../utils/craftskyTypes'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../../lib/shadcn/alert-dialog'

const EFFECT_LINES: Record<'apply' | 'reverse', string[]> = {
  apply: [
    'POST decision: disposition "violation", reason "spam"',
    'Consequences: visibilityHide, strike',
    'evidenceNotes: "Synthetic fixture review"',
  ],
  reverse: [
    'POST effect-changes: negateEffects visibilityHide, strike',
    'rationale: "Synthetic reversal"',
  ],
}

/**
 * Confirms the exact effects before dispatch. On a retry the STORED
 * expectedRevision is shown — never the newer displayed revision — because
 * the stored command is resent unchanged under its original key.
 */
export function ConfirmDialog({
  open,
  kind,
  detail,
  retryCommand,
  onConfirm,
  onCancel,
}: {
  open: boolean
  kind: 'apply' | 'reverse'
  detail: CaseDetail | null
  retryCommand: StoredCommand | null
  onConfirm: () => void
  onCancel: () => void
}) {
  const isRetry = retryCommand !== null
  const shownRevision = retryCommand?.expectedRevision ?? detail?.revision ?? null
  return (
    <AlertDialog open={open}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {isRetry
              ? kind === 'apply'
                ? 'Retry apply with the unchanged stored command?'
                : 'Retry reversal with the unchanged stored command?'
              : kind === 'apply'
                ? 'Apply synthetic hide and strike?'
                : 'Reverse synthetic effects?'}
          </AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div>
              <p>One command to the isolated synthetic fixture, with exactly:</p>
              <ul className="mt-2 list-disc space-y-1 pl-5">
                {EFFECT_LINES[kind].map((line) => (
                  <li key={line} className="font-mono text-xs">{line}</li>
                ))}
              </ul>
              <p className="mt-2">
                {isRetry ? (
                  <>
                    Stored command is resent unchanged: expectedRevision{' '}
                    <span className="font-mono font-semibold">{shownRevision}</span> (the revision
                    stored at first confirmation, not the currently displayed{' '}
                    {detail?.revision ?? '—'}) · same idempotency key{' '}
                    <span className="font-mono">{retryCommand.idempotencyKey.slice(0, 8)}…</span>.
                  </>
                ) : (
                  <>
                    expectedRevision:{' '}
                    <span className="font-mono font-semibold">{shownRevision ?? '—'}</span> (the
                    revision currently displayed) · a new caller idempotency key is generated on
                    confirm and kept unchanged for retries.
                  </>
                )}
              </p>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onCancel}>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>{isRetry ? 'Retry' : 'Confirm'}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
