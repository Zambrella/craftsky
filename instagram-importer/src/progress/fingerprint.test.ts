import { describe, expect, it } from 'vitest'

import {
  deriveManifestFingerprint,
  deriveReviewedManifestFingerprint,
} from './fingerprint'
import type { ReviewPost } from '../domain/types'

describe('versioned manifest fingerprint (UT-017)', () => {
  it('is stable for equivalent ordered item identities and changes materially', async () => {
    const first = await deriveManifestFingerprint([
      { itemKey: 'a', rkey: '3ka' },
      { itemKey: 'b', rkey: '3kb' },
    ])
    expect(
      await deriveManifestFingerprint([
        { itemKey: 'b', rkey: '3kb' },
        { itemKey: 'a', rkey: '3ka' },
      ]),
    ).toBe(first)
    expect(
      await deriveManifestFingerprint([
        { itemKey: 'a', rkey: '3ka' },
        { itemKey: 'c', rkey: '3kc' },
      ]),
    ).not.toBe(first)
  })

  it('tracks content-free selection shape without hashing captions', async () => {
    const post: ReviewPost = {
      itemKey: 'a',
      rkey: '3ka',
      createdAt: '2024-01-02T03:04:05.000Z',
      caption: 'First',
      initialCaption: 'First',
      media: [],
      warnings: [],
      selected: true,
      needsTextOnlyConfirmation: false,
      textOnlyConfirmed: false,
    }
    expect(await deriveReviewedManifestFingerprint([post])).toBe(
      await deriveReviewedManifestFingerprint([
        { ...post, caption: 'PRIVATE CAPTION MUST NOT BE HASHED' },
      ]),
    )
    const withMedia: ReviewPost = {
      ...post,
      media: [
        {
          token: 'ephemeral-token',
          kind: 'image',
          mime: 'image/jpeg',
          width: 100,
          height: 100,
          selected: true,
        },
      ],
    }
    expect(await deriveReviewedManifestFingerprint([withMedia])).not.toBe(
      await deriveReviewedManifestFingerprint([
        {
          ...withMedia,
          media: [{ ...withMedia.media[0], selected: false }],
        },
      ]),
    )
    expect(
      await deriveReviewedManifestFingerprint([
        { ...post, itemKey: 'b', rkey: '3kb' },
        withMedia,
      ]),
    ).toBe(
      await deriveReviewedManifestFingerprint([
        withMedia,
        { ...post, itemKey: 'b', rkey: '3kb' },
      ]),
    )
  })
})
