export type ExportVariant = 'legacy' | 'detailed'
export type SourceMediaKind = 'image' | 'video' | 'unsupported'

export interface AdapterMedia {
  readonly path: string
  readonly kind: SourceMediaKind
}

export interface AdapterPost {
  readonly variant: ExportVariant
  readonly timestamp: number
  readonly caption: string
  readonly media: readonly AdapterMedia[]
}

export interface AdapterSkipOutcome {
  readonly code:
    | 'invalidTimestamp'
    | 'unpublished'
    | 'unsupportedPost'
  readonly ordinal: number
  readonly timestamp?: number
}

export interface AdapterParseResult {
  readonly posts: readonly AdapterPost[]
  readonly skipped: readonly AdapterSkipOutcome[]
  readonly recognizedPostRecords: number
}

export interface NormalizedAdapterResult {
  readonly posts: readonly AdapterPost[]
  readonly skipped: readonly {
    readonly code: 'ambiguousDuplicate'
    readonly timestamp: number
  }[]
}

export type FacetFeature =
  | {
      readonly $type: 'app.bsky.richtext.facet#link'
      readonly uri: string
    }
  | {
      readonly $type: 'app.bsky.richtext.facet#tag'
      readonly tag: string
    }

export interface RichTextFacet {
  readonly index: {
    readonly byteStart: number
    readonly byteEnd: number
  }
  readonly features: readonly FacetFeature[]
}

export type SafeWarningCode =
  | 'captionRepaired'
  | 'captionTruncated'
  | 'imagesOmitted'
  | 'videoOmitted'
  | 'unsupportedMediaOmitted'
  | 'mediaUnavailable'
  | 'textOnlyConfirmationRequired'

export type UserFacingWarningCode = Exclude<
  SafeWarningCode,
  'captionRepaired'
>

export function isUserFacingWarningCode(
  warning: SafeWarningCode,
): warning is UserFacingWarningCode {
  return warning !== 'captionRepaired'
}

export type SafeSkipCode =
  | 'ambiguousDuplicate'
  | 'invalidTimestamp'
  | 'unpublished'
  | 'videoOnly'
  | 'emptyPost'
  | 'unsupportedPost'
  | 'rkeyCollision'

export interface ReviewMedia {
  readonly token: string
  readonly kind: 'image'
  readonly mime: SupportedReviewMime
  readonly width: number
  readonly height: number
  readonly selected: boolean
}

export type SupportedReviewMime =
  | 'image/jpeg'
  | 'image/png'
  | 'image/webp'

export interface ReviewPost {
  readonly itemKey: string
  readonly rkey: string
  readonly createdAt: string
  readonly caption: string
  readonly initialCaption: string
  readonly media: readonly ReviewMedia[]
  readonly warnings: readonly SafeWarningCode[]
  readonly selected: boolean
  readonly needsTextOnlyConfirmation: boolean
  readonly textOnlyConfirmed: boolean
}

export interface SkippedPost {
  readonly itemKey: string
  readonly createdAt?: string
  readonly code: SafeSkipCode
}

export interface ReviewCounts {
  readonly selectedPosts: number
  readonly selectedImages: number
  readonly transformedPosts: number
  readonly warningPosts: number
  readonly skippedPosts: number
}

export interface ReviewManifest {
  readonly schemaVersion: 1
  readonly fingerprint: string
  readonly posts: readonly ReviewPost[]
  readonly skipped: readonly SkippedPost[]
  readonly counts: ReviewCounts
}
