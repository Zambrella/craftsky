import { describe, expect, it } from 'vitest'

import { safeWorkerErrorCode } from './errorCode'

describe('worker compatibility diagnostics', () => {
  it('fails the archive preflight for a browser missing media APIs', () => {
    expect(
      safeWorkerErrorCode(new Error('browserMediaUnsupported')),
    ).toBe('browserUnsupported')
  })

  it('keeps source-specific failures content-free', () => {
    expect(safeWorkerErrorCode(new Error('imagePixelLimit'))).toBe(
      'mediaUnavailable',
    )
    expect(safeWorkerErrorCode(new Error('PRIVATE CAPTION'))).toBe(
      'archiveUnsupported',
    )
  })

  it.each([
    'archiveEntryLimit',
    'centralDirectoryLimit',
    'candidateMetadataLimit',
  ] as const)('preserves the safe configured limit code %s', (code) => {
    expect(safeWorkerErrorCode(new Error(code))).toBe(code)
  })
})
