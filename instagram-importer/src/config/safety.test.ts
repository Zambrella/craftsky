import { describe, expect, it } from 'vitest'

import {
  IMPORT_SAFETY_POLICY,
  withinCompressionRatio,
  withinInclusiveLimit,
} from './safety'

describe('instagram-import-v1 safety policy (UT-001)', () => {
  it('keeps every approved resource limit in one named policy', () => {
    expect(IMPORT_SAFETY_POLICY).toEqual({
      id: 'instagram-import-v1',
      maxEntries: 100_000,
      maxCentralDirectoryBytes: 64 * 1024 * 1024,
      maxCandidateJsonBytes: 32 * 1024 * 1024,
      maxCombinedCandidateJsonBytes: 128 * 1024 * 1024,
      maxPosts: 25_000,
      maxSourceImageBytes: 64 * 1024 * 1024,
      maxDecodedPixels: 25_000_000,
      maxDimension: 12_000,
      maxFinalImageWidth: 4_000,
      maxFinalImageHeight: 4_000,
      maxCompressionRatio: 200,
      maxConcurrentImageTransforms: 1,
      maxFinalBlobBytes: 2_000_000,
    })
    expect(IMPORT_SAFETY_POLICY).not.toHaveProperty('maxArchiveBytes')
  })

  it('treats configured limits as inclusive', () => {
    expect(withinInclusiveLimit(9, 10)).toBe(true)
    expect(withinInclusiveLimit(10, 10)).toBe(true)
    expect(withinInclusiveLimit(11, 10)).toBe(false)
  })

  it.each(
    Object.entries(IMPORT_SAFETY_POLICY).filter(
      (entry): entry is [string, number] => typeof entry[1] === 'number',
    ),
  )('checks one below, at, and above %s', (_name, limit) => {
    expect(withinInclusiveLimit(Math.max(0, limit - 1), limit)).toBe(true)
    expect(withinInclusiveLimit(limit, limit)).toBe(true)
    expect(withinInclusiveLimit(limit + 1, limit)).toBe(false)
  })

  it('rejects zero-byte and over-limit compressed representations', () => {
    expect(withinCompressionRatio(199, 1, 200)).toBe(true)
    expect(withinCompressionRatio(200, 1, 200)).toBe(true)
    expect(withinCompressionRatio(201, 1, 200)).toBe(false)
    expect(withinCompressionRatio(1, 0, 200)).toBe(false)
    expect(withinCompressionRatio(0, 0, 200)).toBe(true)
  })
})
