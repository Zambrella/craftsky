import type { ReviewPost } from '../domain/types'
import type { RollbackTarget } from '../progress/state'
import { CRAFTSKY_POST_COLLECTION } from '../generated/craftsky-feed-post'
import {
  assertReviewPostPublishable,
  buildPostRecord,
} from './postRecord'
import { buildFacets } from '../text/facets'

export interface RepoClient {
  getRecord(input: {
    repo: string
    collection: string
    rkey: string
  }): Promise<{ data: { cid?: string; value: unknown } }>
  uploadBlob(
    blob: Blob,
    options?: { encoding?: string },
  ): Promise<{ data: { blob: unknown } }>
  createRecord(input: {
    repo: string
    collection: string
    rkey: string
    record: unknown
  }): Promise<{ data: { uri: string; cid: string } }>
  deleteRecord(input: {
    repo: string
    collection: string
    rkey: string
    swapRecord: string
  }): Promise<unknown>
}

export interface SanitizedMedia {
  readonly buffer: ArrayBuffer
  readonly mime: string
  readonly width: number
  readonly height: number
}

export type PublishOutcome =
  | {
      readonly status: 'created'
      readonly atUri: string
      readonly createdCid: string
    }
  | { readonly status: 'alreadyExisting' | 'collision' }

interface ProtocolError extends Error {
  readonly status?: number
  readonly error?: string
}

function protocolError(error: unknown, code: string): boolean {
  return (
    error instanceof Error &&
    (error as ProtocolError).error === code
  )
}

const BASE32_ALPHABET = 'abcdefghijklmnopqrstuvwxyz234567'

function encodeBase32(bytes: Uint8Array): string {
  let accumulator = 0
  let availableBits = 0
  let encoded = ''
  for (const byte of bytes) {
    accumulator = (accumulator << 8) | byte
    availableBits += 8
    while (availableBits >= 5) {
      availableBits -= 5
      encoded += BASE32_ALPHABET[(accumulator >>> availableBits) & 31]
    }
  }
  if (availableBits > 0) {
    encoded += BASE32_ALPHABET[(accumulator << (5 - availableBits)) & 31]
  }
  return encoded
}

async function rawBlobCid(buffer: ArrayBuffer): Promise<string> {
  const hash = new Uint8Array(
    await crypto.subtle.digest('SHA-256', buffer),
  )
  const cid = new Uint8Array(4 + hash.byteLength)
  cid.set([0x01, 0x55, 0x12, 0x20])
  cid.set(hash, 4)
  return `b${encodeBase32(cid)}`
}

function jsonRepresentation(value: object): unknown {
  const toJSON = (value as { toJSON?: unknown }).toJSON
  if (typeof toJSON !== 'function') return value
  try {
    return toJSON.call(value)
  } catch {
    return value
  }
}

function isLink(value: unknown): value is { readonly $link: string } {
  return (
    typeof value === 'object' &&
    value !== null &&
    !Array.isArray(value) &&
    Object.keys(value).length === 1 &&
    typeof (value as Record<string, unknown>).$link === 'string'
  )
}

function isExactLexValue(actual: unknown, expected: unknown): boolean {
  if (Object.is(actual, expected)) return true
  if (
    typeof actual !== 'object' ||
    actual === null ||
    typeof expected !== 'object' ||
    expected === null
  ) {
    return false
  }

  const representedActual = jsonRepresentation(actual)
  if (representedActual !== actual) {
    return isExactLexValue(representedActual, expected)
  }

  if (isLink(expected)) {
    if (isLink(actual)) return actual.$link === expected.$link
    const toString = (actual as { toString?: unknown }).toString
    if (
      typeof toString !== 'function' ||
      toString === Object.prototype.toString
    ) {
      return false
    }
    try {
      return toString.call(actual) === expected.$link
    } catch {
      return false
    }
  }
  if (Array.isArray(actual) || Array.isArray(expected)) {
    return (
      Array.isArray(actual) &&
      Array.isArray(expected) &&
      actual.length === expected.length &&
      actual.every((value, index) =>
        isExactLexValue(value, expected[index]),
      )
    )
  }

  const actualRecord = actual as Record<string, unknown>
  const expectedRecord = expected as Record<string, unknown>
  const actualKeys = Object.keys(actualRecord).sort()
  const expectedKeys = Object.keys(expectedRecord).sort()
  return (
    actualKeys.length === expectedKeys.length &&
    actualKeys.every((key, index) => key === expectedKeys[index]) &&
    expectedKeys.every((key) =>
      isExactLexValue(actualRecord[key], expectedRecord[key]),
    )
  )
}

