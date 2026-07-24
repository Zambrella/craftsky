import { describe, expect, it } from 'vitest'

import type { ReviewPost } from '../domain/types'
import { buildPostRecord } from './postRecord'

const post: ReviewPost = {
  itemKey: 'private-item-key',
  rkey: '3kexampletid2',
  createdAt: '2024-01-02T03:04:05.000Z',
  caption: 'See #knitting',
  initialCaption: 'See #knitting',
  media: [
    {
      token: 'private-media-token',
      kind: 'image',
      mime: 'image/jpeg',
      width: 1080,
      height: 1350,
      selected: true,
    },
  ],
  warnings: [],
  selected: true,
  needsTextOnlyConfirmation: false,
  textOnlyConfirmed: false,
}

describe('CraftSky imported post record (UT-009)', () => {
  it('contains final public content and minimal provenance only', () => {
    const record = buildPostRecord(post, [
      {
        token: 'private-media-token',
        blob: {
          $type: 'blob',
          ref: { $link: 'bafyblob' },
          mimeType: 'image/jpeg',
          size: 2,
        },
      },
    ])
    expect(record).toMatchObject({
      $type: 'social.craftsky.feed.post',
      text: 'See #knitting',
      createdAt: '2024-01-02T03:04:05.000Z',
      externalImport: { source: 'instagram' },
      images: [
        {
          image: {
            $type: 'blob',
            ref: { $link: 'bafyblob' },
            mimeType: 'image/jpeg',
            size: 2,
          },
          alt: '',
          aspectRatio: { width: 1080, height: 1350 },
        },
      ],
    })
    const publicJson = JSON.stringify(record)
    for (const forbidden of [
      'private-item-key',
      'private-media-token',
      'handle',
      'filename',
      'fingerprint',
      'instagram.com',
    ]) {
      expect(publicJson).not.toContain(forbidden)
    }
  })

  it('requires explicit confirmation before a text-only transformed post', () => {
    expect(() =>
      buildPostRecord(
        {
          ...post,
          media: [],
          needsTextOnlyConfirmation: true,
          textOnlyConfirmed: false,
        },
        [],
      ),
    ).toThrow('textOnlyConfirmationRequired')
  })
})
