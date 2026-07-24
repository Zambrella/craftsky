import type { SafeErrorCode } from '../privacy/safeDiagnostics'

const MEDIA_ERROR_CODES = new Set([
  'mediaUnavailable',
  'sourceImageLimit',
  'imageDimensionLimit',
  'imagePixelLimit',
  'animatedImageUnsupported',
  'unsupportedImageFormat',
  'finalImageLimit',
])

const ARCHIVE_LIMIT_ERROR_CODES = new Set<SafeErrorCode>([
  'archiveEntryLimit',
  'centralDirectoryLimit',
  'candidateMetadataLimit',
])

export function safeWorkerErrorCode(error: unknown): SafeErrorCode {
  const message = error instanceof Error ? error.message : ''
  if (message === 'browserMediaUnsupported') {
    return 'browserUnsupported'
  }
  if (ARCHIVE_LIMIT_ERROR_CODES.has(message as SafeErrorCode)) {
    return message as SafeErrorCode
  }
  return MEDIA_ERROR_CODES.has(message)
    ? 'mediaUnavailable'
    : 'archiveUnsupported'
}
