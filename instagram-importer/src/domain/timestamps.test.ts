import { describe, expect, it } from 'vitest'

import { normalizeSourceTimestamp } from './timestamps'

describe('source timestamp window (UT-004)', () => {
  const now = new Date('2026-07-23T12:00:00.000Z')
  const earliest = '2010-10-06T00:00:00.000Z'
  const latest = '2026-07-24T12:00:00.000Z'

  it('accepts exact inclusive boundaries and seconds or milliseconds', () => {
    expect(
      normalizeSourceTimestamp(Date.parse(earliest) / 1000, now),
    ).toBe(earliest)
    expect(normalizeSourceTimestamp(Date.parse(latest), now)).toBe(latest)
  })

  it('rejects malformed and out-of-window values without substituting now', () => {
    for (const value of [
      undefined,
      'not-a-date',
      Date.parse(earliest) / 1000 - 1,
      Date.parse(latest) + 1,
    ]) {
      expect(normalizeSourceTimestamp(value, now)).toBeNull()
    }
  })
})
