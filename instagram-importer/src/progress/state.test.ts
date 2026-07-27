import { describe, expect, it } from 'vitest'

import {
  applySessionPublishFailure,
  applySessionPublishOutcome,
  markCreated,
  markExistingUnowned,
  rollbackTargets,
  rollbackOwnershipCount,
  sessionPublishOutcomeStatus,
  type ImportItemRow,
} from './state'

const item = (): ImportItemRow => ({
  sessionId: 'session',
  itemKey: 'item',
  rkey: '3kexampletid',
  status: 'running',
  attempts: 1,
})

describe('progress ownership transitions (UT-010)', () => {
  it('claims rollback ownership only with a durable create URI and CID', () => {
    const created = markCreated(
      item(),
      'at://did:plc:example/social.craftsky.feed.post/3kexampletid',
      'bafy-created-cid',
    )
    expect(rollbackTargets([created])).toEqual([
      {
        atUri:
          'at://did:plc:example/social.craftsky.feed.post/3kexampletid',
        rkey: '3kexampletid',
        createdCid: 'bafy-created-cid',
      },
    ])
  })

  it('never reclaims an existing record after an ambiguous create result', () => {
    const existing = markExistingUnowned(item())
    expect(existing.status).toBe('alreadyExisting')
    expect(rollbackTargets([existing])).toEqual([])
  })

  it('retains a failed rollback target so deletion can be retried safely', () => {
    const created = markCreated(
      item(),
      'at://did:plc:example/social.craftsky.feed.post/3kexampletid',
      'bafy-created-cid',
    )
    expect(
      rollbackTargets([{ ...created, status: 'rollbackFailed' }]),
    ).toHaveLength(1)
    expect(
      rollbackTargets([{ ...created, status: 'rollbackConflict' }]),
    ).toHaveLength(1)
    expect(
      rollbackTargets([
        {
          ...created,
          status: 'rollbackAbsent',
          atUri: undefined,
          createdCid: undefined,
        },
      ]),
    ).toEqual([])
    expect(
      rollbackOwnershipCount([
        { ...created, status: 'rollbackConflict' },
      ]),
    ).toBe(1)
  })

  it('keeps a resumed owned record visible as rollback-owned', () => {
    const created = markCreated(
      item(),
      'at://did:plc:example/social.craftsky.feed.post/3kexampletid',
      'bafy-created-cid',
    )
    expect(
      sessionPublishOutcomeStatus(created, 'alreadyExisting'),
    ).toBe('created')
    expect(
      sessionPublishOutcomeStatus(created, 'collision'),
    ).toBe('collision')
    expect(
      sessionPublishOutcomeStatus(
        markExistingUnowned(item()),
        'alreadyExisting',
      ),
    ).toBe('alreadyExisting')
  })

  it('preserves prior ownership when a resumed remote record has changed', () => {
    const created = markCreated(
      item(),
      'at://did:plc:example/social.craftsky.feed.post/3kexampletid',
      'bafy-original-cid',
    )
    const completed = applySessionPublishOutcome(
      created,
      {
        ...created,
        status: 'running',
        attempts: created.attempts + 1,
      },
      { status: 'collision' },
    )

    expect(completed).toMatchObject({
      status: 'created',
      atUri: created.atUri,
      createdCid: created.createdCid,
      safeCode: 'recordCollision',
      attempts: 2,
    })
    expect(
      sessionPublishOutcomeStatus(completed, 'collision'),
    ).toBe('collision')
    expect(rollbackTargets([completed])).toEqual([
      {
        atUri: created.atUri,
        rkey: created.rkey,
        createdCid: created.createdCid,
      },
    ])
  })

  it('keeps exact ownership and replaces it only after a new create', () => {
    const created = markCreated(
      item(),
      'at://did:plc:example/social.craftsky.feed.post/3kexampletid',
      'bafy-original-cid',
    )
    const running = {
      ...created,
      status: 'running' as const,
      attempts: 2,
    }

    expect(
      applySessionPublishOutcome(created, running, {
        status: 'alreadyExisting',
      }),
    ).toMatchObject({
      status: 'created',
      atUri: created.atUri,
      createdCid: created.createdCid,
    })
    expect(
      applySessionPublishOutcome(created, running, {
        status: 'created',
        atUri:
          'at://did:plc:example/social.craftsky.feed.post/3kexampletid',
        createdCid: 'bafy-recreated-cid',
      }),
    ).toMatchObject({
      status: 'created',
      createdCid: 'bafy-recreated-cid',
    })
  })

  it('preserves rollback ownership across a resumed publication failure', () => {
    const created = markCreated(
      item(),
      'at://did:plc:example/social.craftsky.feed.post/3kexampletid',
      'bafy-original-cid',
    )
    const failed = applySessionPublishFailure(
      created,
      { ...created, status: 'running', attempts: 2 },
      'publicationFailed',
    )

    expect(failed).toMatchObject({
      status: 'created',
      atUri: created.atUri,
      createdCid: created.createdCid,
      safeCode: 'publicationFailed',
    })
    expect(rollbackTargets([failed])).toHaveLength(1)
  })
})
