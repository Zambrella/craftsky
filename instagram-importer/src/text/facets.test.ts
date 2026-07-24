import { describe, expect, it } from 'vitest'

import { buildFacets } from './facets'

describe('final caption facets (UT-006)', () => {
  it('uses UTF-8 byte ranges for URLs and hashtags but not Instagram handles', () => {
    const text =
      '🧶 See https://craftsky.social/x and #knitting with @synthetic_maker'
    const facets = buildFacets(text)

    expect(facets).toEqual([
      {
        index: { byteStart: 9, byteEnd: 34 },
        features: [
          {
            $type: 'app.bsky.richtext.facet#link',
            uri: 'https://craftsky.social/x',
          },
        ],
      },
      {
        index: { byteStart: 39, byteEnd: 48 },
        features: [
          {
            $type: 'app.bsky.richtext.facet#tag',
            tag: 'knitting',
          },
        ],
      },
    ])
  })

  it('leaves an oversized URL as plain text instead of producing an invalid facet', () => {
    expect(buildFacets(`https://example.com/${'a'.repeat(9_000)}`)).toEqual(
      [],
    )
  })

  it('facets a hashtag at the 64-grapheme boundary', () => {
    const tag = 'a'.repeat(64)

    expect(buildFacets(`#${tag}`)).toEqual([
      {
        index: { byteStart: 0, byteEnd: 65 },
        features: [
          {
            $type: 'app.bsky.richtext.facet#tag',
            tag,
          },
        ],
      },
    ])
  })

  it('leaves a 65-grapheme hashtag wholly plain instead of faceting its prefix', () => {
    expect(buildFacets(`#${'a'.repeat(65)}`)).toEqual([])
  })
})
