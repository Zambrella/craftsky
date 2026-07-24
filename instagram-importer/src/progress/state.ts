import type { SafeErrorCode } from '../privacy/safeDiagnostics'

export type ImportSessionStatus =
  | 'reviewed'
  | 'running'
  | 'paused'
  | 'complete'
  | 'rollingBack'

export type ImportItemStatus =
  | 'remaining'
  | 'running'
  | 'created'
  | 'alreadyExisting'
  | 'collision'
  | 'failed'
  | 'rolledBack'
  | 'rollbackAbsent'
  | 'rollbackConflict'
  | 'rollbackFailed'

export interface ImportSessionRow {
  readonly id: string
  readonly schemaVersion: 1
  readonly manifestFingerprint: string
  readonly did: string
  readonly status: ImportSessionStatus
  readonly createdAt: string
  readonly updatedAt: string
}

export interface ImportItemRow {
  readonly sessionId: string
  readonly itemKey: string
  readonly rkey: string
  readonly status: ImportItemStatus
  readonly atUri?: string
  readonly createdCid?: string
  readonly safeCode?: SafeErrorCode
  readonly attempts: number
}

export interface RollbackTarget {
  readonly atUri: string
  readonly rkey: string
  readonly createdCid: string
}

export type SessionPublishOutcomeStatus =
  | 'created'
  | 'alreadyExisting'
  | 'collision'

export type DurablePublishOutcome =
  | {
      readonly status: 'created'
      readonly atUri: string
      readonly createdCid: string
    }
  | {
      readonly status: 'alreadyExisting' | 'collision'
    }

export function markCreated(
  item: ImportItemRow,
  atUri: string,
  createdCid: string,
): ImportItemRow {
  if (!atUri.startsWith('at://') || !createdCid) {
    throw new Error('invalidCreateResult')
  }
  return {
    ...item,
    status: 'created',
    atUri,
    createdCid,
    safeCode: undefined,
  }
}

export function markExistingUnowned(
  item: ImportItemRow,
): ImportItemRow {
  return {
    ...item,
    status: 'alreadyExisting',
    atUri: undefined,
    createdCid: undefined,
    safeCode: undefined,
  }
}

export function markFailed(
  item: ImportItemRow,
  safeCode: SafeErrorCode,
): ImportItemRow {
  return {
    ...item,
    status: 'failed',
    atUri: undefined,
    createdCid: undefined,
    safeCode,
  }
}

function hasRollbackOwnership(item: ImportItemRow): boolean {
  return (
    (item.status === 'created' ||
      item.status === 'rollbackConflict' ||
      item.status === 'rollbackFailed') &&
    Boolean(item.atUri) &&
    Boolean(item.createdCid)
  )
}

export function applySessionPublishOutcome(
  stored: ImportItemRow,
  running: ImportItemRow,
  outcome: DurablePublishOutcome,
): ImportItemRow {
  if (outcome.status === 'created') {
    return markCreated(running, outcome.atUri, outcome.createdCid)
  }
  if (outcome.status === 'alreadyExisting') {
    return hasRollbackOwnership(stored)
      ? {
          ...running,
          status: 'created',
          atUri: stored.atUri,
          createdCid: stored.createdCid,
          safeCode: undefined,
        }
      : markExistingUnowned(running)
  }
  return hasRollbackOwnership(stored)
    ? {
        ...running,
        status: 'created',
        atUri: stored.atUri,
        createdCid: stored.createdCid,
        safeCode: 'recordCollision',
      }
    : {
        ...running,
        status: 'collision',
        atUri: undefined,
        createdCid: undefined,
        safeCode: 'recordCollision',
      }
}

export function applySessionPublishFailure(
  stored: ImportItemRow,
  running: ImportItemRow,
  safeCode: SafeErrorCode,
): ImportItemRow {
  return hasRollbackOwnership(stored)
    ? {
        ...running,
        status: 'created',
        atUri: stored.atUri,
        createdCid: stored.createdCid,
        safeCode,
      }
    : markFailed(running, safeCode)
}

export function rollbackTargets(
  items: readonly ImportItemRow[],
): RollbackTarget[] {
  return items.flatMap((item) =>
    (item.status === 'created' ||
      item.status === 'rollbackConflict' ||
      item.status === 'rollbackFailed') &&
    item.atUri &&
    item.createdCid
      ? [
          {
            atUri: item.atUri,
            rkey: item.rkey,
            createdCid: item.createdCid,
          },
        ]
      : [],
  )
}

export function rollbackOwnershipCount(
  items: readonly ImportItemRow[],
): number {
  return items.filter(
    (item) =>
      (item.status === 'created' ||
        item.status === 'rollbackConflict' ||
        item.status === 'rollbackFailed') &&
      Boolean(item.atUri) &&
      Boolean(item.createdCid),
  ).length
}

export function sessionPublishOutcomeStatus(
  item: ImportItemRow,
  remoteStatus: SessionPublishOutcomeStatus,
): SessionPublishOutcomeStatus {
  return item.status === 'created' &&
    remoteStatus === 'alreadyExisting'
    ? 'created'
    : remoteStatus
}
