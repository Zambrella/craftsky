import {
  BlobReader,
  Writer,
  ZipReader,
  type FileEntry,
} from '@zip.js/zip.js'

import {
  IMPORT_SAFETY_POLICY,
  withinCompressionRatio,
} from '../config/safety'
import {
  type AdapterSkipOutcome,
  type AdapterPost,
  type ReviewManifest,
  type ReviewMedia,
  type ReviewPost,
  type SafeWarningCode,
  type SkippedPost,
} from '../domain/types'
import { deterministicRecordKeys } from '../domain/recordKey'
import { partitionRecordKeyCollisions } from '../domain/rkeyCollisions'
import { normalizeSourceTimestamp } from '../domain/timestamps'
import { sanitizeImage, type SanitizedImage } from '../media/sanitize'
import { prepareCaption } from '../text/caption'
import { calculateReviewCounts } from '../review/reviewState'
import { parseDetailedExport, parseLegacyExport } from './adapters'
import { mergeAdapterPosts } from './normalize'
import {
  classifyPostMetadataPath,
  isSafeArchivePath,
  resolveArchiveMediaPath,
} from './paths'
import { inspectZipEnvelope } from './zipEnvelope'

export interface ArchiveInspection {
  readonly manifest: ReviewManifest
  readonly reader: ZipReader<Blob>
  readonly tokenEntries: Map<string, FileEntry>
}

export type InspectionProgress = (
  phase: 'directory' | 'metadata' | 'mediaPreflight',
  completed: number,
  total?: number,
) => void

interface Candidate {
  readonly entry: FileEntry
  readonly wrapper: string
  readonly variant: 'legacy' | 'detailed'
}

interface PendingAdapterSkip extends AdapterSkipOutcome {
  readonly candidateIndex: number
  readonly variant: 'legacy' | 'detailed'
}

interface PendingReviewPost {
  readonly canonicalIdentityKey: string
  readonly value: Omit<ReviewPost, 'rkey'>
}

const SUPPORTED_ZIP_COMPRESSION_METHODS = new Set([0, 8, 9])

class BoundedUint8ArrayWriter extends Writer<Uint8Array> {
  private bytes = new Uint8Array()
  private offset = 0

  constructor(
    private readonly maxBytes: number,
    private readonly overflowCode: string,
  ) {
    super()
  }

  override init(size = 0): Promise<void> {
    if (
      !Number.isSafeInteger(size) ||
      size < 0 ||
      size > this.maxBytes
    ) {
      return Promise.reject(new Error(this.overflowCode))
    }
    this.bytes = new Uint8Array(size)
    this.offset = 0
    return Promise.resolve(super.init?.(size)).then(() => undefined)
  }

  override writeUint8Array(array: Uint8Array): Promise<void> {
    const nextOffset = this.offset + array.byteLength
    if (
      !Number.isSafeInteger(nextOffset) ||
      nextOffset > this.maxBytes ||
      nextOffset > this.bytes.byteLength
    ) {
      return Promise.reject(new Error(this.overflowCode))
    }
    this.bytes.set(array, this.offset)
    this.offset = nextOffset
    return Promise.resolve()
  }

  override getData(): Promise<Uint8Array> {
    return Promise.resolve(this.bytes.slice(0, this.offset))
  }
}

async function digest(value: string): Promise<string> {
  const bytes = new TextEncoder().encode(value)
  const hash = await crypto.subtle.digest('SHA-256', bytes)
  return [...new Uint8Array(hash)]
    .map((byte) => byte.toString(16).padStart(2, '0'))
    .join('')
}

function dedupeWarnings(
  input: readonly SafeWarningCode[],
): SafeWarningCode[] {
  return [...new Set(input)].slice(0, 8)
}

function archivePathComparisonKey(path: string): string {
  return path.normalize('NFKC').toLocaleLowerCase('en-US')
}

async function readEntryBytes(
  entry: FileEntry,
  signal: AbortSignal,
): Promise<Uint8Array> {
  signal.throwIfAborted()
  if (
    entry.uncompressedSize > IMPORT_SAFETY_POLICY.maxSourceImageBytes ||
    !withinCompressionRatio(
      entry.uncompressedSize,
      entry.compressedSize,
      IMPORT_SAFETY_POLICY.maxCompressionRatio,
    )
  ) {
    throw new Error('mediaUnavailable')
  }
  return entry.getData(
    new BoundedUint8ArrayWriter(
      IMPORT_SAFETY_POLICY.maxSourceImageBytes,
      'mediaUnavailable',
    ),
    {
      signal,
      checkSignature: true,
      preventClose: true,
      useWebWorkers: false,
    },
  )
}

