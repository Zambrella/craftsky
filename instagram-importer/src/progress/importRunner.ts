import type { ReviewManifest, ReviewPost } from '../domain/types'
import {
  rollbackOwnedRecords,
  type RepoClient,
  type RollbackOutcome,
  type SanitizedMedia,
} from '../pds/publisher'
import { publishStoredItem } from './publishStoredItem'
import type { ProgressRepository, StoredImport } from './repository'
import {
  applySessionPublishFailure,
  applySessionPublishOutcome,
  rollbackTargets,
  rollbackOwnershipCount,
  type ImportItemRow,
  type ImportSessionRow,
} from './state'

export interface ImportRunUpdate {
  readonly session: ImportSessionRow
  readonly item: ImportItemRow
  readonly completed: number
  readonly total: number
}

export interface RunImportInput {
  readonly repository: ProgressRepository
  readonly repo: RepoClient
  readonly did: string
  readonly manifest: ReviewManifest
  readonly sanitize: (
    token: string,
    signal?: AbortSignal,
  ) => Promise<SanitizedMedia>
  readonly signal?: AbortSignal
  readonly retryFailed?: boolean
  readonly onUpdate?: (update: ImportRunUpdate) => void
}

function now(): string {
  return new Date().toISOString()
}

function newSession(
  did: string,
  manifestFingerprint: string,
): ImportSessionRow {
  const timestamp = now()
  return {
    id: crypto.randomUUID(),
    schemaVersion: 1,
    manifestFingerprint,
    did,
    status: 'reviewed',
    createdAt: timestamp,
    updatedAt: timestamp,
  }
}

function newItem(sessionId: string, post: ReviewPost): ImportItemRow {
  return {
    sessionId,
    itemKey: post.itemKey,
    rkey: post.rkey,
    status: 'remaining',
    attempts: 0,
  }
}

async function createOrResume(
  input: RunImportInput,
  posts: readonly ReviewPost[],
): Promise<StoredImport> {
  const existing = await input.repository.findResumable(
    input.did,
    input.manifest.fingerprint,
  )
  if (existing) return existing
  const session = newSession(input.did, input.manifest.fingerprint)
  const items = posts.map((post) => newItem(session.id, post))
  await input.repository.createImport(session, items)
  return { session, items }
}

function completedCount(items: readonly ImportItemRow[]): number {
  return items.filter(
    (item) => item.status !== 'remaining' && item.status !== 'running',
  ).length
}

export async function runImport(
  input: RunImportInput,
): Promise<StoredImport> {
  const posts = input.manifest.posts.filter(
    (post) =>
      post.selected &&
      (!post.needsTextOnlyConfirmation || post.textOnlyConfirmed),
  )
  const postsByKey = new Map(posts.map((post) => [post.itemKey, post]))
  const stored = await createOrResume(input, posts)
  let session: ImportSessionRow = {
    ...stored.session,
    status: 'running',
    updatedAt: now(),
  }
  const items = [...stored.items]
  await input.repository.putSession(session)

  for (let index = 0; index < items.length; index += 1) {
    if (input.signal?.aborted) {
      session = { ...session, status: 'paused', updatedAt: now() }
      await input.repository.putSession(session)
      break
    }
    const current = items[index]
    const eligible =
      current.status === 'remaining' ||
      current.status === 'running' ||
      (input.retryFailed && current.status === 'failed')
    if (!eligible) continue
    const post = postsByKey.get(current.itemKey)
    if (!post) continue

    let item: ImportItemRow = {
      ...current,
      status: 'running',
      attempts: current.attempts + 1,
      safeCode: undefined,
    }
    items[index] = item
    await input.repository.putItem(item)
    try {
      const outcome = await publishStoredItem({
        stored: current,
        repo: input.repo,
        did: input.did,
        post,
        sanitize: input.sanitize,
        signal: input.signal,
      })
      item = applySessionPublishOutcome(current, item, outcome)
    } catch {
      item = applySessionPublishFailure(
        current,
        item,
        'publicationFailed',
      )
    }
    items[index] = item
    await input.repository.putItem(item)
    input.onUpdate?.({
      session,
      item,
      completed: completedCount(items),
      total: items.length,
    })
    if (input.signal?.aborted) {
      session = { ...session, status: 'paused', updatedAt: now() }
      await input.repository.putSession(session)
      break
    }
  }

  if (session.status === 'running') {
    session = { ...session, status: 'complete', updatedAt: now() }
    await input.repository.putSession(session)
  }
  return { session, items }
}

export async function rollbackImport(input: {
  readonly repository: ProgressRepository
  readonly repo: RepoClient
  readonly stored: StoredImport
}): Promise<{
  readonly stored: StoredImport | null
  readonly items: readonly ImportItemRow[]
  readonly outcomes: readonly RollbackOutcome[]
  readonly summary: {
    readonly deleted: number
    readonly absent: number
    readonly conflicts: number
    readonly failed: number
  }
}> {
  let session: ImportSessionRow = {
    ...input.stored.session,
    status: 'rollingBack',
    updatedAt: now(),
  }
  await input.repository.putSession(session)
  const outcomes = await rollbackOwnedRecords(
    input.repo,
    session.did,
    rollbackTargets(input.stored.items),
  )
  const outcomeByRkey = new Map(
    outcomes.map((outcome) => [outcome.rkey, outcome]),
  )
  const summary = outcomes.reduce(
    (counts, outcome) => ({
      deleted:
        counts.deleted + (outcome.status === 'rolledBack' ? 1 : 0),
      absent: counts.absent + (outcome.status === 'absent' ? 1 : 0),
      conflicts:
        counts.conflicts + (outcome.status === 'conflict' ? 1 : 0),
      failed: counts.failed + (outcome.status === 'failed' ? 1 : 0),
    }),
    { deleted: 0, absent: 0, conflicts: 0, failed: 0 },
  )
  const items: ImportItemRow[] = input.stored.items.map((item) => {
    const outcome = outcomeByRkey.get(item.rkey)
    if (!outcome) return item
    if (outcome.status === 'rolledBack' || outcome.status === 'absent') {
      return {
        ...item,
        status:
          outcome.status === 'rolledBack'
            ? ('rolledBack' as const)
            : ('rollbackAbsent' as const),
        atUri: undefined,
        createdCid: undefined,
        safeCode: undefined,
      }
    }
    return {
      ...item,
      status:
        outcome.status === 'conflict'
          ? ('rollbackConflict' as const)
          : ('rollbackFailed' as const),
      safeCode:
        outcome.status === 'conflict'
          ? 'rollbackConflict'
          : 'publicationFailed',
    } satisfies ImportItemRow
  })
  await input.repository.putItems(items)
  if (rollbackOwnershipCount(items) === 0) {
    await input.repository.clearSession(session.id)
    return { stored: null, items, outcomes, summary }
  }
  session = { ...session, status: 'complete', updatedAt: now() }
  await input.repository.putSession(session)
  return {
    stored: { session, items },
    items,
    outcomes,
    summary,
  }
}
