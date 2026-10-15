/**
 * Normalizes a thrown resource error into metadata the UI can surface.
 * Only code/requestId/status are extracted when the resource provides them;
 * nothing is invented. The message is always carried through. Null or
 * non-Error throws are handled without crashing.
 *
 * Observed resource behavior: typed calls throw an Error whose message is the
 * JSON response body, e.g.
 *   {"error":"case_revision_conflict","message":"case revision conflict","requestId":"..."}
 * The `error` field is used as the API code in that case.
 */
export type NormalizedApiError = {
  status?: number
  code?: string
  requestId?: string
  message: string
}

export function extractApiError(error: unknown): NormalizedApiError {
  const rawMessage =
    error instanceof Error && typeof error.message === 'string' && error.message.length > 0
      ? error.message
      : error === null || error === undefined
        ? 'The resource call threw an empty error.'
        : String(error)
  const out: NormalizedApiError = { message: rawMessage }
  const e = error as {
    status?: unknown
    code?: unknown
    requestId?: unknown
    data?: { status?: unknown; code?: unknown; requestId?: unknown; message?: unknown }
    response?: {
      status?: unknown
      data?: { status?: unknown; code?: unknown; requestId?: unknown; message?: unknown }
    }
  } | null | undefined
  const pickNumber = (v: unknown): number | undefined =>
    typeof v === 'number' ? v : undefined
  const pickString = (v: unknown): string | undefined =>
    typeof v === 'string' && v.length > 0 ? v : undefined

  // Optional access throughout: e may be null/undefined (or any primitive).
  out.status =
    pickNumber(e?.status) ??
    pickNumber(e?.response?.status) ??
    pickNumber(e?.data?.status)
  out.code =
    pickString(e?.code) ??
    pickString(e?.response?.data?.code) ??
    pickString(e?.data?.code)
  out.requestId =
    pickString(e?.requestId) ??
    pickString(e?.response?.data?.requestId) ??
    pickString(e?.data?.requestId)

  // Some resources embed the JSON error body directly in the thrown message.
  if (out.message.trimStart().startsWith('{')) {
    try {
      const parsed = JSON.parse(out.message) as {
        error?: unknown
        message?: unknown
        requestId?: unknown
        code?: unknown
        status?: unknown
      }
      out.code = out.code ?? pickString(parsed.error) ?? pickString(parsed.code)
      out.requestId = out.requestId ?? pickString(parsed.requestId)
      out.status = out.status ?? pickNumber(parsed.status)
      const bodyMessage = pickString(parsed.message)
      if (bodyMessage) out.message = bodyMessage
    } catch {
      // Not JSON — keep the raw message.
    }
  }
  return out
}

/**
 * True only for the conflict contract actually observed on this fixture:
 * code 'case_revision_conflict' or HTTP status 409. Nothing is invented.
 */
export function isRevisionConflict(err: {
  status?: number
  code?: string
}): boolean {
  return err.code === 'case_revision_conflict' || err.status === 409
}

/** Caller-supplied UUID for the idempotency key contract. */
export function isUuid(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value)
  )
}
