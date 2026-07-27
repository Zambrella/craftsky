import type { ReviewPost, SkippedPost } from './types'

export function partitionRecordKeyCollisions(
  posts: readonly ReviewPost[],
): {
  readonly posts: ReviewPost[]
  readonly skipped: SkippedPost[]
} {
  const counts = new Map<string, number>()
  for (const post of posts) {
    counts.set(post.rkey, (counts.get(post.rkey) ?? 0) + 1)
  }
  const retained: ReviewPost[] = []
  const skipped: SkippedPost[] = []
  for (const post of posts) {
    if ((counts.get(post.rkey) ?? 0) > 1) {
      skipped.push({
        itemKey: post.itemKey,
        createdAt: post.createdAt,
        code: 'rkeyCollision',
      })
    } else {
      retained.push(post)
    }
  }
  return { posts: retained, skipped }
}