export async function sanitizeArchiveEntry(
  entry: FileEntry,
  signal: AbortSignal,
): Promise<SanitizedImage> {
  const bytes = await readEntryBytes(entry, signal)
  return sanitizeImage(bytes, signal)
}

function postIdentity(createdAt: string, post: AdapterPost): string {
  return [
    'instagram-import-post-v1',
    createdAt,
    ...post.media.map(
      (media) => `${media.kind}:${media.path.normalize('NFC')}`,
    ),
  ].join('\u0000')
}

async function normalizePost(
  post: AdapterPost,
  wrapper: string,
  entries: ReadonlyMap<string, FileEntry>,
  ambiguousEntryNames: ReadonlySet<string>,
  tokenEntries: Map<string, FileEntry>,
  now: Date,
  signal: AbortSignal,
  progress: InspectionProgress,
  progressIndex: number,
  progressTotal: number,
): Promise<{ post?: PendingReviewPost; skipped?: SkippedPost }> {
  const createdAt = normalizeSourceTimestamp(post.timestamp, now)
  const fallbackIdentity = [
    'invalid-instagram-post-v1',
    post.timestamp,
    ...post.media.map((media) => `${media.kind}:${media.path}`),
  ].join('\u0000')
  if (!createdAt) {
    const itemKey = (
      await digest(`invalid:${fallbackIdentity}`)
    ).slice(0, 32)
    signal.throwIfAborted()
    return {
      skipped: {
        itemKey,
        code: 'invalidTimestamp',
      },
    }
  }
  const identity = postIdentity(createdAt, post)
  const itemKey = (await digest(identity)).slice(0, 32)
  signal.throwIfAborted()
  const prepared = prepareCaption(post.caption)
  const warnings: SafeWarningCode[] = [...prepared.warnings]

  const imageSources = post.media.filter((media) => media.kind === 'image')
  const videoCount = post.media.filter((media) => media.kind === 'video').length
  const unsupportedCount = post.media.filter(
    (media) => media.kind === 'unsupported',
  ).length
  if (imageSources.length === 0 && videoCount > 0) {
    return {
      skipped: { itemKey, createdAt, code: 'videoOnly' },
    }
  }
  if (videoCount > 0) warnings.push('videoOmitted')
  if (unsupportedCount > 0) warnings.push('unsupportedMediaOmitted')
  if (imageSources.length > 4) warnings.push('imagesOmitted')

  const media: ReviewMedia[] = []
  for (const source of imageSources) {
    if (media.length >= 4) break
    signal.throwIfAborted()
    let archivePath: string
    try {
      archivePath = resolveArchiveMediaPath(wrapper, source.path)
    } catch {
      warnings.push('mediaUnavailable')
      continue
    }
    if (ambiguousEntryNames.has(archivePathComparisonKey(archivePath))) {
      warnings.push('mediaUnavailable')
      continue
    }
    const entry = entries.get(archivePath)
    if (!entry) {
      warnings.push('mediaUnavailable')
      continue
    }
    try {
      const sanitized = await sanitizeArchiveEntry(entry, signal)
      signal.throwIfAborted()
      const token = crypto.randomUUID()
      tokenEntries.set(token, entry)
      media.push({
        token,
        kind: 'image',
        mime: sanitized.mime,
        width: sanitized.width,
        height: sanitized.height,
        selected: true,
      })
    } catch (error) {
      if (
        error instanceof Error &&
        error.message === 'browserMediaUnsupported'
      ) {
        throw error
      }
      warnings.push('mediaUnavailable')
    }
    progress('mediaPreflight', progressIndex, progressTotal)
  }

  if (!prepared.text && media.length === 0) {
    return { skipped: { itemKey, createdAt, code: 'emptyPost' } }
  }
  const needsTextOnlyConfirmation =
    prepared.text.length > 0 &&
    media.length === 0 &&
    post.media.length > 0
  if (needsTextOnlyConfirmation) {
    warnings.push('textOnlyConfirmationRequired')
  }
  return {
    post: {
      canonicalIdentityKey: itemKey,
      value: {
        itemKey,
        createdAt,
        caption: prepared.text,
        initialCaption: prepared.text,
        media,
        warnings: dedupeWarnings(warnings),
        selected: true,
        needsTextOnlyConfirmation,
        textOnlyConfirmed: false,
      },
    },
  }
}

