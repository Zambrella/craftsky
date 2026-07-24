import {
  BlobWriter,
  TextReader,
  Uint8ArrayReader,
  ZipWriter,
} from '@zip.js/zip.js'
import { afterEach, describe, expect, it, vi } from 'vitest'

import * as imageSanitizer from '../media/sanitize'
import { inspectArchive } from './inspectArchive'

interface SyntheticEntry {
  readonly path: string
  readonly text?: string
  readonly bytes?: Uint8Array
  readonly password?: string
}

async function syntheticArchive(
  entries: readonly SyntheticEntry[],
): Promise<File> {
  const writer = new ZipWriter(new BlobWriter('application/zip'))
  for (const entry of entries) {
    await writer.add(
      entry.path,
      entry.bytes
        ? new Uint8ArrayReader(entry.bytes)
        : new TextReader(entry.text ?? ''),
      { level: 0, password: entry.password },
    )
  }
  const blob = await writer.close()
  return new File([blob], 'synthetic-export.zip', {
    type: 'application/zip',
  })
}

async function withCompressionMethod(
  file: File,
  targetPath: string,
  compressionMethod: number,
): Promise<File> {
  const bytes = new Uint8Array(await file.arrayBuffer())
  const view = new DataView(
    bytes.buffer,
    bytes.byteOffset,
    bytes.byteLength,
  )
  const decoder = new TextDecoder()
  let patchedLocal = false
  let patchedCentral = false
  for (let offset = 0; offset <= bytes.byteLength - 30; offset += 1) {
    const signature = view.getUint32(offset, true)
    if (signature === 0x04034b50) {
      const filenameLength = view.getUint16(offset + 26, true)
      const filename = decoder.decode(
        bytes.subarray(offset + 30, offset + 30 + filenameLength),
      )
      if (filename === targetPath) {
        view.setUint16(offset + 8, compressionMethod, true)
        patchedLocal = true
      }
    } else if (signature === 0x02014b50) {
      const filenameLength = view.getUint16(offset + 28, true)
      const filename = decoder.decode(
        bytes.subarray(offset + 46, offset + 46 + filenameLength),
      )
      if (filename === targetPath) {
        view.setUint16(offset + 10, compressionMethod, true)
        patchedCentral = true
      }
    }
  }
  if (!patchedLocal || !patchedCentral) {
    throw new Error('synthetic entry was not found')
  }
  return new File([bytes], file.name, { type: file.type })
}

async function withUncompressedSize(
  file: File,
  targetPath: string,
  uncompressedSize: number,
): Promise<File> {
  const bytes = new Uint8Array(await file.arrayBuffer())
  const view = new DataView(
    bytes.buffer,
    bytes.byteOffset,
    bytes.byteLength,
  )
  const decoder = new TextDecoder()
  let patchedLocal = false
  let patchedCentral = false
  for (let offset = 0; offset <= bytes.byteLength - 30; offset += 1) {
    const signature = view.getUint32(offset, true)
    if (signature === 0x04034b50) {
      const filenameLength = view.getUint16(offset + 26, true)
      const filename = decoder.decode(
        bytes.subarray(offset + 30, offset + 30 + filenameLength),
      )
      if (filename === targetPath) {
        view.setUint32(offset + 22, uncompressedSize, true)
        patchedLocal = true
      }
    } else if (signature === 0x02014b50) {
      const filenameLength = view.getUint16(offset + 28, true)
      const filename = decoder.decode(
        bytes.subarray(offset + 46, offset + 46 + filenameLength),
      )
      if (filename === targetPath) {
        view.setUint32(offset + 24, uncompressedSize, true)
        patchedCentral = true
      }
    }
  }
  if (!patchedLocal || !patchedCentral) {
    throw new Error('synthetic entry was not found')
  }
  return new File([bytes], file.name, { type: file.type })
}

