import { describe, expect, it } from 'vitest'

import type { ReviewPost } from './types'
import { deterministicRecordKeys } from './recordKey'
import { partitionRecordKeyCollisions } from './rkeyCollisions'

const post = (itemKey: string, rkey: string): ReviewPost => ({
  itemKey,
  rkey,
  createdAt: '2024-01-02T03:04:05.000Z',
  caption: itemKey,
  initialCaption: itemKey,
  media: [],
  warnings: [],
  selected: true,
  needsTextOnlyConfirmation: false,
  textOnlyConfirmed: false,
})

describe('deterministic rkey collision safety (UT-008)', () => {
  it('skips every affected source item instead of choosing or overwriting', () => {
    const result = partitionRecordKeyCollisions([
      post('a', 'same'),
      post('b', 'same'),
      post('c', 'different'),
    ])
    expect(result.posts.map((item) => item.itemKey)).toEqual(['c'])
    expect(result.skipped).toEqual([
      expect.objectContaining({ itemKey: 'a', code: 'rkeyCollision' }),
      expect.objectContaining({ itemKey: 'b', code: 'rkeyCollision' }),
    ])
  })

  it('fails closed for canonically identical sources without retaining a winner', () => {
    const createdAt = '2024-01-02T03:04:05.000Z'
    const [composed, decomposed, distinct] = deterministicRecordKeys([
      {
        createdAt,
        canonicalIdentityKey: '00000000000000000000000000000001',
      },
      {
        createdAt,
        canonicalIdentityKey: '00000000000000000000000000000001',
      },
      {
        createdAt,
        canonicalIdentityKey: '00000000000000000000000000000002',
      },
    ])
    const candidates = [
      post('composed', composed),
      post('decomposed', decomposed),
      post('distinct', distinct),
    ]
    expect(composed).toBe(decomposed)
    expect(distinct).not.toBe(composed)

    const result = partitionRecordKeyCollisions(candidates)
    expect(result.posts.map((candidate) => candidate.itemKey)).toEqual([
      'distinct',
    ])
    expect(result.skipped).toEqual([
      expect.objectContaining({
        itemKey: 'composed',
        code: 'rkeyCollision',
      }),
      expect.objectContaining({
        itemKey: 'decomposed',
        code: 'rkeyCollision',
      }),
    ])
  })
})
