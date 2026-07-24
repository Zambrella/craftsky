import { describe, expect, it } from 'vitest'

import {
  parseDetailedExport,
  parseLegacyExport,
} from './adapters'
import { mergeAdapterPosts } from './normalize'

const media = (
  uri: string,
  creationTimestamp: number,
  title = '',
) => ({
  uri,
  creation_timestamp: creationTimestamp,
  title,
  media_metadata: {},
})

describe('Instagram export adapters (UT-003)', () => {
  it('normalizes the legacy singleton and preserves source image order', () => {
    const result = parseLegacyExport({
      title: 'Caption',
      creation_timestamp: 1_700_000_003,
      media: [
        media('media/posts/one.jpg', 1_700_000_001),
        media('media/posts/two.jpg', 1_700_000_000),
        media('media/posts/clip.mp4', 1_700_000_002),
      ],
    })

    expect(result).toEqual({
      posts: [
        {
          variant: 'legacy',
          timestamp: 1_700_000_000_000,
          caption: 'Caption',
          media: [
            { path: 'media/posts/one.jpg', kind: 'image' },
            { path: 'media/posts/two.jpg', kind: 'image' },
            { path: 'media/posts/clip.mp4', kind: 'video' },
          ],
        },
      ],
      skipped: [],
      recognizedPostRecords: 1,
    })
  })

  it('accepts current date-partitioned and other post-media paths across dual representations', () => {
    const sources = [
      media(
        'media/posts/202603/synthetic-one.jpg',
        1_700_000_000,
        'Synthetic one',
      ),
      media(
        'media/other/synthetic-two.jpg',
        1_700_000_100,
        'Synthetic two',
      ),
    ]
    const legacy = parseLegacyExport(
      sources.map((source) => ({
        title: '',
        creation_timestamp: source.creation_timestamp,
        media: [source],
      })),
    )
    const detailed = parseDetailedExport(
      sources.map((source) => ({
        timestamp: source.creation_timestamp,
        fbid: 'synthetic-structural-fixture',
        media: [],
        label_values: [
          { label: 'Caption', value: source.title },
          { label: 'Media', media: [source] },
          { label: 'Draft', value: 'False' },
          { label: 'Published', value: 'True' },
        ],
      })),
    )

    expect(legacy.skipped).toEqual([])
    expect(detailed.skipped).toEqual([])
    expect(legacy.posts).toHaveLength(2)
    expect(detailed.posts).toHaveLength(2)
    const merged = mergeAdapterPosts([
      ...legacy.posts,
      ...detailed.posts,
    ])
    expect(merged.skipped).toEqual([])
    expect(merged.posts).toEqual(legacy.posts)
  })

  it('uses only the versioned detailed containers and reverses carousel media', () => {
    const nested = [
      {
        dict: [
          { label: 'x' },
          { label: 'y' },
          {
            label: 'Media',
            media: [media('media/posts/three.jpg', 30)],
          },
        ],
      },
      {
        dict: [
          { label: 'x' },
          { label: 'y' },
          {
            label: 'Media',
            media: [media('media/posts/two.jpg', 20)],
          },
        ],
      },
      {
        dict: [
          { label: 'x' },
          { label: 'y' },
          {
            label: 'Media',
            media: [media('media/posts/one.jpg', 10)],
          },
        ],
      },
    ]
    const published = {
      timestamp: 33,
      label_values: [
        { label: 'Caption', value: 'Detailed caption' },
        {
          label: 'Media',
          media: [media('media/posts/one.jpg', 10)],
        },
        { label: 'Draft', value: 'False' },
        { label: 'Published', value: 'True' },
        { label: 'Metadata', dict: nested },
      ],
    }

    expect(parseDetailedExport(published)).toEqual({
      posts: [
        {
          variant: 'detailed',
          timestamp: 10_000,
          caption: 'Detailed caption',
          media: [
            { path: 'media/posts/one.jpg', kind: 'image' },
            { path: 'media/posts/two.jpg', kind: 'image' },
            { path: 'media/posts/three.jpg', kind: 'image' },
          ],
        },
      ],
      skipped: [],
      recognizedPostRecords: 1,
    })
  })

  it('returns content-free skips for unpublished and invalid-timestamp posts', () => {
    const published = {
      timestamp: 33,
      label_values: [
        { label: 'Caption', value: 'Detailed caption' },
        {
          label: 'Media',
          media: [media('media/posts/one.jpg', 10)],
        },
        { label: 'Draft', value: 'False' },
        { label: 'Published', value: 'True' },
      ],
    }
    const draft = {
      ...published,
      label_values: published.label_values.map((entry) =>
        entry.label === 'Draft'
          ? { ...entry, value: 'True' }
          : entry,
      ),
    }
    const missingTimestamp = {
      title: 'Must not cross the adapter boundary',
      media: [{ uri: 'media/posts/one.jpg' }],
    }

    expect(parseDetailedExport(draft)).toEqual({
      posts: [],
      skipped: [
        { code: 'unpublished', ordinal: 0, timestamp: 10_000 },
      ],
      recognizedPostRecords: 1,
    })
    const invalidResult = parseLegacyExport(missingTimestamp)
    expect(invalidResult).toEqual({
      posts: [],
      skipped: [{ code: 'invalidTimestamp', ordinal: 0 }],
      recognizedPostRecords: 1,
    })
    expect(JSON.stringify(invalidResult)).not.toContain(
      'Must not cross the adapter boundary',
    )
  })

  it('fails closed for boolean and unrecognized publication-state values', () => {
    const base = {
      timestamp: 1_700_000_000,
      label_values: [
        { label: 'Caption', value: 'Must stay private' },
        {
          label: 'Media',
          media: [
            media('media/posts/one.jpg', 1_700_000_000),
          ],
        },
      ],
    }
    const booleanDraft = {
      ...base,
      label_values: [
        ...base.label_values,
        { label: 'Draft', value: true },
      ],
    }
    const unknownPublished = {
      ...base,
      label_values: [
        ...base.label_values,
        { label: 'Published', value: 'Unknown' },
      ],
    }

    for (const value of [booleanDraft, unknownPublished]) {
      expect(parseDetailedExport(value)).toEqual({
        posts: [],
        skipped: [
          {
            code: 'unpublished',
            ordinal: 0,
            timestamp: 1_700_000_000_000,
          },
        ],
        recognizedPostRecords: 1,
      })
    }

    const mediaTimestampOnlyDraft = {
      label_values: [
        { label: 'Caption', value: 'Must stay private' },
        {
          label: 'Media',
          media: [
            media('media/posts/one.jpg', 1_700_000_123),
          ],
        },
        { label: 'Draft', value: 'True' },
      ],
    }
    expect(parseDetailedExport(mediaTimestampOnlyDraft)).toEqual({
      posts: [],
      skipped: [
        {
          code: 'unpublished',
          ordinal: 0,
          timestamp: 1_700_000_123_000,
        },
      ],
      recognizedPostRecords: 1,
    })
  })

  it('accepts the observed unlabeled detailed carousel container', () => {
    const result = parseDetailedExport({
      timestamp: 1_700_000_100,
      media: [],
      fbid: 'synthetic-structural-fixture',
      label_values: [
        { label: 'Caption', value: 'Synthetic caption' },
        { label: 'Draft', value: 'False' },
        { label: 'Published', value: 'True' },
        {
          dict: [
            {
              title: 'Synthetic item',
              dict: [
                {
                  label: 'Media',
                  media: [
                    media(
                      'media/posts/synthetic.jpg',
                      1_700_000_000,
                    ),
                  ],
                },
              ],
            },
          ],
        },
      ],
    })

    expect(result.posts).toMatchObject([
      {
        variant: 'detailed',
        timestamp: 1_700_000_000_000,
        caption: 'Synthetic caption',
        media: [
          { path: 'media/posts/synthetic.jpg', kind: 'image' },
        ],
      },
    ])
    expect(result.skipped).toEqual([])
  })

  it('never promotes arbitrary nested label-value objects into posts', () => {
    const canary = 'PRIVATE-NESTED-LABEL-CANARY'
    const result = parseDetailedExport({
      timestamp: 1_700_000_000,
      label_values: [
        { label: 'Caption', value: canary },
        {
          label: 'Private messages',
          dict: [
            {
              dict: [
                {
                  label: 'Media',
                  media: [
                    media(
                      'media/posts/looks-like-a-post.jpg',
                      1_700_000_000,
                    ),
                  ],
                },
              ],
            },
          ],
        },
        {
          label: 'Metadata',
          private: {
            dict: [
              {
                label: 'Media',
                media: [
                  media(
                    'media/posts/also-private.jpg',
                    1_700_000_000,
                  ),
                ],
              },
            ],
          },
        },
      ],
    })

    expect(result).toEqual({
      posts: [],
      skipped: [
        {
          code: 'unsupportedPost',
          ordinal: 0,
          timestamp: 1_700_000_000_000,
        },
      ],
      recognizedPostRecords: 1,
    })
    expect(JSON.stringify(result)).not.toContain(canary)
    expect(JSON.stringify(result)).not.toContain('also-private')
  })

  it('never returns private or unrelated media paths as publishable media', () => {
    const canary = 'PRIVATE-PATH-CAPTION-CANARY'
    const legacy = parseLegacyExport({
      title: canary,
      media: [media('messages/private.jpg', 1_700_000_000)],
    })
    const detailed = parseDetailedExport({
      timestamp: 1_700_000_000,
      label_values: [
        { label: 'Caption', value: canary },
        {
          label: 'Media',
          media: [
            media(
              'your_instagram_activity/messages/private.jpg',
              1_700_000_000,
            ),
          ],
        },
      ],
    })
    const mixed = parseLegacyExport({
      title: canary,
      media: [
        media('media/posts/public.jpg', 1_700_000_000),
        media('messages/private.jpg', 1_700_000_000),
      ],
    })

    for (const result of [legacy, detailed, mixed]) {
      expect(result.posts).toEqual([])
      expect(result.skipped).toEqual([
        {
          code: 'unsupportedPost',
          ordinal: 0,
          timestamp: 1_700_000_000_000,
        },
      ])
      expect(JSON.stringify(result)).not.toContain(canary)
      expect(JSON.stringify(result)).not.toContain('private.jpg')
    }
  })

  it('distinguishes wholly unsupported metadata from post-shaped skips', () => {
    expect(
      parseLegacyExport({
        messages: [{ sender: 'private-canary' }],
      }),
    ).toEqual({
      posts: [],
      skipped: [],
      recognizedPostRecords: 0,
    })
    expect(
      parseDetailedExport({
        relationships: {
          followers: ['private-canary'],
        },
      }),
    ).toEqual({
      posts: [],
      skipped: [],
      recognizedPostRecords: 0,
    })
  })

  it('merges equivalent seconds and milliseconds timestamp representations', () => {
    const [legacy] = parseLegacyExport({
      title: 'Same instant',
      creation_timestamp: 1_700_000_000,
      media: [
        media('media/posts/one.jpg', 1_700_000_000),
      ],
    }).posts
    const [detailed] = parseDetailedExport({
      timestamp: 1_700_000_000_000,
      label_values: [
        { label: 'Caption', value: 'Same instant' },
        {
          label: 'Media',
          media: [
            media('media/posts/one.jpg', 1_700_000_000_000),
          ],
        },
      ],
    }).posts
    if (!legacy || !detailed) {
      throw new Error('timestamp equivalence fixtures did not parse')
    }

    expect(legacy.timestamp).toBe(1_700_000_000_000)
    expect(detailed.timestamp).toBe(1_700_000_000_000)
    expect(mergeAdapterPosts([legacy, detailed])).toEqual({
      posts: [legacy],
      skipped: [],
    })
  })

  it('orders same-timestamp posts independently of export row order', () => {
    const parsed = parseLegacyExport([
      {
        title: 'Second source row',
        media: [media('media/posts/b.jpg', 1_700_000_000)],
      },
      {
        title: 'First source row',
        media: [media('media/posts/a.jpg', 1_700_000_000)],
      },
    ]).posts

    expect(mergeAdapterPosts(parsed).posts).toEqual(
      mergeAdapterPosts([...parsed].reverse()).posts,
    )
  })

  it('merges equivalent shapes and skips a material conflict', () => {
    const [legacy] = parseLegacyExport({
      title: 'Same',
      media: [media('media/posts/one.jpg', 100)],
    }).posts
    if (!legacy) throw new Error('legacy test fixture did not parse')
    const detailed = {
      ...legacy,
      variant: 'detailed' as const,
    }

    expect(mergeAdapterPosts([legacy, detailed])).toMatchObject({
      posts: [{ caption: 'Same' }],
      skipped: [],
    })
    expect(
      mergeAdapterPosts([
        legacy,
        { ...detailed, caption: 'Materially different' },
      ]),
    ).toMatchObject({
      posts: [],
      skipped: [{ code: 'ambiguousDuplicate' }],
    })
    expect(
      mergeAdapterPosts([
        legacy,
        { ...detailed, timestamp: legacy.timestamp + 1 },
      ]),
    ).toMatchObject({
      posts: [],
      skipped: [{ code: 'ambiguousDuplicate' }],
    })
    expect(
      mergeAdapterPosts([
        {
          ...legacy,
          media: [
            { path: 'media/posts/one.jpg', kind: 'image' },
            { path: 'media/posts/two.jpg', kind: 'image' },
          ],
        },
        {
          ...detailed,
          media: [
            { path: 'media/posts/two.jpg', kind: 'image' },
            { path: 'media/posts/one.jpg', kind: 'image' },
          ],
        },
      ]),
    ).toMatchObject({
      posts: [],
      skipped: [{ code: 'ambiguousDuplicate' }],
    })
  })

  it('merges equivalent media identities across Unicode normalization forms', () => {
    const composed = {
      variant: 'legacy' as const,
      timestamp: 1_700_000_000_000,
      caption: 'Same',
      media: [
        {
          path: 'media/posts/caf\u00e9.jpg',
          kind: 'image' as const,
        },
      ],
    }
    const decomposed = {
      ...composed,
      variant: 'detailed' as const,
      media: [
        {
          path: 'media/posts/cafe\u0301.jpg',
          kind: 'image' as const,
        },
      ],
    }

    expect(mergeAdapterPosts([composed, decomposed])).toEqual({
      posts: [composed],
      skipped: [],
    })
  })
})