function isMatchingImportedRecord(
  value: unknown,
  expected: unknown,
): boolean {
  return isExactLexValue(value, expected)
}

function representedRecord(
  value: unknown,
): Record<string, unknown> | undefined {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return undefined
  }
  const represented = jsonRepresentation(value)
  return typeof represented === 'object' &&
    represented !== null &&
    !Array.isArray(represented)
    ? (represented as Record<string, unknown>)
    : undefined
}

function hasExactKeys(
  value: Record<string, unknown>,
  expectedKeys: readonly string[],
): boolean {
  const keys = Object.keys(value).sort()
  const expected = [...expectedKeys].sort()
  return (
    keys.length === expected.length &&
    keys.every((key, index) => key === expected[index])
  )
}

function hasMatchingImportedRecordShape(
  value: unknown,
  post: ReviewPost,
): boolean {
  const record = representedRecord(value)
  if (!record) return false
  const expectedFacets = buildFacets(post.caption)
  const expectedMedia = post.media.filter((media) => media.selected)
  const expectedKeys = [
    '$type',
    'text',
    'createdAt',
    'externalImport',
    ...(expectedFacets.length > 0 ? ['facets'] : []),
    ...(expectedMedia.length > 0 ? ['images'] : []),
  ]
  if (
    !hasExactKeys(record, expectedKeys) ||
    record.$type !== CRAFTSKY_POST_COLLECTION ||
    record.text !== post.caption ||
    record.createdAt !== post.createdAt ||
    !isExactLexValue(record.externalImport, { source: 'instagram' }) ||
    (expectedFacets.length > 0 &&
      !isExactLexValue(record.facets, expectedFacets))
  ) {
    return false
  }
  if (expectedMedia.length === 0) return true
  if (
    !Array.isArray(record.images) ||
    record.images.length !== expectedMedia.length
  ) {
    return false
  }
  return record.images.every((value, index) => {
    const image = representedRecord(value)
    const blob = representedRecord(image?.image)
    const media = expectedMedia[index]
    return (
      image !== undefined &&
      blob !== undefined &&
      media !== undefined &&
      hasExactKeys(image, ['image', 'alt', 'aspectRatio']) &&
      hasExactKeys(blob, ['$type', 'ref', 'mimeType', 'size']) &&
      image.alt === '' &&
      isExactLexValue(image.aspectRatio, {
        width: media.width,
        height: media.height,
      }) &&
      blob.$type === 'blob' &&
      blob.mimeType === media.mime &&
      typeof blob.size === 'number' &&
      Number.isInteger(blob.size) &&
      blob.size >= 0 &&
      blob.ref !== undefined
    )
  })
}

async function sanitizeConfirmedMedia(
  media: ReviewPost['media'][number],
  sanitize: (token: string) => Promise<SanitizedMedia>,
): Promise<SanitizedMedia> {
  const sanitized = await sanitize(media.token)
  if (
    sanitized.mime !== media.mime ||
    sanitized.width !== media.width ||
    sanitized.height !== media.height
  ) {
    throw new Error('mediaShapeChanged')
  }
  return sanitized
}

async function expectedExistingRecord(
  post: ReviewPost,
  sanitize: (token: string) => Promise<SanitizedMedia>,
): Promise<unknown> {
  const uploaded = []
  for (const media of post.media.filter((item) => item.selected)) {
    const sanitized = await sanitizeConfirmedMedia(media, sanitize)
    const bytes = sanitized.buffer
    uploaded.push({
      token: media.token,
      blob: {
        $type: 'blob',
        ref: { $link: await rawBlobCid(bytes) },
        mimeType: sanitized.mime,
        size: bytes.byteLength,
      },
    })
  }
  return buildPostRecord(post, uploaded)
}

