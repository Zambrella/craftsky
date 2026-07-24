import type { ReviewPost } from '../domain/types'

export interface FingerprintItem {
  readonly itemKey: string
  readonly rkey: string
  readonly publicShape?: string
}

export async function deriveManifestFingerprint(
  items: readonly FingerprintItem[],
): Promise<string> {
  const sortedItems = [...items].sort((left, right) => {
    const leftKey = `${left.itemKey}\u0000${left.rkey}\u0000${left.publicShape ?? ''}`
    const rightKey = `${right.itemKey}\u0000${right.rkey}\u0000${right.publicShape ?? ''}`
    return leftKey < rightKey ? -1 : leftKey > rightKey ? 1 : 0
  })
  const canonical = [
    'instagram-import-manifest-v1',
    ...sortedItems.map(
      (item) =>
        `${item.itemKey}:${item.rkey}:${item.publicShape ?? ''}`,
    ),
  ].join('\u0000')
  const digest = await crypto.subtle.digest(
    'SHA-256',
    new TextEncoder().encode(canonical),
  )
  return [...new Uint8Array(digest)]
    .map((byte) => byte.toString(16).padStart(2, '0'))
    .join('')
}

export async function deriveReviewedManifestFingerprint(
  posts: readonly ReviewPost[],
): Promise<string> {
  return deriveManifestFingerprint(
    posts
      .filter((post) => post.selected)
      .map((post) => ({
        itemKey: post.itemKey,
        rkey: post.rkey,
        publicShape: JSON.stringify({
          imageSelection: post.media.map((media) => media.selected),
          textOnlyConfirmed: post.textOnlyConfirmed,
        }),
      })),
  )
}
