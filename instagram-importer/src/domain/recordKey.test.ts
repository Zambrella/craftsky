import { describe, expect, it } from 'vitest'

import { deterministicRecordKeys } from './recordKey'

const identityKey = (value: number): string =>
  value.toString(16).padStart(32, '0')

describe('deterministic imported post rkey (UT-008)', () => {
  it('is a stable valid TID and distinguishes same-time source identities', () => {
    const sources = [
      {
        createdAt: '2024-01-02T03:04:05.000Z',
        canonicalIdentityKey: identityKey(1),
      },
      {
        createdAt: '2024-01-02T03:04:05.000Z',
        canonicalIdentityKey: identityKey(2),
      },
    ]
    const [first, second] = deterministicRecordKeys(sources)
    expect(first).toMatch(/^[234567abcdefghijklmnopqrstuvwxyz]{13}$/)
    expect(deterministicRecordKeys(sources)[0]).toBe(first)
    expect(second).not.toBe(first)
  })

  it('keeps an existing source key when a later export adds an earlier identity', () => {
    const existing = {
      createdAt: '2024-01-02T03:04:05.000Z',
      canonicalIdentityKey: identityKey(2),
    }
    const added = {
      createdAt: existing.createdAt,
      canonicalIdentityKey: identityKey(1),
    }

    const [originalKey] = deterministicRecordKeys([existing])
    const [, laterKey] = deterministicRecordKeys([added, existing])

    expect(laterKey).toBe(originalKey)
  })

  it('assigns 1,025 same-time source identities stable distinct TIDs independent of input order', () => {
    const sources = Array.from({ length: 1_025 }, (_, index) => ({
      createdAt: '2024-01-02T03:04:05.000Z',
      canonicalIdentityKey: identityKey(index + 1),
    }))

    const forward = deterministicRecordKeys(sources)
    const reverseSources = [...sources].reverse()
    const reverse = deterministicRecordKeys(reverseSources)
    const reverseByIdentity = new Map(
      reverseSources.map((source, index) => [
        source.canonicalIdentityKey,
        reverse[index],
      ]),
    )

    expect(forward).toHaveLength(sources.length)
    expect(new Set(forward).size).toBe(sources.length)
    expect(
      forward.every((rkey) =>
        /^[234567abcdefghijklmnopqrstuvwxyz]{13}$/.test(rkey),
      ),
    ).toBe(true)
    expect(
      sources.map((source) =>
        reverseByIdentity.get(source.canonicalIdentityKey),
      ),
    ).toEqual(forward)
    expect(deterministicRecordKeys(sources)).toEqual(forward)
  })
})
