import type { ReviewPost } from '../domain/types'
import {
  CRAFTSKY_POST_COLLECTION,
  INSTAGRAM_IMPORT_SOURCE,
  POST_IMAGE_ACCEPT,
  POST_IMAGE_MAX_COUNT,
  POST_TEXT_MAX_BYTES,
  POST_TEXT_MAX_GRAPHEMES,
  isValidCraftskyFeedPostRecord,
  type CraftskyFeedPostRecord,
} from '../generated/craftsky-feed-post'
import { buildFacets } from '../text/facets'

export interface UploadedReviewMedia {
  readonly token: string
  readonly blob: unknown
}

export function assertReviewPostPublishable(post: ReviewPost): void {
  if (post.needsTextOnlyConfirmation && !post.textOnlyConfirmed) {
    throw new Error('textOnlyConfirmationRequired')
  }
  const selectedMedia = post.media.filter((media) => media.selected)
  if (
    !/^[234567a-z]{13}$/u.test(post.rkey) ||
    !Number.isFinite(Date.parse(post.createdAt)) ||
    new TextEncoder().encode(post.caption).byteLength >
      POST_TEXT_MAX_BYTES ||
    [
      ...new Intl.Segmenter('en', { granularity: 'grapheme' }).segment(
        post.caption,
      ),
    ].length > POST_TEXT_MAX_GRAPHEMES ||
    selectedMedia.length > POST_IMAGE_MAX_COUNT ||
    (post.caption.length === 0 && selectedMedia.length === 0) ||
    selectedMedia.some(
      (media) =>
        !POST_IMAGE_ACCEPT.includes(media.mime) ||
        !Number.isInteger(media.width) ||
        !Number.isInteger(media.height) ||
        media.width < 1 ||
        media.height < 1,
    )
  ) {
    throw new Error('invalidPostRecord')
  }
}

export function buildPostRecord(
  post: ReviewPost,
  uploadedMedia: readonly UploadedReviewMedia[],
): CraftskyFeedPostRecord {
  assertReviewPostPublishable(post)

  const blobs = new Map(
    uploadedMedia.map((media) => [media.token, media.blob] as const),
  )
  const images = post.media
    .filter((media) => media.selected)
    .map((media) => {
      const image = blobs.get(media.token)
      if (!image) throw new Error('missingUploadedImage')
      return {
        image,
        alt: '',
        aspectRatio: {
          width: media.width,
          height: media.height,
        },
      }
    })
  const facets = buildFacets(post.caption)
  const record: CraftskyFeedPostRecord = {
    $type: CRAFTSKY_POST_COLLECTION,
    text: post.caption,
    ...(facets.length > 0 ? { facets } : {}),
    ...(images.length > 0 ? { images } : {}),
    createdAt: post.createdAt,
    externalImport: { source: INSTAGRAM_IMPORT_SOURCE },
  }

  if (!isValidCraftskyFeedPostRecord(record)) {
    throw new Error('invalidPostRecord')
  }
  if (record.text.length === 0 && images.length === 0) {
    throw new Error('emptyPost')
  }
  return record
}
