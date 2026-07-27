export type CaptionWarning = 'captionRepaired' | 'captionTruncated'

export interface PreparedCaption {
  readonly text: string
  readonly warnings: readonly CaptionWarning[]
}

const SUSPICIOUS_MOJIBAKE = /(?:Ã.|Â.|â.|ð.|�)/
const MAX_GRAPHEMES = 2_000
const MAX_BYTES = 20_000

export function repairMojibake(input: string): {
  readonly text: string
  readonly repaired: boolean
} {
  if (!SUSPICIOUS_MOJIBAKE.test(input)) {
    return { text: input, repaired: false }
  }
  const codeUnits = [...input]
  if (codeUnits.some((character) => character.charCodeAt(0) > 255)) {
    return { text: input, repaired: false }
  }
  const bytes = Uint8Array.from(
    codeUnits.map((character) => character.charCodeAt(0)),
  )
  try {
    const decoded = new TextDecoder('utf-8', { fatal: true }).decode(bytes)
    const roundTrip = [...new TextEncoder().encode(decoded)]
      .map((byte) => String.fromCharCode(byte))
      .join('')
    if (roundTrip === input && decoded !== input) {
      return { text: decoded, repaired: true }
    }
  } catch {
    // Ambiguous or invalid byte-like text stays unchanged.
  }
  return { text: input, repaired: false }
}

export function prepareCaption(input: string): PreparedCaption {
  const repaired = repairMojibake(input)
  const warnings: CaptionWarning[] = repaired.repaired
    ? ['captionRepaired']
    : []
  const segmenter = new Intl.Segmenter('en', { granularity: 'grapheme' })
  const graphemes = [...segmenter.segment(repaired.text)].map(
    (part) => part.segment,
  )
  let retained = graphemes.slice(0, MAX_GRAPHEMES)
  const encoder = new TextEncoder()
  while (
    retained.length > 0 &&
    encoder.encode(retained.join('')).byteLength > MAX_BYTES
  ) {
    retained = retained.slice(0, -1)
  }
  const text = retained.join('')
  if (text !== repaired.text) {
    warnings.push('captionTruncated')
  }
  return { text, warnings }
}
