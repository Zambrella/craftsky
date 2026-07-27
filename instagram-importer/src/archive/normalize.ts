import type {
  AdapterPost,
  NormalizedAdapterResult,
} from '../domain/types'

function materiallyEquivalent(left: AdapterPost, right: AdapterPost): boolean {
  return (
    left.timestamp === right.timestamp &&
    left.caption === right.caption &&
    left.media.length === right.media.length &&
    left.media.every(
      (item, index) =>
        item.path.normalize('NFC') ===
          right.media[index]?.path.normalize('NFC') &&
        item.kind === right.media[index]?.kind,
    )
  )
}

function canonicalPostKey(post: AdapterPost): string {
  return JSON.stringify([
    post.timestamp,
    post.caption,
    ...post.media.map((item) => [
      item.kind,
      item.path.normalize('NFC'),
    ]),
  ])
}

function compareStrings(left: string, right: string): number {
  return left < right ? -1 : left > right ? 1 : 0
}

function canonicalGroupKey(posts: readonly AdapterPost[]): string {
  return posts
    .map(canonicalPostKey)
    .sort(compareStrings)
    .join('\u0000')
}

export function mergeAdapterPosts(
  input: readonly AdapterPost[],
): NormalizedAdapterResult {
  const parents = input.map((_, index) => index)
  const find = (index: number): number => {
    let root = index
    while (parents[root] !== root) root = parents[root]
    while (parents[index] !== index) {
      const next = parents[index]
      parents[index] = root
      index = next
    }
    return root
  }
  const union = (left: number, right: number): void => {
    const leftRoot = find(left)
    const rightRoot = find(right)
    if (leftRoot !== rightRoot) parents[rightRoot] = leftRoot
  }
  const seenMediaPath = new Map<string, number>()
  input.forEach((post, index) => {
    for (const media of post.media) {
      const key = media.path.normalize('NFC')
      const mediaMatch = seenMediaPath.get(key)
      if (mediaMatch !== undefined) union(index, mediaMatch)
      else seenMediaPath.set(key, index)
    }
  })

  const byIdentity = new Map<number, AdapterPost[]>()
  input.forEach((post, index) => {
    const root = find(index)
    const current = byIdentity.get(root) ?? []
    current.push(post)
    byIdentity.set(root, current)
  })
  if (input.length === 0) {
    return { posts: [], skipped: [] }
  }

  const posts: AdapterPost[] = []
  const skipped: Array<{
    code: 'ambiguousDuplicate'
    timestamp: number
  }> = []
  const canonicalGroups = [...byIdentity.values()].sort((left, right) =>
    compareStrings(canonicalGroupKey(left), canonicalGroupKey(right)),
  )
  for (const candidates of canonicalGroups) {
    const first = candidates[0]
    if (candidates.every((candidate) => materiallyEquivalent(first, candidate))) {
      posts.push(first)
    } else {
      skipped.push({ code: 'ambiguousDuplicate', timestamp: first.timestamp })
    }
  }
  posts.sort(
    (left, right) =>
      left.timestamp - right.timestamp ||
      compareStrings(canonicalPostKey(left), canonicalPostKey(right)),
  )
  skipped.sort((left, right) => left.timestamp - right.timestamp)
  return { posts, skipped }
}
