import { describe, expect, it } from 'vitest'

import {
  SafeImporterError,
  safeProgressDiagnostic,
} from './safeDiagnostics'

describe('safe diagnostics (UT-012)', () => {
  it('retains only closed codes/counts and never stringifies content or CIDs', () => {
    const sentinel = 'PRIVATE-CAPTION-SENTINEL'
    const cid = 'bafy-private-created-cid'
    const error = new SafeImporterError('archiveUnsupported', {
      inspectedEntries: 3,
    })
    const diagnostic = JSON.stringify({
      error,
      progress: safeProgressDiagnostic({
        status: 'created',
        attempts: 1,
        createdCid: cid,
        caption: sentinel,
      }),
    })
    expect(diagnostic).toContain('archiveUnsupported')
    expect(diagnostic).toContain('inspectedEntries')
    expect(diagnostic).not.toContain(sentinel)
    expect(diagnostic).not.toContain(cid)
    expect(
      safeProgressDiagnostic({
        status: 'rollbackAbsent',
        attempts: 1,
        createdCid: cid,
      }),
    ).toEqual({ status: 'rollbackAbsent', attempts: 1 })
  })
})