function legacyVideoPost(
  caption: string,
  path = 'media/posts/clip.mp4',
): Record<string, unknown> {
  return {
    title: caption,
    media: [
      {
        uri: path,
        creation_timestamp: 1_700_000_000,
      },
    ],
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('archive adapter integration (IT-002, IT-003)', () => {
  it('reports directory progress after 50ms before the 250-entry interval', async () => {
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify(legacyVideoPost('synthetic')),
      },
    ])
    let elapsed = 0
    vi.spyOn(performance, 'now').mockImplementation(() => {
      elapsed += 51
      return elapsed
    })
    const progress = vi.fn()

    const inspected = await inspectArchive(
      file,
      new Date('2026-07-23T12:00:00Z'),
      new AbortController().signal,
      progress,
    )
    try {
      expect(progress).toHaveBeenCalledWith('directory', 1, 1)
    } finally {
      await inspected.reader.close()
    }
  })

  it('fails wholly unsupported metadata with safe Posts-only guidance', async () => {
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts.json',
        text: JSON.stringify({
          messages: [{ sender: 'PRIVATE-SHAPE-CANARY' }],
        }),
      },
    ])

    await expect(
      inspectArchive(
        file,
        new Date('2026-07-23T12:00:00Z'),
        new AbortController().signal,
      ),
    ).rejects.toThrow('unsupportedPostsMetadata')
  })

  it('bounds actual metadata output even when ZIP size fields under-report it', async () => {
    const path = 'your_instagram_activity/media/posts_1.json'
    const ordinary = await syntheticArchive([
      {
        path,
        text: JSON.stringify(legacyVideoPost('synthetic')),
      },
    ])
    const file = await withUncompressedSize(ordinary, path, 1)

    await expect(
      inspectArchive(
        file,
        new Date('2026-07-23T12:00:00Z'),
        new AbortController().signal,
      ),
    ).rejects.toThrow('candidateMetadataLimit')
  })

  it('turns a private media reference into a content-free visible skip', async () => {
    const canary = 'PRIVATE-PATH-CAPTION-CANARY'
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify(
          legacyVideoPost(canary, 'messages/private.jpg'),
        ),
      },
      {
        path: 'messages/private.jpg',
        bytes: new Uint8Array([1, 2, 3]),
      },
    ])

    const inspected = await inspectArchive(
      file,
      new Date('2026-07-23T12:00:00Z'),
      new AbortController().signal,
    )
    try {
      expect(inspected.manifest.posts).toEqual([])
      expect(inspected.manifest.skipped).toHaveLength(1)
      expect(inspected.manifest.skipped[0]?.code).toBe(
        'unsupportedPost',
      )
      expect(JSON.stringify(inspected.manifest)).not.toContain(canary)
      expect(JSON.stringify(inspected.manifest)).not.toContain(
        'private.jpg',
      )
      expect(inspected.tokenEntries.size).toBe(0)
    } finally {
      await inspected.reader.close()
    }
  })

  it('rejects a Unicode-confusable twin of a supported metadata path', async () => {
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts.json',
        text: JSON.stringify(legacyVideoPost('supported')),
      },
      {
        path: 'your_instagram_activity/media/poſts.json',
        text: JSON.stringify({
          messages: ['PRIVATE-CONFUSABLE-CANARY'],
        }),
      },
    ])

    await expect(
      inspectArchive(
        file,
        new Date('2026-07-23T12:00:00Z'),
        new AbortController().signal,
      ),
    ).rejects.toThrow('ambiguousArchivePath')
  })

  it('applies maxPosts after equivalent raw rows are merged', async () => {
    const rawRows = Array.from({ length: 25_001 }, (_, ordinal) => ({
      ...legacyVideoPost('equivalent'),
      ignored: ordinal.toString(36).padStart(8, '0'),
    }))
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify(rawRows),
      },
    ])

    const inspected = await inspectArchive(
      file,
      new Date('2026-07-23T12:00:00Z'),
      new AbortController().signal,
    )
    try {
      expect(inspected.manifest.posts).toEqual([])
      expect(inspected.manifest.skipped).toHaveLength(1)
      expect(inspected.manifest.skipped[0]?.code).toBe('videoOnly')
    } finally {
      await inspected.reader.close()
    }
  }, 20_000)

  it('gives same-timestamp conflict groups distinct safe item keys', async () => {
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify([
          legacyVideoPost('first a', 'media/posts/a.mp4'),
          legacyVideoPost('second a', 'media/posts/a.mp4'),
          legacyVideoPost('first b', 'media/posts/b.mp4'),
          legacyVideoPost('second b', 'media/posts/b.mp4'),
        ]),
      },
    ])

    const inspected = await inspectArchive(
      file,
      new Date('2026-07-23T12:00:00Z'),
      new AbortController().signal,
    )
    try {
      expect(inspected.manifest.skipped.map((item) => item.code)).toEqual(
        ['ambiguousDuplicate', 'ambiguousDuplicate'],
      )
      expect(
        new Set(
          inspected.manifest.skipped.map((item) => item.itemKey),
        ).size,
      ).toBe(2)
      expect(
        inspected.manifest.skipped.map((item) => item.createdAt),
      ).toEqual([
        '2023-11-14T22:13:20.000Z',
        '2023-11-14T22:13:20.000Z',
      ])
    } finally {
      await inspected.reader.close()
    }
  })

  it('merges recent dual export representations using the media caption and timestamp', async () => {
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify([
          {
            title: '',
            creation_timestamp: 1_700_000_200,
            media: [
              {
                uri: 'media/posts/clip.mp4',
                creation_timestamp: 1_700_000_100,
                title: 'Synthetic caption',
              },
            ],
          },
        ]),
      },
      {
        path: 'your_instagram_activity/media/posts.json',
        text: JSON.stringify([
          {
            timestamp: 1_700_000_300,
            fbid: 'synthetic-structural-fixture',
            media: [],
            label_values: [
              {
                label: 'Media',
                media: [
                  {
                    uri: 'media/posts/clip.mp4',
                    creation_timestamp: 1_700_000_100,
                    title: 'Synthetic caption',
                  },
                ],
              },
              { label: 'Draft', value: 'False' },
              { label: 'Published', value: 'True' },
              {
                label: 'Creation time',
                timestamp_value: 1_600_000_050,
              },
              { label: 'Caption', value: 'Synthetic caption' },
            ],
          },
        ]),
      },
    ])

    const inspected = await inspectArchive(
      file,
      new Date('2026-07-24T12:00:00Z'),
      new AbortController().signal,
    )
    try {
      expect(inspected.manifest.posts).toEqual([])
      expect(inspected.manifest.skipped).toHaveLength(1)
      expect(inspected.manifest.skipped[0]).toMatchObject({
        code: 'videoOnly',
        createdAt: '2023-11-14T22:15:00.000Z',
      })
    } finally {
      await inspected.reader.close()
    }
  })

  it('keeps same-timestamp manifest identity stable across row order', async () => {
    const rows = [
      {
        title: 'Synthetic A',
        media: [
          {
            uri: 'media/posts/a.jpg',
            creation_timestamp: 1_700_000_000,
          },
        ],
      },
      {
        title: 'Synthetic B',
        media: [
          {
            uri: 'media/posts/b.jpg',
            creation_timestamp: 1_700_000_000,
          },
        ],
      },
    ]
    const [forwardFile, reverseFile] = await Promise.all([
      syntheticArchive([
        {
          path: 'your_instagram_activity/media/posts_1.json',
          text: JSON.stringify(rows),
        },
      ]),
      syntheticArchive([
        {
          path: 'your_instagram_activity/media/posts_1.json',
          text: JSON.stringify([...rows].reverse()),
        },
      ]),
    ])
    const now = new Date('2026-07-23T12:00:00Z')
    const forward = await inspectArchive(
      forwardFile,
      now,
      new AbortController().signal,
    )
    const reverse = await inspectArchive(
      reverseFile,
      now,
      new AbortController().signal,
    )
    try {
      expect(
        forward.manifest.posts.map(({ itemKey, rkey }) => ({
          itemKey,
          rkey,
        })),
      ).toEqual(
        reverse.manifest.posts.map(({ itemKey, rkey }) => ({
          itemKey,
          rkey,
        })),
      )
      expect(forward.manifest.fingerprint).toBe(
        reverse.manifest.fingerprint,
      )
    } finally {
      await Promise.all([
        forward.reader.close(),
        reverse.reader.close(),
      ])
    }
  })

  it('keeps 1,025 same-time posts distinct and stable across archive order', async () => {
    const rows = Array.from({ length: 1_025 }, (_, index) => ({
      title: `Synthetic post ${index}`,
      media: [
        {
          uri: `media/posts/${index.toString().padStart(4, '0')}.heic`,
          creation_timestamp: 1_700_000_000,
        },
      ],
    }))
    const [forwardFile, reverseFile] = await Promise.all([
      syntheticArchive([
        {
          path: 'your_instagram_activity/media/posts_1.json',
          text: JSON.stringify(rows),
        },
      ]),
      syntheticArchive([
        {
          path: 'your_instagram_activity/media/posts_1.json',
          text: JSON.stringify([...rows].reverse()),
        },
      ]),
    ])
    const now = new Date('2026-07-23T12:00:00Z')
    const forward = await inspectArchive(
      forwardFile,
      now,
      new AbortController().signal,
    )
    const reverse = await inspectArchive(
      reverseFile,
      now,
      new AbortController().signal,
    )
    try {
      const forwardKeys = forward.manifest.posts.map((post) => post.rkey)
      expect(forward.manifest.posts).toHaveLength(rows.length)
      expect(new Set(forwardKeys).size).toBe(rows.length)
      expect(
        forward.manifest.posts.map(({ itemKey, rkey }) => ({
          itemKey,
          rkey,
        })),
      ).toEqual(
        reverse.manifest.posts.map(({ itemKey, rkey }) => ({
          itemKey,
          rkey,
        })),
      )
      expect(forward.manifest.fingerprint).toBe(
        reverse.manifest.fingerprint,
      )
    } finally {
      await Promise.all([
        forward.reader.close(),
        reverse.reader.close(),
      ])
    }
  }, 30_000)

  it('does not downgrade a missing browser media capability to an omission', async () => {
    vi.spyOn(imageSanitizer, 'sanitizeImage').mockRejectedValue(
      new Error('browserMediaUnsupported'),
    )
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify({
          title: 'image post',
          media: [
            {
              uri: 'media/posts/photo.jpg',
              creation_timestamp: 1_700_000_000,
            },
          ],
        }),
      },
      {
        path: 'media/posts/photo.jpg',
        bytes: new Uint8Array([0xff, 0xd8, 0xff, 0xd9]),
      },
    ])

    await expect(
      inspectArchive(
        file,
        new Date('2026-07-23T12:00:00Z'),
        new AbortController().signal,
      ),
    ).rejects.toThrow('browserMediaUnsupported')
  })

  it('retains the first four usable images when an earlier source image is invalid', async () => {
    const sanitizeImage = vi
      .spyOn(imageSanitizer, 'sanitizeImage')
      .mockImplementation((bytes) => {
        if (bytes[0] === 0) {
          return Promise.reject(new Error('corruptImageHeader'))
        }
        return Promise.resolve({
          buffer: new Uint8Array([bytes[0] ?? 1]).buffer,
          mime: 'image/jpeg',
          width: 100,
          height: 100,
        })
      })
    const media = Array.from({ length: 5 }, (_, index) => ({
      uri: `media/posts/${index + 1}.jpg`,
      creation_timestamp: 1_700_000_000,
    }))
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify({ title: 'image post', media }),
      },
      ...media.map((item, index) => ({
        path: item.uri,
        bytes: new Uint8Array([index]),
      })),
    ])

    const inspected = await inspectArchive(
      file,
      new Date('2026-07-23T12:00:00Z'),
      new AbortController().signal,
    )
    try {
      expect(sanitizeImage).toHaveBeenCalledTimes(5)
      expect(inspected.manifest.posts[0]?.media).toHaveLength(4)
      expect(inspected.manifest.posts[0]?.warnings).toEqual(
        expect.arrayContaining(['imagesOmitted', 'mediaUnavailable']),
      )
    } finally {
      await inspected.reader.close()
    }
  })

  it('bounds transformed captions before they cross the worker manifest boundary', async () => {
    vi.spyOn(imageSanitizer, 'sanitizeImage').mockResolvedValue({
      buffer: new Uint8Array([1]).buffer,
      mime: 'image/jpeg',
      width: 100,
      height: 100,
    })
    const privateTail = 'PRIVATE-OVERSIZED-CAPTION-TAIL'
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify({
          title: `${'x'.repeat(25_000)}${privateTail}`,
          media: [
            {
              uri: 'media/posts/one.jpg',
              creation_timestamp: 1_700_000_000,
            },
          ],
        }),
      },
      {
        path: 'media/posts/one.jpg',
        bytes: new Uint8Array([1]),
      },
    ])

    const inspected = await inspectArchive(
      file,
      new Date('2026-07-23T12:00:00Z'),
      new AbortController().signal,
    )
    try {
      const post = inspected.manifest.posts[0]
      expect(post?.caption).toHaveLength(2_000)
      expect(post?.initialCaption).toBe(post?.caption)
      expect(post?.warnings).toContain('captionTruncated')
      expect(JSON.stringify(inspected.manifest)).not.toContain(privateTail)
    } finally {
      await inspected.reader.close()
    }
  })

  it('rejects an encrypted referenced image instead of partially importing its caption', async () => {
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify({
          title: 'caption must not be partially imported',
          media: [
            {
              uri: 'media/posts/photo.jpg',
              creation_timestamp: 1_700_000_000,
            },
          ],
        }),
      },
      {
        path: 'media/posts/photo.jpg',
        bytes: new Uint8Array([0xff, 0xd8, 0xff, 0xd9]),
        password: 'synthetic-password',
      },
    ])

    await expect(
      inspectArchive(
        file,
        new Date('2026-07-23T12:00:00Z'),
        new AbortController().signal,
      ),
    ).rejects.toThrow('encryptedArchiveUnsupported')
  })

  it('rejects an encrypted entry even when it is outside the Posts allow-list', async () => {
    const file = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify(legacyVideoPost('synthetic')),
      },
      {
        path: 'messages/inbox/private.json',
        text: 'private data',
        password: 'synthetic-password',
      },
    ])

    await expect(
      inspectArchive(
        file,
        new Date('2026-07-23T12:00:00Z'),
        new AbortController().signal,
      ),
    ).rejects.toThrow('encryptedArchiveUnsupported')
  })

  it('rejects an unsupported-compression referenced image instead of partially importing its caption', async () => {
    const ordinary = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify({
          title: 'caption must not be partially imported',
          media: [
            {
              uri: 'media/posts/photo.jpg',
              creation_timestamp: 1_700_000_000,
            },
          ],
        }),
      },
      {
        path: 'media/posts/photo.jpg',
        bytes: new Uint8Array([0xff, 0xd8, 0xff, 0xd9]),
      },
    ])
    const file = await withCompressionMethod(
      ordinary,
      'media/posts/photo.jpg',
      12,
    )

    await expect(
      inspectArchive(
        file,
        new Date('2026-07-23T12:00:00Z'),
        new AbortController().signal,
      ),
    ).rejects.toThrow('archiveCompressionUnsupported')
  })

  it('rejects unsupported compression even outside the Posts allow-list', async () => {
    const ordinary = await syntheticArchive([
      {
        path: 'your_instagram_activity/media/posts_1.json',
        text: JSON.stringify(legacyVideoPost('synthetic')),
      },
      {
        path: 'messages/inbox/private.json',
        text: 'private data',
      },
    ])
    const file = await withCompressionMethod(
      ordinary,
      'messages/inbox/private.json',
      12,
    )

    await expect(
      inspectArchive(
        file,
        new Date('2026-07-23T12:00:00Z'),
        new AbortController().signal,
      ),
    ).rejects.toThrow('archiveCompressionUnsupported')
  })
})