export async function inspectArchive(
  file: File,
  now: Date,
  signal: AbortSignal,
  progress: InspectionProgress = () => undefined,
): Promise<ArchiveInspection> {
  const envelope = await inspectZipEnvelope(file)
  signal.throwIfAborted()
  const reader = new ZipReader(new BlobReader(file), {
    useWebWorkers: false,
  })
  const candidates: Candidate[] = []
  const entries = new Map<string, FileEntry>()
  const canonicalEntryNames = new Set<string>()
  const candidateCanonicalNames = new Set<string>()
  const ambiguousEntryNames = new Set<string>()
  let enumerated = 0
  let lastDirectoryProgressAt = performance.now()
  try {
    for await (const entry of reader.getEntriesGenerator()) {
      signal.throwIfAborted()
      enumerated += 1
      if (enumerated > envelope.entryCount || enumerated > IMPORT_SAFETY_POLICY.maxEntries) {
        throw new Error('archiveEntryLimit')
      }
      if (entry.encrypted) {
        throw new Error('encryptedArchiveUnsupported')
      }
      if (
        !SUPPORTED_ZIP_COMPRESSION_METHODS.has(entry.compressionMethod)
      ) {
        throw new Error('archiveCompressionUnsupported')
      }
      const progressNow = performance.now()
      if (
        enumerated % 250 === 0 ||
        progressNow - lastDirectoryProgressAt >= 50
      ) {
        progress('directory', enumerated, envelope.entryCount)
        lastDirectoryProgressAt = progressNow
      }
      if (entry.directory || !isSafeArchivePath(entry.filename)) continue
      const comparisonKey = archivePathComparisonKey(entry.filename)
      if (canonicalEntryNames.has(comparisonKey)) {
        ambiguousEntryNames.add(comparisonKey)
        const supported = classifyPostMetadataPath(entry.filename)
        if (supported || candidateCanonicalNames.has(comparisonKey)) {
          throw new Error('ambiguousArchivePath')
        }
        continue
      }
      canonicalEntryNames.add(comparisonKey)
      entries.set(entry.filename, entry)
      const classified = classifyPostMetadataPath(entry.filename)
      if (classified) {
        candidates.push({ entry, ...classified })
        candidateCanonicalNames.add(comparisonKey)
      }
    }

    if (candidates.length === 0) throw new Error('postsMetadataMissing')
    const wrappers = new Set(candidates.map((candidate) => candidate.wrapper))
    if (wrappers.size !== 1) throw new Error('ambiguousArchivePath')

    let combinedJsonBytes = 0
    const adapterPosts: AdapterPost[] = []
    const adapterSkips: PendingAdapterSkip[] = []
    let recognizedPostRecords = 0
    for (const [index, candidate] of candidates.entries()) {
      signal.throwIfAborted()
      if (
        candidate.entry.uncompressedSize >
        IMPORT_SAFETY_POLICY.maxCandidateJsonBytes
      ) {
        throw new Error('candidateMetadataLimit')
      }
      if (
        !withinCompressionRatio(
          candidate.entry.uncompressedSize,
          candidate.entry.compressedSize,
          IMPORT_SAFETY_POLICY.maxCompressionRatio,
        )
      ) {
        throw new Error('candidateMetadataLimit')
      }
      combinedJsonBytes += candidate.entry.uncompressedSize
      if (
        combinedJsonBytes >
        IMPORT_SAFETY_POLICY.maxCombinedCandidateJsonBytes
      ) {
        throw new Error('candidateMetadataLimit')
      }
      const text = new TextDecoder().decode(
        await candidate.entry.getData(
          new BoundedUint8ArrayWriter(
            IMPORT_SAFETY_POLICY.maxCandidateJsonBytes,
            'candidateMetadataLimit',
          ),
          {
            signal,
            checkSignature: true,
            preventClose: true,
            useWebWorkers: false,
          },
        ),
      )
      signal.throwIfAborted()
      let decoded: unknown
      try {
        decoded = JSON.parse(text) as unknown
      } catch {
        throw new Error('malformedPostsMetadata')
      }
      const parsed =
        candidate.variant === 'legacy'
          ? parseLegacyExport(decoded)
          : parseDetailedExport(decoded)
      signal.throwIfAborted()
      recognizedPostRecords += parsed.recognizedPostRecords
      adapterPosts.push(...parsed.posts)
      adapterSkips.push(
        ...parsed.skipped.map((item) => ({
          ...item,
          candidateIndex: index,
          variant: candidate.variant,
        })),
      )
      progress('metadata', index + 1, candidates.length)
    }
    if (recognizedPostRecords === 0) {
      throw new Error('unsupportedPostsMetadata')
    }

    const merged = mergeAdapterPosts(adapterPosts)
    signal.throwIfAborted()
    if (
      merged.posts.length +
        merged.skipped.length +
        adapterSkips.length >
      IMPORT_SAFETY_POLICY.maxPosts
    ) {
      throw new Error('postLimit')
    }
    const wrapper = candidates[0].wrapper
    const tokenEntries = new Map<string, FileEntry>()
    const pendingPosts: PendingReviewPost[] = []
    const skipped: SkippedPost[] = []
    for (const item of adapterSkips) {
      signal.throwIfAborted()
      const itemKey = (
        await digest(
          [
            'adapter-skip-v1',
            item.variant,
            item.candidateIndex,
            item.ordinal,
            item.code,
          ].join(':'),
        )
      ).slice(0, 32)
      signal.throwIfAborted()
      skipped.push({
        itemKey,
        code: item.code,
        ...(item.timestamp === undefined
          ? {}
          : {
              createdAt:
                normalizeSourceTimestamp(item.timestamp, now) ??
                undefined,
            }),
      })
    }
    for (const [groupOrdinal, item] of merged.skipped.entries()) {
      signal.throwIfAborted()
      const itemKey = (
        await digest(
          `ambiguous:v1:${groupOrdinal}:${item.timestamp}`,
        )
      ).slice(0, 32)
      signal.throwIfAborted()
      skipped.push({
        itemKey,
        createdAt:
          normalizeSourceTimestamp(item.timestamp, now) ?? undefined,
        code: 'ambiguousDuplicate',
      })
    }
    let preflightIndex = 0
    const preflightTotal = merged.posts.reduce(
      (count, post) =>
        count +
        post.media.filter((media) => media.kind === 'image').length,
      0,
    )
    for (const post of merged.posts) {
      signal.throwIfAborted()
      const postImageCount = post.media.filter(
        (media) => media.kind === 'image',
      ).length
      const progressAtPostStart = preflightIndex
      const normalized = await normalizePost(
        post,
        wrapper,
        entries,
        ambiguousEntryNames,
        tokenEntries,
        now,
        signal,
        (phase, _completed, total) => {
          preflightIndex += 1
          progress(phase, preflightIndex, total)
        },
        preflightIndex,
        preflightTotal,
      )
      const decidedImageCount = preflightIndex - progressAtPostStart
      if (decidedImageCount < postImageCount) {
        preflightIndex += postImageCount - decidedImageCount
        progress('mediaPreflight', preflightIndex, preflightTotal)
      }
      signal.throwIfAborted()
      if (normalized.post) pendingPosts.push(normalized.post)
      if (normalized.skipped) skipped.push(normalized.skipped)
    }
    const rkeys = deterministicRecordKeys(
      pendingPosts.map((post) => ({
        createdAt: post.value.createdAt,
        canonicalIdentityKey: post.canonicalIdentityKey,
      })),
    )
    const posts = pendingPosts.map((pendingPost, index): ReviewPost => {
      const rkey = rkeys[index]
      if (!rkey) throw new Error('recordKeyAllocationFailed')
      return { ...pendingPost.value, rkey }
    })
    const partitioned = partitionRecordKeyCollisions(posts)
    const collisionItemKeys = new Set(
      partitioned.skipped.map((collision) => collision.itemKey),
    )
    for (const collided of posts) {
      if (!collisionItemKeys.has(collided.itemKey)) continue
      for (const media of collided.media) {
        tokenEntries.delete(media.token)
      }
    }
    skipped.push(...partitioned.skipped)
    const finalPosts = [...partitioned.posts].sort((left, right) => {
      const createdAtOrder =
        left.createdAt < right.createdAt
          ? -1
          : left.createdAt > right.createdAt
            ? 1
            : 0
      if (createdAtOrder !== 0) return createdAtOrder
      if (left.itemKey !== right.itemKey) {
        return left.itemKey < right.itemKey ? -1 : 1
      }
      return left.rkey < right.rkey ? -1 : left.rkey > right.rkey ? 1 : 0
    })
    const fingerprint = await digest(
      [
        'instagram-import-manifest-v1',
        ...finalPosts.map(
          (post) =>
            `${post.itemKey}:${post.rkey}:${post.caption}:${post.media
              .map(
                (media) =>
                  `${media.mime}:${media.width}x${media.height}`,
              )
              .join(',')}`,
        ),
        ...skipped.map((post) => `${post.itemKey}:${post.code}`),
      ].join('\u0000'),
    )
    signal.throwIfAborted()
    return {
      reader,
      tokenEntries,
      manifest: {
        schemaVersion: 1,
        fingerprint,
        posts: finalPosts,
        skipped,
        counts: calculateReviewCounts(finalPosts, skipped.length),
      },
    }
  } catch (error) {
    await reader.close().catch(() => undefined)
    throw error
  }
}
