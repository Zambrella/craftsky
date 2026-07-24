import { describe, expect, it, vi } from 'vitest'
import { jsonToLex } from '@atproto/api'

import type { ReviewPost } from '../domain/types'
import {
  publishPost,
  rollbackOwnedRecords,
  type RepoClient,
} from './publisher'

const post: ReviewPost = {
  itemKey: 'item',
  rkey: '3kexampletid2',
  createdAt: '2024-01-02T03:04:05.000Z',
  caption: 'Caption',
  initialCaption: 'Caption',
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

function missingRecord(): Error & { error: string; status: number } {
  return Object.assign(new Error('missing'), {
    error: 'RecordNotFound',
    status: 400,
  })
}

function repo(overrides: Partial<RepoClient> = {}): RepoClient {
  return {
    getRecord: vi.fn().mockRejectedValue(missingRecord()),
    uploadBlob: vi.fn().mockResolvedValue({
      data: {
        blob: {
          $type: 'blob',
          ref: { $link: 'bafyblob' },
          mimeType: 'image/jpeg',
          size: 2,
        },
      },
    }),
    createRecord: vi.fn().mockResolvedValue({
      data: {
        uri: 'at://did:plc:test/social.craftsky.feed.post/3kexampletid2',
        cid: 'bafy-created',
      },
    }),
    deleteRecord: vi.fn().mockResolvedValue({ data: {} }),
    ...overrides,
  }
}

describe('PDS publisher (IT-007)', () => {
  it('uploads every confirmed image before create and returns owned CID', async () => {
    const client = repo()
    const outcome = await publishPost({
      repo: client,
      did: 'did:plc:test',
      post,
      sanitize: vi.fn().mockResolvedValue({
        buffer: new Uint8Array([0xff, 0xd8]).buffer,
        mime: 'image/jpeg',
        width: 10,
        height: 10,
      }),
    })
    expect(outcome).toEqual({
      status: 'created',
      atUri:
        'at://did:plc:test/social.craftsky.feed.post/3kexampletid2',
      createdCid: 'bafy-created',
    })
    expect(client.uploadBlob).toHaveBeenCalledTimes(1)
    expect(client.createRecord).toHaveBeenCalledTimes(1)
  })

  it('treats matching existing imports as unowned and other records as collisions', async () => {
    const existingImport = repo({
      getRecord: vi.fn().mockResolvedValue({
        data: {
          value: {
            $type: 'social.craftsky.feed.post',
            createdAt: post.createdAt,
            text: post.caption,
            images: [
              {
                image: jsonToLex({
                  $type: 'blob',
                  ref: {
                    $link:
                      'bafkreidrky5nqadbib7n5hdpgfudmkcl2nyquuqmlj4swxw2ds3qg2iicu',
                  },
                  mimeType: 'image/jpeg',
                  size: 2,
                }),
                alt: '',
                aspectRatio: { width: 10, height: 10 },
              },
            ],
            externalImport: { source: 'instagram' },
          },
        },
      }),
    })
    await expect(
      publishPost({
        repo: existingImport,
        did: 'did:plc:test',
        post,
        sanitize: vi.fn().mockResolvedValue({
          buffer: new Uint8Array([0xff, 0xd8]).buffer,
          mime: 'image/jpeg',
          width: 10,
          height: 10,
        }),
      }),
    ).resolves.toEqual({ status: 'alreadyExisting' })
    expect(existingImport.uploadBlob).not.toHaveBeenCalled()
    expect(existingImport.createRecord).not.toHaveBeenCalled()

    const collision = repo({
      getRecord: vi.fn().mockResolvedValue({
        data: {
          value: {
            $type: 'social.craftsky.feed.post',
            createdAt: post.createdAt,
          },
        },
      }),
    })
    await expect(
      publishPost({
        repo: collision,
        did: 'did:plc:test',
        post,
        sanitize: vi.fn(),
      }),
    ).resolves.toEqual({ status: 'collision' })

    const sameTimestampDifferentContent = repo({
      getRecord: vi.fn().mockResolvedValue({
        data: {
          value: {
            $type: 'social.craftsky.feed.post',
            createdAt: post.createdAt,
            text: 'Different post at a colliding deterministic key',
            externalImport: { source: 'instagram' },
          },
        },
      }),
    })
    await expect(
      publishPost({
        repo: sameTimestampDifferentContent,
        did: 'did:plc:test',
        post,
        sanitize: vi.fn(),
      }),
    ).resolves.toEqual({ status: 'collision' })
  })

  it('treats an existing record with different image bytes as a collision', async () => {
    const sameShapeDifferentBlob = repo({
      getRecord: vi.fn().mockResolvedValue({
        data: {
          value: {
            $type: 'social.craftsky.feed.post',
            createdAt: post.createdAt,
            text: post.caption,
            images: [
              {
                image: {
                  $type: 'blob',
                  ref: { $link: 'bafkreidifferentblobidentity' },
                  mimeType: 'image/jpeg',
                  size: 2,
                },
                alt: '',
                aspectRatio: { width: 10, height: 10 },
              },
            ],
            externalImport: { source: 'instagram' },
          },
        },
      }),
    })

    await expect(
      publishPost({
        repo: sameShapeDifferentBlob,
        did: 'did:plc:test',
        post,
        sanitize: vi.fn().mockResolvedValue({
          buffer: new Uint8Array([0xff, 0xd8]).buffer,
          mime: 'image/jpeg',
          width: 10,
          height: 10,
        }),
      }),
    ).resolves.toEqual({ status: 'collision' })
    expect(sameShapeDifferentBlob.uploadBlob).not.toHaveBeenCalled()
    expect(sameShapeDifferentBlob.createRecord).not.toHaveBeenCalled()
  })

  it('checks a session-owned CID before reprocessing media on resume', async () => {
    const sanitize = vi.fn()
    const unchangedOwned = repo({
      getRecord: vi.fn().mockResolvedValue({
        data: {
          cid: 'bafy-owned',
          value: {
            text: 'The value does not need a content comparison',
          },
        },
      }),
    })

    await expect(
      publishPost({
        repo: unchangedOwned,
        did: 'did:plc:test',
        post,
        sanitize,
        ownedCreatedCid: 'bafy-owned',
      }),
    ).resolves.toEqual({ status: 'alreadyExisting' })
    expect(sanitize).not.toHaveBeenCalled()
    expect(unchangedOwned.uploadBlob).not.toHaveBeenCalled()
    expect(unchangedOwned.createRecord).not.toHaveBeenCalled()

    const replacedOwned = repo({
      getRecord: vi.fn().mockResolvedValue({
        data: {
          cid: 'bafy-replacement',
          value: {
            $type: 'social.craftsky.feed.post',
            text: post.caption,
          },
        },
      }),
    })
    await expect(
      publishPost({
        repo: replacedOwned,
        did: 'did:plc:test',
        post,
        sanitize,
        ownedCreatedCid: 'bafy-owned',
      }),
    ).resolves.toEqual({ status: 'collision' })
    expect(sanitize).not.toHaveBeenCalled()
    expect(replacedOwned.uploadBlob).not.toHaveBeenCalled()
    expect(replacedOwned.createRecord).not.toHaveBeenCalled()
  })

  it('treats unexpected material fields on an existing record as a collision', async () => {
    const existingWithReply = repo({
      getRecord: vi.fn().mockResolvedValue({
        data: {
          value: {
            $type: 'social.craftsky.feed.post',
            createdAt: post.createdAt,
            text: post.caption,
            images: [
              {
                image: {
                  $type: 'blob',
                  ref: {
                    $link:
                      'bafkreidrky5nqadbib7n5hdpgfudmkcl2nyquuqmlj4swxw2ds3qg2iicu',
                  },
                  mimeType: 'image/jpeg',
                  size: 2,
                },
                alt: '',
                aspectRatio: { width: 10, height: 10 },
              },
            ],
            externalImport: { source: 'instagram' },
            reply: {
              root: { uri: 'at://did:plc:other/post/root' },
              parent: { uri: 'at://did:plc:other/post/parent' },
            },
          },
        },
      }),
    })

    await expect(
      publishPost({
        repo: existingWithReply,
        did: 'did:plc:test',
        post,
        sanitize: vi.fn().mockResolvedValue({
          buffer: new Uint8Array([0xff, 0xd8]).buffer,
          mime: 'image/jpeg',
          width: 10,
          height: 10,
        }),
      }),
    ).resolves.toEqual({ status: 'collision' })
    expect(existingWithReply.uploadBlob).not.toHaveBeenCalled()
    expect(existingWithReply.createRecord).not.toHaveBeenCalled()
  })

  it('uses the created CID as rollback precondition and reports replacement conflict', async () => {
    const deleteRecord = vi
      .fn()
      .mockRejectedValueOnce(
        Object.assign(new Error('swap failed'), {
          error: 'InvalidSwap',
          status: 400,
        }),
      )
    const outcomes = await rollbackOwnedRecords(
      repo({ deleteRecord }),
      'did:plc:test',
      [
        {
          atUri:
            'at://did:plc:test/social.craftsky.feed.post/3kexampletid',
          rkey: '3kexampletid',
          createdCid: 'bafy-created',
        },
      ],
    )
    expect(deleteRecord).toHaveBeenCalledWith({
      repo: 'did:plc:test',
      collection: 'social.craftsky.feed.post',
      rkey: '3kexampletid',
      swapRecord: 'bafy-created',
    })
    expect(outcomes).toEqual([
      { rkey: '3kexampletid', status: 'conflict' },
    ])
  })

  it('fails the post if publish-time media no longer matches review', async () => {
    const client = repo()
    await expect(
      publishPost({
        repo: client,
        did: 'did:plc:test',
        post,
        sanitize: vi.fn().mockResolvedValue({
          buffer: new Uint8Array([0xff, 0xd8]).buffer,
          mime: 'image/jpeg',
          width: 9,
          height: 10,
        }),
      }),
    ).rejects.toThrow('mediaShapeChanged')
    expect(client.uploadBlob).not.toHaveBeenCalled()
    expect(client.createRecord).not.toHaveBeenCalled()
  })
})
