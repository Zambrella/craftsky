import { describe, expect, it, vi } from 'vitest'

import type { ReviewPost } from '../domain/types'
import type { RepoClient } from '../pds/publisher'
import { publishStoredItem } from './publishStoredItem'
import type { ImportItemRow } from './state'

const post: ReviewPost = {
  itemKey: 'item',
  rkey: '3kexampletid2',
  createdAt: '2024-01-02T03:04:05.000Z',
  caption: 'Synthetic caption',
  initialCaption: 'Synthetic caption',
  media: [
    {
      token: 'media',
      kind: 'image',
      mime: 'image/jpeg',
      width: 10,
      height: 10,
      selected: true,
    },
  ],
  warnings: [],
  selected: true,
  needsTextOnlyConfirmation: false,
  textOnlyConfirmed: false,
}

const stored: ImportItemRow = {
  sessionId: 'session',
  itemKey: post.itemKey,
  rkey: post.rkey,
  status: 'created',
  atUri:
    'at://did:plc:test/social.craftsky.feed.post/3kexampletid2',
  createdCid: 'bafy-owned',
  attempts: 1,
}

function repo(cid: string): RepoClient {
  return {
    getRecord: vi.fn().mockResolvedValue({
      data: {
        cid,
        value: {
          text: 'No content comparison is needed for an owned CID',
        },
      },
    }),
    uploadBlob: vi.fn(),
    createRecord: vi.fn(),
    deleteRecord: vi.fn(),
  }
}

describe('production stored-item publisher (IT-018)', () => {
  it('passes the publication AbortSignal to selected-media sanitization', async () => {
    const signalController = new AbortController()
    const client: RepoClient = {
      getRecord: vi.fn().mockRejectedValue(
        Object.assign(new Error('missing'), {
          error: 'RecordNotFound',
          status: 400,
        }),
      ),
      uploadBlob: vi.fn().mockResolvedValue({
        data: {
          blob: {
            $type: 'blob',
            ref: { $link: 'bafyblob' },
            mimeType: 'image/jpeg',
            size: 1,
          },
        },
      }),
      createRecord: vi.fn().mockResolvedValue({
        data: {
          uri:
            'at://did:plc:test/social.craftsky.feed.post/3kexampletid2',
          cid: 'bafy-created',
        },
      }),
      deleteRecord: vi.fn(),
    }
    const sanitize = vi.fn(
      (_token: string, signal?: AbortSignal) => {
        if (signal !== signalController.signal) {
          throw new Error('signalMissing')
        }
        return Promise.resolve({
          buffer: new Uint8Array([1]).buffer,
          mime: 'image/jpeg' as const,
          width: 10,
          height: 10,
        })
      },
    )

    await expect(
      publishStoredItem({
        stored: {
          ...stored,
          status: 'remaining',
          atUri: undefined,
          createdCid: undefined,
        },
        repo: client,
        did: 'did:plc:test',
        post,
        sanitize,
        signal: signalController.signal,
      }),
    ).resolves.toEqual({
      status: 'created',
      atUri:
        'at://did:plc:test/social.craftsky.feed.post/3kexampletid2',
      createdCid: 'bafy-created',
    })
    expect(sanitize).toHaveBeenCalledTimes(1)
  })

  it('uses persisted ownership to skip image work for an unchanged success', async () => {
    const client = repo('bafy-owned')
    const sanitize = vi.fn()

    await expect(
      publishStoredItem({
        stored,
        repo: client,
        did: 'did:plc:test',
        post,
        sanitize,
      }),
    ).resolves.toEqual({ status: 'alreadyExisting' })

    expect(client.getRecord).toHaveBeenCalledTimes(1)
    expect(sanitize).not.toHaveBeenCalled()
    expect(client.uploadBlob).not.toHaveBeenCalled()
    expect(client.createRecord).not.toHaveBeenCalled()
  })

  it('fails closed without image work when the owned record was replaced', async () => {
    const client = repo('bafy-replacement')
    const sanitize = vi.fn()

    await expect(
      publishStoredItem({
        stored,
        repo: client,
        did: 'did:plc:test',
        post,
        sanitize,
      }),
    ).resolves.toEqual({ status: 'collision' })

    expect(sanitize).not.toHaveBeenCalled()
    expect(client.uploadBlob).not.toHaveBeenCalled()
    expect(client.createRecord).not.toHaveBeenCalled()
  })
})
