import { describe, expect, it } from 'vitest'

import {
  CRAFTSKY_POST_COLLECTION,
  INSTAGRAM_IMPORT_SOURCE,
  isValidCraftskyFeedPostRecord,
} from './craftsky-feed-post'

describe('generated CraftSky post contract (IT-001)', () => {
  const ordinary = {
    $type: CRAFTSKY_POST_COLLECTION,
    text: 'Historical post',
    createdAt: '2024-01-02T03:04:05.000Z',
  }

  it('validates ordinary, Instagram, and unknown bounded sources', () => {
    expect(isValidCraftskyFeedPostRecord(ordinary)).toBe(true)
    expect(
      isValidCraftskyFeedPostRecord({
        ...ordinary,
        externalImport: { source: INSTAGRAM_IMPORT_SOURCE },
      }),
    ).toBe(true)
    expect(
      isValidCraftskyFeedPostRecord({
        ...ordinary,
        externalImport: { source: 'future-source' },
      }),
    ).toBe(true)
  })

  it('rejects missing, overlong, or detail-bearing provenance', () => {
    expect(
      isValidCraftskyFeedPostRecord({
        ...ordinary,
        externalImport: {},
      }),
    ).toBe(false)
    expect(
      isValidCraftskyFeedPostRecord({
        ...ordinary,
        externalImport: { source: 'x'.repeat(65) },
      }),
    ).toBe(false)
    expect(
      isValidCraftskyFeedPostRecord({
        ...ordinary,
        externalImport: {
          source: 'instagram',
          handle: 'must-not-be-public',
        },
      }),
    ).toBe(false)
  })
})
