import type {
  ReviewCounts,
  ReviewPost,
  SafeWarningCode,
} from '../domain/types'
import { prepareCaption } from '../text/caption'

const MAX_TEXT_BYTES = 20_000

function warnings(
  input: readonly SafeWarningCode[],
): readonly SafeWarningCode[] {
  return [...new Set(input)].slice(0, 8)
}

function enforcePostShape(post: ReviewPost): ReviewPost {
  const selectedImages = post.media.filter((item) => item.selected).length
  const hasText = post.caption.length > 0
  const textOnly = selectedImages === 0 && hasText
  return {
    ...post,
    selected: post.selected && (hasText || selectedImages > 0),
    needsTextOnlyConfirmation: textOnly,
    textOnlyConfirmed: textOnly ? post.textOnlyConfirmed : false,
  }
}

export function calculateReviewCounts(
  posts: readonly ReviewPost[],
  skippedPosts: number,
): ReviewCounts {
  const selected = posts.filter((post) => post.selected)
  return {
    selectedPosts: selected.length,
    selectedImages: selected.reduce(
      (count, post) =>
        count + post.media.filter((item) => item.selected).length,
      0,
    ),
    transformedPosts: posts.filter(
      (post) =>
        post.caption !== post.initialCaption ||
        post.warnings.some((warning) =>
          [
            'captionRepaired',
            'captionTruncated',
            'imagesOmitted',
            'videoOmitted',
            'unsupportedMediaOmitted',
            'mediaUnavailable',
          ].includes(warning),
        ),
    ).length,
    warningPosts: posts.filter((post) => post.warnings.length > 0).length,
    skippedPosts,
  }
}

export function updatePostSelection(
  posts: readonly ReviewPost[],
  itemKey: string,
  selected: boolean,
): ReviewPost[] {
  return posts.map((post) =>
    post.itemKey === itemKey
      ? enforcePostShape({ ...post, selected })
      : post,
  )
}

export function updateMediaSelection(
  posts: readonly ReviewPost[],
  itemKey: string,
  token: string,
  selected: boolean,
): ReviewPost[] {
  return posts.map((post) => {
    if (post.itemKey !== itemKey) return post
    return enforcePostShape({
      ...post,
      media: post.media.map((item) =>
        item.token === token ? { ...item, selected } : item,
      ),
      textOnlyConfirmed: false,
    })
  })
}

export function updateCaption(
  posts: readonly ReviewPost[],
  itemKey: string,
  caption: string,
): ReviewPost[] {
  return posts.map((post) => {
    if (post.itemKey !== itemKey) return post
    const prepared = prepareCaption(caption)
    const segmenter = new Intl.Segmenter('en', {
      granularity: 'grapheme',
    })
    const parts = [...segmenter.segment(prepared.text)].map(
      (part) => part.segment,
    )
    while (
      parts.length > 0 &&
      new TextEncoder().encode(parts.join('')).byteLength > MAX_TEXT_BYTES
    ) {
      parts.pop()
    }
    const finalCaption = parts.join('')
    const wasByteTruncated = finalCaption !== prepared.text
    return enforcePostShape({
      ...post,
      caption: finalCaption,
      warnings: warnings([
        ...post.warnings.filter(
          (warning) => warning !== 'captionTruncated',
        ),
        ...(prepared.warnings.includes('captionTruncated') ||
        wasByteTruncated
          ? (['captionTruncated'] as const)
          : []),
      ]),
      textOnlyConfirmed: false,
    })
  })
}

export function setAllPostSelection(
  posts: readonly ReviewPost[],
  selected: boolean,
): ReviewPost[] {
  return posts.map((post) =>
    enforcePostShape({ ...post, selected }),
  )
}

export function setPostSelectionByKey(
  posts: readonly ReviewPost[],
  itemKeys: ReadonlySet<string>,
  selected: boolean,
): ReviewPost[] {
  return posts.map((post) =>
    itemKeys.has(post.itemKey)
      ? enforcePostShape({ ...post, selected })
      : post,
  )
}

export function confirmTextOnly(
  posts: readonly ReviewPost[],
  itemKey: string,
): ReviewPost[] {
  return posts.map((post) =>
    post.itemKey === itemKey && post.needsTextOnlyConfirmation
      ? { ...post, textOnlyConfirmed: true }
      : post,
  )
}
