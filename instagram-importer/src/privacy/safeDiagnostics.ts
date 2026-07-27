export const SAFE_ERROR_CODES = [
  'archiveUnsupported',
  'archiveEntryLimit',
  'centralDirectoryLimit',
  'candidateMetadataLimit',
  'browserUnsupported',
  'invalidTimestamp',
  'mediaUnavailable',
  'oauthDenied',
  'oauthAuthorityMismatch',
  'notCraftskyMember',
  'recordCollision',
  'publicationFailed',
  'rollbackConflict',
  'previewWritesDisabled',
] as const

export type SafeErrorCode = (typeof SAFE_ERROR_CODES)[number]

const SAFE_ERROR_CODE_SET = new Set<string>(SAFE_ERROR_CODES)

export function isSafeErrorCode(value: unknown): value is SafeErrorCode {
  return typeof value === 'string' && SAFE_ERROR_CODE_SET.has(value)
}

export interface SafeErrorContext {
  readonly inspectedEntries?: number
  readonly candidatePosts?: number
  readonly selectedPosts?: number
  readonly selectedImages?: number
  readonly attempt?: number
}

export class SafeImporterError extends Error {
  readonly code: SafeErrorCode
  readonly context: SafeErrorContext

  constructor(code: SafeErrorCode, context: SafeErrorContext = {}) {
    super(code)
    this.name = 'SafeImporterError'
    this.code = code
    this.context = Object.freeze({ ...context })
  }

  toJSON(): { code: SafeErrorCode; context: SafeErrorContext } {
    return { code: this.code, context: this.context }
  }

  override toString(): string {
    return `SafeImporterError(${this.code})`
  }
}

const SAFE_PROGRESS_STATUSES = new Set([
  'remaining',
  'running',
  'created',
  'alreadyExisting',
  'collision',
  'failed',
  'rolledBack',
  'rollbackAbsent',
  'rollbackConflict',
  'rollbackFailed',
])

export function safeProgressDiagnostic(value: unknown): {
  readonly status: string
  readonly attempts: number
} {
  if (typeof value !== 'object' || value === null) {
    return { status: 'unknown', attempts: 0 }
  }
  const record = value as Record<string, unknown>
  const status =
    typeof record.status === 'string' &&
    SAFE_PROGRESS_STATUSES.has(record.status)
      ? record.status
      : 'unknown'
  const attempts =
    typeof record.attempts === 'number' &&
    Number.isInteger(record.attempts) &&
    record.attempts >= 0
      ? record.attempts
      : 0
  return { status, attempts }
}
