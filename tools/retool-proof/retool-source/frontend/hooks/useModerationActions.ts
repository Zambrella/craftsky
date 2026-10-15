import { useCallback, useRef, useState } from 'react'
import type { MutationResult } from '../utils/craftskyTypes'

export type ActionKind = 'apply' | 'reverse'

export type ActionPhase = 'idle' | 'confirm' | 'running' | 'conflict' | 'error' | 'success'

export type ActionFailure = {
  message: string
  conflict: boolean
  status?: number
  code?: string
  requestId?: string
}

/**
 * The full original command, persisted from first confirmation. A retry
 * always resends this object unchanged — the idempotency key AND the
 * expectedRevision — so the body can never drift under the same key.
 */
export type StoredCommand = {
  idempotencyKey: string
  expectedRevision: number
}

export type ActionUiState = {
  phase: ActionPhase
  command: StoredCommand | null
  successRevision: number | null
  replayed: boolean | null
  failure: ActionFailure | null
}

const INITIAL: ActionUiState = {
  phase: 'idle',
  command: null,
  successRevision: null,
  replayed: null,
  failure: null,
}

function newIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  // Fallback for contexts without randomUUID (non-secure origins).
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

type Runner = (command: StoredCommand) => Promise<MutationResult>

function toFailure(result: Exclude<MutationResult, { ok: true }>): ActionFailure {
  const failure: ActionFailure = { message: result.message, conflict: result.conflict }
  if (result.status !== undefined) failure.status = result.status
  if (result.code !== undefined) failure.code = result.code
  if (result.requestId !== undefined) failure.requestId = result.requestId
  return failure
}

/**
 * State machine for the two synthetic actions. Contracts enforced here:
 * - The idempotency key is generated when an action is first confirmed and,
 *   together with the then-displayed expectedRevision, forms a stored command
 *   that is preserved unchanged across retries. Retry takes no revision
 *   parameter, so a newer displayed revision cannot enter the body.
 * - On a revision conflict the stored command is retired: the payload must
 *   change, so it is never resent under a new revision without a new
 *   confirmation.
 * - On success the stored command is retired and the command result is final.
 *   Post-success read refreshes run while `settling` is true; a rejected
 *   onSettledSuccess is caught INSIDE the success path — it can never reach
 *   the dispatch catch and relabel a committed command as failed. Owned read
 *   errors remain visible in the page's read channels.
 * - A thrown execution error (e.g. timeout) may occur AFTER the backend
 *   committed the command; the failure path preserves the unchanged command
 *   so an identical replay resolves that uncertainty.
 */
export function useModerationActions(opts: {
  runApply: Runner
  runReverse: Runner
  onSettledSuccess: () => void | Promise<void>
}) {
  const [states, setStates] = useState<Record<ActionKind, ActionUiState>>({
    apply: INITIAL,
    reverse: INITIAL,
  })
  const [settling, setSettling] = useState(false)
  const commandRef = useRef<Record<ActionKind, StoredCommand | null>>({
    apply: null,
    reverse: null,
  })
  const runningRef = useRef(false)

  const patch = useCallback(
    (kind: ActionKind, partial: Partial<ActionUiState>) => {
      setStates((prev) => ({ ...prev, [kind]: { ...prev[kind], ...partial } }))
    },
    [],
  )

  const begin = useCallback(
    (kind: ActionKind) => patch(kind, { phase: 'confirm' }),
    [patch],
  )

  const cancel = useCallback(
    (kind: ActionKind) => patch(kind, { phase: 'idle' }),
    [patch],
  )

  const dismiss = useCallback(
    (kind: ActionKind) => {
      commandRef.current[kind] = null
      patch(kind, {
        phase: 'idle',
        command: null,
        failure: null,
        successRevision: null,
        replayed: null,
      })
    },
    [patch],
  )

  /** Single dispatch path shared by first confirmation and retry. */
  const dispatch = useCallback(
    async (kind: ActionKind, command: StoredCommand) => {
      if (runningRef.current) return
      runningRef.current = true
      patch(kind, { phase: 'running', command, failure: null })
      try {
        const run: Runner = kind === 'apply' ? opts.runApply : opts.runReverse
        const result = await run(command)
        if (result.ok) {
          commandRef.current[kind] = null
          patch(kind, {
            phase: 'success',
            command: null,
            successRevision: result.revision,
            replayed: result.replayed,
            failure: null,
          })
          setSettling(true)
          try {
            await opts.onSettledSuccess()
          } catch {
            // Owned read-refresh error: it stays in the page's read channels
            // and must not relabel the committed command above as failed.
          } finally {
            setSettling(false)
          }
        } else {
          const failure = toFailure(result)
          if (result.conflict) {
            // Do not retry a changed payload: retire the command and wait for
            // a refresh plus a fresh confirmation.
            commandRef.current[kind] = null
            patch(kind, { phase: 'conflict', failure, command: null })
          } else {
            // Keep the command unchanged so an identical retry replays.
            patch(kind, { phase: 'error', failure })
          }
        }
      } catch (error) {
        // A thrown execution error (e.g. timeout) may have occurred after the
        // backend committed the command. The unchanged command is preserved so
        // an identical replay resolves that uncertainty.
        patch(kind, {
          phase: 'error',
          failure: {
            message: error instanceof Error ? error.message : String(error),
            conflict: false,
          },
        })
      } finally {
        runningRef.current = false
      }
    },
    [opts, patch],
  )

  /**
   * First confirmation: stores the displayed revision with a fresh key. If a
   * command is already stored for this action (a retry after a non-conflict
   * failure), the stored command is reused verbatim and the expectedRevision
   * argument is ignored — the body cannot drift under the same key.
   */
  const execute = useCallback(
    async (kind: ActionKind, expectedRevision: number) => {
      if (runningRef.current || settling) return
      const command = commandRef.current[kind] ?? {
        idempotencyKey: newIdempotencyKey(),
        expectedRevision,
      }
      commandRef.current[kind] = command
      await dispatch(kind, command)
    },
    [dispatch, settling],
  )

  /** Retry: resends the stored command unchanged. Takes no revision parameter. */
  const retry = useCallback(
    async (kind: ActionKind) => {
      const stored = commandRef.current[kind]
      if (stored === null || runningRef.current || settling) return
      await dispatch(kind, stored)
    },
    [dispatch, settling],
  )

  const dispatching = states.apply.phase === 'running' || states.reverse.phase === 'running'
  const anyRunning = dispatching || settling
  const roundTripComplete =
    states.apply.phase === 'success' && states.reverse.phase === 'success'

  return { states, settling, dispatching, anyRunning, roundTripComplete, begin, cancel, dismiss, execute, retry }
}
