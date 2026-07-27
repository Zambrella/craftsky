import type { FacetFeature, RichTextFacet } from '../domain/types'

const TOKEN_PATTERN =
  /https?:\/\/[^\s<>"'`]+|#[\p{L}\p{M}\p{N}_]+/gu
const MAX_URI_BYTES = 8_192
const MAX_TAG_BYTES = 640
const MAX_TAG_GRAPHEMES = 64
const graphemeSegmenter = new Intl.Segmenter('en', {
  granularity: 'grapheme',
})

function byteLength(value: string): number {
  return new TextEncoder().encode(value).byteLength
}

function trimUrl(token: string): string {
  return token.replace(/[),.!?;:]+$/u, '')
}

export function buildFacets(text: string): RichTextFacet[] {
  const facets: RichTextFacet[] = []
  for (const match of text.matchAll(TOKEN_PATTERN)) {
    const raw = match[0]
    const token = raw.startsWith('http') ? trimUrl(raw) : raw
    const characterStart = match.index
    const byteStart = byteLength(text.slice(0, characterStart))
    const byteEnd = byteStart + byteLength(token)
    let feature: FacetFeature
    if (token.startsWith('#')) {
      const tag = token.slice(1)
      if (
        byteLength(tag) > MAX_TAG_BYTES ||
        [...graphemeSegmenter.segment(tag)].length > MAX_TAG_GRAPHEMES
      ) {
        continue
      }
      feature = {
        $type: 'app.bsky.richtext.facet#tag',
        tag,
      }
    } else {
      if (byteLength(token) > MAX_URI_BYTES) continue
      try {
        const parsed = new URL(token)
        if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
          continue
        }
        feature = {
          $type: 'app.bsky.richtext.facet#link',
          uri: parsed.toString(),
        }
      } catch {
        continue
      }
    }
    facets.push({
      index: { byteStart, byteEnd },
      features: [feature],
    })
  }
  return facets
}
