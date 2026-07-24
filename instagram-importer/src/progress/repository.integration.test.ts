import { describe, expect, it, vi } from 'vitest'

import {
  rollbackOwnedRecords,
  type RepoClient,
} from '../pds/publisher'
import { ImporterDatabase } from './database'
import { ProgressRepository } from './repository'
import {
  applySessionPublishOutcome,
  rollbackTargets,
  rollbackOwnershipCount,
  sessionPublishOutcomeStatus,
  type ImportItemRow,
} from './state'

describe('content-free progress persistence (IT-005)', () => {
  it('persists allowed ownership state and clears only the selected session', async () => {
    const database = new ImporterDatabase(
      `instagram-importer-test-${crypto.randomUUID()}`,
    )
    const repository = new ProgressRepository(database)
    await repository.createSession({
      id: 'session-a',
      schemaVersion: 1,
      manifestFingerprint: 'manifest-fingerprint',
      did: 'did:plc:test',
      status: 'running',
      createdAt: '2026-07-23T12:00:00.000Z',
      updatedAt: '2026-07-23T12:00:00.000Z',
    })
    await repository.putItem({
      sessionId: 'session-a',
      itemKey: 'item',
      rkey: '3kexampletid',
      status: 'created',
      atUri:
        'at://did:plc:test/social.craftsky.feed.post/3kexampletid',
      createdCid: 'bafy-created-cid',
      attempts: 1,
    })

    const serialized = JSON.stringify(await repository.load('session-a'))
    expect(serialized).toContain('bafy-created-cid')
    for (const forbidden of [
      'PRIVATE CAPTION',
      'media/posts/photo.jpg',
      'oauth-access-token',
      'archive.zip',
    ]) {
      expect(serialized).not.toContain(forbidden)
    }

    await database.items.update(['session-a', 'item'], {
      safeCode: 'PRIVATE-CONTENT' as ImportItemRow['safeCode'],
    })
    expect(
      (await repository.load('session-a'))?.items[0]?.safeCode,
    ).toBeUndefined()

    await repository.clearSession('session-a')
    expect(await repository.load('session-a')).toBeNull()
    database.close()
    await database.delete()
  })

  it('retains rollback ownership after reload and an edited-review collision', async () => {
    const databaseName = `owned-resume-${crypto.randomUUID()}`
    const originalDatabase = new ImporterDatabase(databaseName)
    const originalRepository = new ProgressRepository(originalDatabase)
    const session = {
      id: 'session-owned',
      schemaVersion: 1 as const,
      manifestFingerprint: 'reviewed-manifest',
      did: 'did:plc:test',
      status: 'complete' as const,
      createdAt: '2026-07-23T12:00:00.000Z',
      updatedAt: '2026-07-23T12:00:00.000Z',
    }
    const owned: ImportItemRow = {
      sessionId: session.id,
      itemKey: 'stable-source-item',
      rkey: '3kexampletid',
      status: 'created',
      atUri:
        'at://did:plc:test/social.craftsky.feed.post/3kexampletid',
      createdCid: 'bafy-original-cid',
      attempts: 1,
    }
    await originalRepository.createImport(session, [owned])
    originalDatabase.close()

    const resumedDatabase = new ImporterDatabase(databaseName)
    const resumedRepository = new ProgressRepository(resumedDatabase)
    const resumed = await resumedRepository.load(session.id)
    const stored = resumed?.items[0]
    if (!stored) throw new Error('owned fixture did not reload')
    const completed = applySessionPublishOutcome(
      stored,
      {
        ...stored,
        status: 'running',
        attempts: stored.attempts + 1,
      },
      { status: 'collision' },
    )
    await resumedRepository.putItem(completed)
    resumedDatabase.close()

    const verifiedDatabase = new ImporterDatabase(databaseName)
    const verifiedRepository = new ProgressRepository(verifiedDatabase)
    const verified = await verifiedRepository.load(session.id)
    const verifiedItem = verified?.items[0]
    if (!verifiedItem) throw new Error('owned result did not persist')
    expect(verifiedItem).toMatchObject({
      status: 'created',
      atUri: owned.atUri,
      createdCid: owned.createdCid,
      safeCode: 'recordCollision',
      attempts: 2,
    })
    expect(
      sessionPublishOutcomeStatus(verifiedItem, 'collision'),
    ).toBe('collision')
    expect(rollbackOwnershipCount(verified?.items ?? [])).toBe(1)
    expect(rollbackTargets(verified?.items ?? [])).toEqual([
      {
        atUri: owned.atUri,
        rkey: owned.rkey,
        createdCid: owned.createdCid,
      },
    ])
    const deleteRecord = vi.fn().mockRejectedValue(
      Object.assign(new Error('remote record was edited'), {
        error: 'InvalidSwap',
        status: 400,
      }),
    )
    const rollback = await rollbackOwnedRecords(
      {
        getRecord: vi.fn(),
        uploadBlob: vi.fn(),
        createRecord: vi.fn(),
        deleteRecord,
      } satisfies RepoClient,
      session.did,
      rollbackTargets(verified?.items ?? []),
    )
    expect(rollback).toEqual([
      { rkey: owned.rkey, status: 'conflict' },
    ])
    expect(deleteRecord).toHaveBeenCalledWith({
      repo: session.did,
      collection: 'social.craftsky.feed.post',
      rkey: owned.rkey,
      swapRecord: owned.createdCid,
    })

    verifiedDatabase.close()
    await verifiedDatabase.delete()
  })
})
