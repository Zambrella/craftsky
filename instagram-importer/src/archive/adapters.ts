import type {
  AdapterMedia,
  AdapterParseResult,
  AdapterPost,
  AdapterSkipOutcome,
  SourceMediaKind,
} from '../domain/types'
import { canonicalizeSourceTimestamp } from '../domain/timestamps'
import { isSupportedInstagramPostMediaPath } from './paths'

type UnknownRecord = Record<string, unknown>

interface IndexedRecord {
  readonly ordinal: number
  readonly value: UnknownRecord
}

interface ParsedMedia {
  readonly media: AdapterMedia[]
  readonly timestamps: number[]
  readonly titles: string[]
  readonly acceptedReferences: number
  readonly rejectedReferences: boolean
}

const DETAILED_POST_LABELS = new Set([
  'Caption',
  'Creation time',
  'Draft',
  'Media',
  'Published',
])

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function hasOwn(record: UnknownRecord, property: string): boolean {
  return Object.prototype.hasOwnProperty.call(record, property)
}

function hasAnyOwn(
  record: UnknownRecord,
  properties: readonly string[],
): boolean {
  return properties.some((property) => hasOwn(record, property))
}

function asIndexedRecords(value: unknown): IndexedRecord[] {
  if (Array.isArray(value)) {
    return value.flatMap((item, ordinal) =>
      isRecord(item) ? [{ ordinal, value: item }] : [],
    )
  }
  return isRecord(value) ? [{ ordinal: 0, value }] : []
}

function asFiniteTimestamp(value: unknown): number | null {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) {
    return null
  }
  return canonicalizeSourceTimestamp(value)
}

function mediaKind(path: string): SourceMediaKind {
  const extension = path.split('.').pop()?.toLowerCase()
  if (
    extension === 'jpg' ||
    extension === 'jpeg' ||
    extension === 'png' ||
    extension === 'webp'
  ) {
    return 'image'
  }
  if (
    extension === 'mp4' ||
    extension === 'mov' ||
    extension === 'm4v'
  ) {
    return 'video'
  }
  return 'unsupported'
}

function parseMedia(value: unknown): ParsedMedia {
  const media: AdapterMedia[] = []
  const timestamps: number[] = []
  const titles: string[] = []
  let acceptedReferences = 0
  let rejectedReferences = false

  if (!Array.isArray(value)) {
    return {
      media,
      timestamps,
      titles,
      acceptedReferences,
      rejectedReferences,
    }
  }
  for (const item of value) {
    if (!isRecord(item)) {
      rejectedReferences = true
      continue
    }
    const timestamp = asFiniteTimestamp(item.creation_timestamp)
    if (timestamp !== null) timestamps.push(timestamp)
    if (
      typeof item.uri !== 'string' ||
      !isSupportedInstagramPostMediaPath(item.uri)
    ) {
      rejectedReferences = true
      continue
    }
    acceptedReferences += 1
    media.push({ path: item.uri, kind: mediaKind(item.uri) })
    if (typeof item.title === 'string' && item.title) {
      titles.push(item.title)
    }
  }
  return {
    media,
    timestamps,
    titles,
    acceptedReferences,
    rejectedReferences,
  }
}

function combineMedia(values: readonly unknown[]): ParsedMedia {
  const media: AdapterMedia[] = []
  const timestamps: number[] = []
  const titles: string[] = []
  let acceptedReferences = 0
  let rejectedReferences = false
  for (const value of values) {
    const parsed = parseMedia(value)
    media.push(...parsed.media)
    timestamps.push(...parsed.timestamps)
    titles.push(...parsed.titles)
    acceptedReferences += parsed.acceptedReferences
    rejectedReferences ||= parsed.rejectedReferences
  }
  return {
    media,
    timestamps,
    titles,
    acceptedReferences,
    rejectedReferences,
  }
}

function mediaTimestamps(values: readonly unknown[]): number[] {
  const timestamps: number[] = []
  for (const value of values) {
    if (!Array.isArray(value)) continue
    for (const item of value) {
      if (!isRecord(item)) continue
      const timestamp = asFiniteTimestamp(item.creation_timestamp)
      if (timestamp !== null) timestamps.push(timestamp)
    }
  }
  return timestamps
}

function firstString(values: readonly unknown[]): string {
  for (const value of values) {
    if (typeof value === 'string' && value.length > 0) return value
  }
  return ''
}

function earliestTimestamp(
  ...candidateGroups: readonly (readonly number[])[]
): number | undefined {
  for (const candidates of candidateGroups) {
    if (candidates.length > 0) return Math.min(...candidates)
  }
  return undefined
}

function skip(
  code: AdapterSkipOutcome['code'],
  ordinal: number,
  timestamp?: number,
): AdapterSkipOutcome {
  return timestamp === undefined
    ? { code, ordinal }
    : { code, ordinal, timestamp }
}

export function parseLegacyExport(value: unknown): AdapterParseResult {
  const posts: AdapterPost[] = []
  const skipped: AdapterSkipOutcome[] = []
  let recognizedPostRecords = 0

  for (const item of asIndexedRecords(value)) {
    const raw = item.value
    if (!hasAnyOwn(raw, ['creation_timestamp', 'media', 'title'])) {
      continue
    }
    recognizedPostRecords += 1
    const wrapperTimestamp = asFiniteTimestamp(raw.creation_timestamp)
    if (!Array.isArray(raw.media)) {
      skipped.push(
        skip(
          'unsupportedPost',
          item.ordinal,
          wrapperTimestamp ?? undefined,
        ),
      )
      continue
    }

    const parsedMedia = parseMedia(raw.media)
    const sourceTimestamp = earliestTimestamp(
      parsedMedia.timestamps,
      wrapperTimestamp === null ? [] : [wrapperTimestamp],
    )
    if (
      parsedMedia.acceptedReferences === 0 ||
      parsedMedia.rejectedReferences
    ) {
      skipped.push(
        skip('unsupportedPost', item.ordinal, sourceTimestamp),
      )
      continue
    }
    if (sourceTimestamp === undefined) {
      skipped.push(skip('invalidTimestamp', item.ordinal))
      continue
    }
    posts.push({
      variant: 'legacy',
      timestamp: sourceTimestamp,
      caption: firstString([raw.title, ...parsedMedia.titles]),
      media: parsedMedia.media,
    })
  }
  return { posts, skipped, recognizedPostRecords }
}

