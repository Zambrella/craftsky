import { describe, expect, it } from 'vitest'

import { prepareCaption, repairMojibake } from './caption'

describe('caption transformations (UT-005)', () => {
  it('repairs only a reversible UTF-8/Latin-1 mojibake round trip', () => {
    expect(repairMojibake('cafÃ©')).toEqual({
      text: 'café',
      repaired: true,
    })
    expect(repairMojibake('ordinary text')).toEqual({
      text: 'ordinary text',
      repaired: false,
    })
  })

  it('truncates by grapheme and returns visible warning state', () => {
    const caption = `${'a'.repeat(1_999)}👨‍👩‍👧‍👦z`
    const prepared = prepareCaption(caption)
    expect([...new Intl.Segmenter('en', { granularity: 'grapheme' }).segment(prepared.text)]).toHaveLength(2_000)
    expect(prepared.text.endsWith('👨‍👩‍👧‍👦')).toBe(true)
    expect(prepared.warnings).toContain('captionTruncated')
  })

  it('also truncates only at grapheme boundaries to the wire byte limit', () => {
    const family = '👨‍👩‍👧‍👦'
    const prepared = prepareCaption(family.repeat(2_000))
    expect(new TextEncoder().encode(prepared.text).byteLength).toBeLessThanOrEqual(
      20_000,
    )
    expect(prepared.text.replaceAll(family, '')).toBe('')
    expect(prepared.warnings).toContain('captionTruncated')
  })
})
