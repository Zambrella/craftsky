import 'fake-indexeddb/auto'

import { describe, expect, it, vi } from 'vitest'

import type { ReviewManifest } from '../domain/types'
import type { RepoClient } from '../pds/publisher'
import { ImporterDatabase } from './database'
import { rollbackImport, runImport } from './importRunner'
import { ProgressRepository } from './repository'

const manifest: ReviewManifest = {
  schemaVersion: 1,
  fingerprint: 'manifest',
  posts: [
    {
      itemKey: 'item',
      rkey: '3kexampletid2',
      createdAt: '2024-01-02T03:04:05.000Z',
      caption: 'Caption',
      initialCaption: 'Caption',
      media: [],
      warnings: [],
      selected: true,
      needsTextOnlyConfirmation: false,
      textOnlyConfirmed: false,
    },
  ],
  skipped: [],
  counts: {
    selectedPosts: 1,
    selectedImages: 0,
    transformedPosts: 0,
    warningPosts: 0,
    skippedPosts: 0,
  },
}

describe('resumable content-free import runner (IT-018)', () => {
  it('does not recreate a durable success when the same import resumes', async () => {
    const database = new ImporterDatabase(`runner-${crypto.randomUUID()}`)
    const repository = new ProgressRepository(database)
    const createRecord = vi.fn().mockResolvedValue({
      data: {
        uri: 'at://did:plc:test/social.craftsky.feed.post/3kexampletid2',
        cid: 'bafy-created',
      },
    })
    const repo = {
      getRecord: vi.fn().mockRejectedValue(
        Object.assign(new Error('missing'), {
          error: 'RecordNotFound',
          status: 400,
        }),
      ),
      uploadBlob: vi.fn(),
      createRecord,
      deleteRecord: vi.fn(),
    } satisfies RepoClient

    const first = await runImport({
      repository,
      repo,
      did: 'did:plc:test',
      manifest,
      sanitize: vi.fn(),
    })
    expect(first.items[0]?.status).toBe('created')

    const resumed = await runImport({
      repository,
      repo,
      did: 'did:plc:test',
      manifest,
      sanitize: vi.fn(),
    })
    expect(resumed.items[0]?.status).toBe('created')
    expect(createRecord).toHaveBeenCalledTimes(1)
    expect(JSON.stringify(resumed)).not.toContain('Caption')

    database.close()
    await database.delete()
  })

  it('keeps CID ownership when a rollback request fails', async () => {
    const database = new ImporterDatabase(`rollback-${crypto.randomUUID()}`)
    const repository = new ProgressRepository(database)
    const session = {
      id: 'session',
      schemaVersion: 1 as const,
      manifestFingerprint: 'manifest',
      did: 'did:plc:test',
      status: 'complete' as const,
      createdAt: '2026-07-23T12:00:00.000Z',
      updatedAt: '2026-07-23T12:00:00.000Z',
    }
    const item = {
      sessionId: session.id,
      itemKey: 'item',
      rkey: '3kexampletid2',
      status: 'created' as const,
      atUri:
        'at://did:plc:test/social.craftsky.feed.post/3kexampletid2',
      createdCid: 'bafy-created',
      attempts: 1,
    }
    await repository.createImport(session, [item])
    const result = await rollbackImport({
      repository,
      repo: {
        getRecord: vi.fn(),
        uploadBlob: vi.fn(),
        createRecord: vi.fn(),
        deleteRecord: vi.fn().mockRejectedValue(new Error('offline')),
      },
      stored: { session, items: [item] },
    })
    expect(result.stored?.items[0]).toMatchObject({
      status: 'rollbackFailed',
      atUri: item.atUri,
      createdCid: item.createdCid,
    })
    expect(await repository.load(session.id)).not.toBeNull()
    database.close()
    await database.delete()
  })

  it('retains and retries a CID conflict instead of clearing history', async () => {
    const database = new ImporterDatabase(
      `rollback-conflict-${crypto.randomUUID()}`,
    )
    const repository = new ProgressRepository(database)
    const session = {
      id: 'session-conflict',
      schemaVersion: 1 as const,
      manifestFingerprint: 'manifest',
      did: 'did:plc:test',
      status: 'complete' as const,
      createdAt: '2026-07-23T12:00:00.000Z',
      updatedAt: '2026-07-23T12:00:00.000Z',
    }
    const item = {
      sessionId: session.id,
      itemKey: 'item',
      rkey: '3kexampletid2',
      status: 'created' as const,
      atUri:
        'at://did:plc:test/social.craftsky.feed.post/3kexampletid2',
      createdCid: 'bafy-created',
      attempts: 1,
    }
    await repository.createImport(session, [item])
    const deleteRecord = vi.fn().mockRejectedValue(
      Object.assign(new Error('changed'), {
        error: 'InvalidSwap',
        status: 400,
      }),
    )
    const repo = {
      getRecord: vi.fn(),
      uploadBlob: vi.fn(),
      createRecord: vi.fn(),
      deleteRecord,
    } satisfies RepoClient

    const first = await rollbackImport({
      repository,
      repo,
      stored: { session, items: [item] },
    })
    expect(first.stored?.items[0]?.status).toBe('rollbackConflict')

    const second = await rollbackImport({
      repository,
      repo,
      stored: first.stored!,
    })
    expect(second.stored?.items[0]).toMatchObject({
      status: 'rollbackConflict',
      atUri: item.atUri,
      createdCid: item.createdCid,
    })
    expect(deleteRecord).toHaveBeenCalledTimes(2)
    expect(await repository.load(session.id)).not.toBeNull()

    database.close()
    await database.delete()
  })

  it('persists deleted, absent, and conflicting rollback results distinctly (AT-007, FR-027)', async () => {
    const database = new ImporterDatabase(
      `rollback-outcomes-${crypto.randomUUID()}`,
    )
    const repository = new ProgressRepository(database)
    const session = {
      id: 'session-outcomes',
      schemaVersion: 1 as const,
      manifestFingerprint: 'manifest',
      did: 'did:plc:test',
      status: 'complete' as const,
      createdAt: '2026-07-23T12:00:00.000Z',
      updatedAt: '2026-07-23T12:00:00.000Z',
    }
    const items = [
      {
        sessionId: session.id,
        itemKey: 'deleted-item',
        rkey: '3kdeletedtid',
        status: 'created' as const,
        atUri:
          'at://did:plc:test/social.craftsky.feed.post/3kdeletedtid',
        createdCid: 'bafy-deleted',
        attempts: 1,
      },
      {
        sessionId: session.id,
        itemKey: 'absent-item',
        rkey: '3kabsenttid',
        status: 'created' as const,
        atUri:
          'at://did:plc:test/social.craftsky.feed.post/3kabsenttid',
        createdCid: 'bafy-absent',
        attempts: 1,
      },
      {
        sessionId: session.id,
        itemKey: 'conflict-item',
        rkey: '3kconflicttid',
        status: 'created' as const,
        atUri:
          'at://did:plc:test/social.craftsky.feed.post/3kconflicttid',
        createdCid: 'bafy-conflict',
        attempts: 1,
      },
    ]
    await repository.createImport(session, items)
    const deleteRecord = vi.fn().mockImplementation(
      ({ rkey }: { readonly rkey: string }) => {
        if (rkey === '3kdeletedtid') return Promise.resolve({})
        if (rkey === '3kabsenttid') {
          return Promise.reject(
            Object.assign(new Error('already absent'), {
              error: 'RecordNotFound',
              status: 400,
            }),
          )
        }
        return Promise.reject(
          Object.assign(new Error('changed'), {
            error: 'InvalidSwap',
            status: 400,
          }),
        )
      },
    )

    const result = await rollbackImport({
      repository,
      repo: {
        getRecord: vi.fn(),
        uploadBlob: vi.fn(),
        createRecord: vi.fn(),
        deleteRecord,
      },
      stored: { session, items },
    })

    expect(result.summary).toEqual({
      deleted: 1,
      absent: 1,
      conflicts: 1,
      failed: 0,
    })
    expect(result.stored?.items).toMatchObject([
      { itemKey: 'deleted-item', status: 'rolledBack' },
      { itemKey: 'absent-item', status: 'rollbackAbsent' },
      { itemKey: 'conflict-item', status: 'rollbackConflict' },
    ])
    const persisted = await repository.load(session.id)
    expect(
      Object.fromEntries(
        (persisted?.items ?? []).map((item) => [
          item.itemKey,
          item.status,
        ]),
      ),
    ).toEqual({
      'deleted-item': 'rolledBack',
      'absent-item': 'rollbackAbsent',
      'conflict-item': 'rollbackConflict',
    })

    if (!result.stored) throw new Error('conflict should retain session')
    const retryDelete = vi.fn().mockResolvedValue({})
    const retried = await rollbackImport({
      repository,
      repo: {
        getRecord: vi.fn(),
        uploadBlob: vi.fn(),
        createRecord: vi.fn(),
        deleteRecord: retryDelete,
      },
      stored: result.stored,
    })
    expect(retryDelete).toHaveBeenCalledTimes(1)
    expect(retryDelete).toHaveBeenCalledWith({
      repo: session.did,
      collection: 'social.craftsky.feed.post',
      rkey: '3kconflicttid',
      swapRecord: 'bafy-conflict',
    })
    expect(retried.summary).toEqual({
      deleted: 1,
      absent: 0,
      conflicts: 0,
      failed: 0,
    })
    expect(retried.stored).toBeNull()
    expect(await repository.load(session.id)).toBeNull()

    database.close()
    await database.delete()
  })
})
