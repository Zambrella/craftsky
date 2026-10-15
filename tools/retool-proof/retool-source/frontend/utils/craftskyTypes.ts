/** Shared shapes mirroring the stage 1 backend functions in /backend/craftsky. */

export type IdentityResult = {
  retoolUser: {
    stableId: number
    fullName: string
    email: string
  }
  apiOperator: {
    actorId: string
    role?: string
    synthetic: boolean
    permissions?: Record<string, boolean>
  }
  caseReference: string
  evidenceId: string
}

export type CaseSummary = {
  caseReference: string
  createdAt?: string
  reportCount?: number
  revision: number
  state: 'open' | 'resolved'
  subjectDid?: string
  subjectType?: string
}

export type CaseDetail = CaseSummary & {
  ownerDid?: string
  safeSnapshot?: { did?: string; type?: string }
  reports?: {
    id?: string
    reporterDid?: string
    reason?: string
    details?: string
    createdAt?: string
  }[]
  decisions?: {
    disposition?: string
    reason?: string
    evidenceNotes?: string
    createdAt?: string
  }[]
  effects?: {
    effectId?: string
    type?: string
    action?: string
    active?: boolean
    rationale?: string
    createdAt?: string
  }[]
  events?: {
    type?: string
    actorId?: string
    sourceSystem?: string
    revision?: number
    createdAt?: string
  }[]
  standing?: {
    activeStrikeCount?: number
    strikeThreshold?: number
    suspended?: boolean
    thresholdSuspended?: boolean
    severeSuspended?: boolean
  }
}

export type MutationFailure = {
  ok: false
  conflict: boolean
  status?: number
  code?: string
  requestId?: string
  message: string
}

export type MutationSuccess = {
  ok: true
  caseReference: string
  revision: number
  replayed: boolean
}

export type MutationResult = MutationSuccess | MutationFailure

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

/** Narrows an untyped hook payload into a MutationResult without inventing success. */
export function asMutationResult(value: unknown): MutationResult {
  if (
    value !== null &&
    typeof value === 'object' &&
    'ok' in value &&
    typeof (value as { ok: unknown }).ok === 'boolean'
  ) {
    return value as MutationResult
  }
  return {
    ok: false,
    conflict: false,
    message: 'Unexpected response shape from the backend function.',
  }
}

/** Narrows an untyped hook payload into an EvidenceTransportResult without inventing success. */
export function asTransportResult(value: unknown): EvidenceTransportResult {
  if (
    value !== null &&
    typeof value === 'object' &&
    'ok' in value &&
    typeof (value as { ok: unknown }).ok === 'boolean'
  ) {
    return value as EvidenceTransportResult
  }
  return { ok: false, message: 'Unexpected response shape from the backend function.' }
}

/** Narrows an untyped hook payload into a CaseDetail, or null when absent. */
export function asCaseDetail(value: unknown): CaseDetail | null {
  if (
    value !== null &&
    typeof value === 'object' &&
    typeof (value as { caseReference?: unknown }).caseReference === 'string' &&
    typeof (value as { revision?: unknown }).revision === 'number'
  ) {
    return value as CaseDetail
  }
  return null
}

/** Narrows an untyped hook payload into an IdentityResult, or null when absent. */
export function asIdentity(value: unknown): IdentityResult | null {
  if (
    value !== null &&
    typeof value === 'object' &&
    typeof (value as { caseReference?: unknown }).caseReference === 'string' &&
    typeof (value as { retoolUser?: unknown }).retoolUser === 'object'
  ) {
    return value as IdentityResult
  }
  return null
}

/** Narrows an untyped hook payload into a case list response, or null when absent. */
export function asCaseList(value: unknown): { cursor?: string; items: CaseSummary[] } | null {
  if (value !== null && typeof value === 'object' && Array.isArray((value as { items?: unknown }).items)) {
    return value as { cursor?: string; items: CaseSummary[] }
  }
  return null
}