export async function publishPost(input: {
  readonly repo: RepoClient
  readonly did: string
  readonly post: ReviewPost
  readonly sanitize: (token: string) => Promise<SanitizedMedia>
  readonly ownedCreatedCid?: string
}): Promise<PublishOutcome> {
  assertReviewPostPublishable(input.post)
  let existing:
    | { data: { cid?: string; value: unknown } }
    | undefined
  try {
    existing = await input.repo.getRecord({
      repo: input.did,
      collection: CRAFTSKY_POST_COLLECTION,
      rkey: input.post.rkey,
    })
  } catch (error) {
    if (!protocolError(error, 'RecordNotFound')) throw error
  }
  if (existing) {
    if (input.ownedCreatedCid) {
      return existing.data.cid === input.ownedCreatedCid
        ? { status: 'alreadyExisting' }
        : { status: 'collision' }
    }
    if (
      !hasMatchingImportedRecordShape(
        existing.data.value,
        input.post,
      )
    ) {
      return { status: 'collision' }
    }
    const expected = await expectedExistingRecord(
      input.post,
      input.sanitize,
    )
    return isMatchingImportedRecord(existing.data.value, expected)
      ? { status: 'alreadyExisting' }
      : { status: 'collision' }
  }

  const uploaded = []
  for (const media of input.post.media.filter((item) => item.selected)) {
    const sanitized = await sanitizeConfirmedMedia(
      media,
      input.sanitize,
    )
    const blob = new Blob([sanitized.buffer], { type: sanitized.mime })
    const response = await input.repo.uploadBlob(blob, {
      encoding: sanitized.mime,
    })
    uploaded.push({ token: media.token, blob: response.data.blob })
  }
  const record = buildPostRecord(input.post, uploaded)
  const created = await input.repo.createRecord({
    repo: input.did,
    collection: CRAFTSKY_POST_COLLECTION,
    rkey: input.post.rkey,
    record,
  })
  if (!created.data.uri || !created.data.cid) {
    throw new Error('ambiguousCreateResult')
  }
  return {
    status: 'created',
    atUri: created.data.uri,
    createdCid: created.data.cid,
  }
}

export interface RollbackOutcome {
  readonly rkey: string
  readonly status: 'rolledBack' | 'absent' | 'conflict' | 'failed'
}

function isExactOwnedTarget(
  target: RollbackTarget,
  did: string,
): boolean {
  const escapedDid = did.replace(/[.*+?^${}()|[\]\\]/gu, '\\$&')
  const match = target.atUri.match(
    new RegExp(
      `^at://${escapedDid}/${CRAFTSKY_POST_COLLECTION}/([a-zA-Z0-9._~:-]+)$`,
      'u',
    ),
  )
  return match?.[1] === target.rkey && target.createdCid.length > 0
}

export async function rollbackOwnedRecords(
  repo: RepoClient,
  did: string,
  targets: readonly RollbackTarget[],
): Promise<RollbackOutcome[]> {
  const outcomes: RollbackOutcome[] = []
  for (const target of targets) {
    if (!isExactOwnedTarget(target, did)) {
      outcomes.push({ rkey: target.rkey, status: 'failed' })
      continue
    }
    try {
      await repo.deleteRecord({
        repo: did,
        collection: CRAFTSKY_POST_COLLECTION,
        rkey: target.rkey,
        swapRecord: target.createdCid,
      })
      outcomes.push({ rkey: target.rkey, status: 'rolledBack' })
    } catch (error) {
      const status = protocolError(error, 'InvalidSwap')
        ? 'conflict'
        : protocolError(error, 'RecordNotFound')
          ? 'absent'
          : 'failed'
      outcomes.push({ rkey: target.rkey, status })
    }
  }
  return outcomes
}
