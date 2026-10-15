import { requireDraftOwner } from './_access'
import { extractApiError } from './_errors'

export type EvidenceTransportResult =
  | {
      ok: true
      representation: string
      contentType: string
      declaredSizeBytes: number
      base64StringLength: number
      decodedByteLength: number
      sizeMatches: boolean
      directBinaryPassthrough: boolean
      streamRepresentation: string
    }
  | { ok: false; status?: number; code?: string; requestId?: string; message: string }

/**
 * Benign binary transport check against POST
 * /v1/admin/safety/evidence/{evidenceId}/access with body
 * { reason: 'Benign synthetic transport verification' }.
 *
 * The fixture is a one-pixel PNG with no real restricted data. The current
 * evidenceId is obtained from getProofIdentity at run time (no hardcoded
 * fixture references). Only structural metadata is returned — never the PNG
 * content and never request header metadata.
 *
 * Representation finding (verified by probe): the buffered typed method
 * returns a JSON envelope { base64Data, sizeBytes, name, type }, and the
 * StreamRaw variant emits a JSON metadata envelope rather than raw PNG bytes,
 * so direct binary passthrough is NOT suitable; the buffered envelope is used.
 */
export default async function (req: { user: User }): Promise<EvidenceTransportResult> {
  requireDraftOwner(req.user)
  try {
    const identity = await craftskySyntheticModerationProof.proof.getProofIdentity()
    const result = await craftskySyntheticModerationProof.safety.postEvidenceAccess(
      identity.data.evidenceId,
      { reason: 'Benign synthetic transport verification' },
    )
    const payload = result.data as {
      base64Data?: unknown
      sizeBytes?: unknown
      type?: unknown
      name?: unknown
    }
    if (typeof payload.base64Data !== 'string') {
      throw new Error('Evidence access envelope did not contain base64Data')
    }
    const compact = payload.base64Data.replace(/[^A-Za-z0-9+/=]/g, '')
    const padding = compact.endsWith('==') ? 2 : compact.endsWith('=') ? 1 : 0
    const decodedByteLength = Math.max(0, Math.floor((compact.length * 3) / 4) - padding)
    const declaredSizeBytes = typeof payload.sizeBytes === 'number' ? payload.sizeBytes : -1
    return {
      ok: true,
      representation: 'buffered JSON envelope containing base64 PNG',
      contentType: typeof payload.type === 'string' ? payload.type : 'unknown',
      declaredSizeBytes,
      base64StringLength: compact.length,
      decodedByteLength,
      sizeMatches: declaredSizeBytes === decodedByteLength,
      directBinaryPassthrough: false,
      streamRepresentation: 'JSON metadata envelope (not raw PNG bytes)',
    }
  } catch (error) {
    return { ok: false, ...extractApiError(error) }
  }
}