function directLabelEntries(value: unknown): UnknownRecord[] {
  if (!Array.isArray(value)) return []
  return value.filter(
    (entry): entry is UnknownRecord =>
      isRecord(entry) && typeof entry.label === 'string',
  )
}

function nestedPostLabelGroups(value: unknown): UnknownRecord[][] {
  if (!Array.isArray(value)) return []
  const result: UnknownRecord[][] = []
  for (const container of value) {
    if (
      !isRecord(container) ||
      !Array.isArray(container.dict) ||
      (hasOwn(container, 'label') && container.label !== 'Metadata')
    ) {
      continue
    }
    for (const candidate of container.dict) {
      if (!isRecord(candidate) || !Array.isArray(candidate.dict)) {
        continue
      }
      const labels = directLabelEntries(candidate.dict)
      if (labels.length > 0) result.push(labels)
    }
  }
  return result
}

function truthValue(value: unknown): boolean | null {
  if (typeof value === 'boolean') return value
  if (typeof value !== 'string') return null
  const normalized = value.trim().toLowerCase()
  if (normalized === 'true') return true
  if (normalized === 'false') return false
  return null
}

function labelValue(
  labels: readonly UnknownRecord[],
  label: string,
): unknown {
  return labels.find((entry) => entry.label === label)?.value
}

export function parseDetailedExport(value: unknown): AdapterParseResult {
  const posts: AdapterPost[] = []
  const skipped: AdapterSkipOutcome[] = []
  let recognizedPostRecords = 0

  for (const item of asIndexedRecords(value)) {
    const raw = item.value
    if (!hasAnyOwn(raw, ['fbid', 'label_values', 'media', 'timestamp'])) {
      continue
    }
    recognizedPostRecords += 1
    const rootTimestamp = asFiniteTimestamp(raw.timestamp)
    if (!Array.isArray(raw.label_values)) {
      skipped.push(
        skip(
          'unsupportedPost',
          item.ordinal,
          rootTimestamp ?? undefined,
        ),
      )
      continue
    }

    const directLabels = directLabelEntries(raw.label_values)
    const nestedLabelGroups = nestedPostLabelGroups(raw.label_values)
    const supportedShape =
      directLabels.some(
        (entry) =>
          typeof entry.label === 'string' &&
          DETAILED_POST_LABELS.has(entry.label),
      ) || nestedLabelGroups.length > 0
    if (!supportedShape) {
      skipped.push(
        skip(
          'unsupportedPost',
          item.ordinal,
          rootTimestamp ?? undefined,
        ),
      )
      continue
    }

    const exactPostLabels = [
      ...directLabels,
      ...nestedLabelGroups.flat(),
    ]
    const nestedMediaLabels = nestedLabelGroups
      .flatMap((labels) =>
        labels.filter((entry) => entry.label === 'Media'),
      )
      .reverse()
    const chosenMediaLabels =
      nestedMediaLabels.length > 0
        ? nestedMediaLabels
        : directLabels.filter((entry) => entry.label === 'Media')
    const labelledTimestamps = exactPostLabels
      .filter((entry) => entry.label === 'Creation time')
      .map((entry) => asFiniteTimestamp(entry.timestamp_value))
      .filter((timestamp): timestamp is number => timestamp !== null)
    const recordTimestamp = earliestTimestamp(
      mediaTimestamps(
        chosenMediaLabels.map((entry) => entry.media),
      ),
      labelledTimestamps,
      rootTimestamp === null ? [] : [rootTimestamp],
    )
    const draftValue = labelValue(exactPostLabels, 'Draft')
    const publishedValue = labelValue(exactPostLabels, 'Published')
    const draft = truthValue(draftValue)
    const published = truthValue(publishedValue)
    if (
      (draftValue !== undefined && draft !== false) ||
      (publishedValue !== undefined && published !== true)
    ) {
      skipped.push(
        skip('unpublished', item.ordinal, recordTimestamp),
      )
      continue
    }

    const parsedMedia = combineMedia(
      chosenMediaLabels.map((entry) => entry.media),
    )
    if (
      parsedMedia.acceptedReferences === 0 ||
      parsedMedia.rejectedReferences
    ) {
      skipped.push(
        skip(
          'unsupportedPost',
          item.ordinal,
          earliestTimestamp(
            parsedMedia.timestamps,
            labelledTimestamps,
            rootTimestamp === null ? [] : [rootTimestamp],
          ),
        ),
      )
      continue
    }

    const sourceTimestamp = earliestTimestamp(
      parsedMedia.timestamps,
      labelledTimestamps,
      rootTimestamp === null ? [] : [rootTimestamp],
    )
    if (sourceTimestamp === undefined) {
      skipped.push(skip('invalidTimestamp', item.ordinal))
      continue
    }

    posts.push({
      variant: 'detailed',
      timestamp: sourceTimestamp,
      caption: firstString([
        directLabels.find((entry) => entry.label === 'Caption')?.value,
      ]),
      media: parsedMedia.media,
    })
  }
  return { posts, skipped, recognizedPostRecords }
}
