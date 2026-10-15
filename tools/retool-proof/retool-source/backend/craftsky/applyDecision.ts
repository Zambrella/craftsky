import { requireDraftOwner } from './_access'
import { extractApiError, isRevisionConflict, isUuid } from './_errors'

type Params = {
  /** Caller-supplied UUID. Preserved unchanged across retries of the same logical action. */
  idempotencyKey: string
  /** The revision displayed to the operator before the action. Never refreshed secretly. */
  expectedRevision: number
}

export type ApplyDecisionResult =
  | { ok: true; caseReference: string; revision: number; replayed: boolean }
  | {
      ok: false
      conflict: boolean
      status?: number
      code?: string
      requestId?: string
      message: string
    }

/**
 * Apply synthetic decision on the synthetic fixture via the PROOF-ONLY typed
 * adapter craftskySyntheticModerationProof.proof.postProofDecision, which
 * forwards the JSON idempotencyKey as an Idempotency-Key header into the real
 * moderation handler, scoped to exactly the synthetic fixture.
 *
 * Command contract (fixed):
 *   disposition: 'violation', reason: 'spam',
 *   evidenceNotes: 'Synthetic fixture review',
 *   consequences: ['visibilityHide', 'strike']
 *
 * Uses the caller-supplied expectedRevision verbatim; no newer revision is
 * fetched before the command.
 */
export default async function (req: { params: Params; user: User }): Promise<ApplyDecisionResult> {
  requireDraftOwner(req.user)
  if (!isUuid(req.params.idempotencyKey)) {
    throw new Error('idempotencyKey must be a caller-supplied UUID')
  }
  if (!Number.isInteger(req.params.expectedRevision) || req.params.expectedRevision < 0) {
    throw new Error('expectedRevision must be a non-negative integer')
  }
  try {
    const result = await craftskySyntheticModerationProof.proof.postProofDecision({
      expectedRevision: req.params.expectedRevision,
      idempotencyKey: req.params.idempotencyKey,
    })
    return {
      ok: true,
      caseReference: result.data.caseReference,
      revision: result.data.revision,
      replayed: result.data.replayed,
    }
  } catch (error) {
    const normalized = extractApiError(error)
    return {
      ok: false,
      conflict: isRevisionConflict(normalized),
      ...normalized,
    }
  }
}
