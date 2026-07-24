import { expect, test } from '@playwright/test'
import {
  BlobWriter,
  TextReader,
  Uint8ArrayReader,
  ZipWriter,
} from '@zip.js/zip.js'

type WorkerClientModule = typeof import('../src/worker/client')

const ONE_PIXEL_PNG = Uint8Array.from(
  Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
    'base64',
  ),
)

async function syntheticInstagramArchive(): Promise<Buffer> {
  const writer = new ZipWriter(new BlobWriter('application/zip'))
  await writer.add(
    'your_instagram_activity/media/posts_1.json',
    new TextReader(
      JSON.stringify({
        title: 'Synthetic knitting history #making',
        media: [
          {
            uri: 'media/posts/synthetic.png',
            creation_timestamp: 1_704_164_645,
          },
        ],
      }),
    ),
  )
  await writer.add(
    'media/posts/synthetic.png',
    new Uint8ArrayReader(ONE_PIXEL_PNG),
  )
  await writer.add(
    'your_instagram_activity/messages/private-canary.json',
    new TextReader('PRIVATE-MESSAGE-CANARY'),
  )
  const blob = await writer.close()
  return Buffer.from(await blob.arrayBuffer())
}

async function syntheticSlowInspectionArchive(): Promise<Buffer> {
  const writer = new ZipWriter(new BlobWriter('application/zip'))
  const posts = Array.from({ length: 5_000 }, (_, index) => ({
    title: `Synthetic post ${index}`,
    media: [
      {
        uri: `media/posts/missing-${index}.png`,
        creation_timestamp: 1_704_164_645 + index,
      },
    ],
  }))
  await writer.add(
    'your_instagram_activity/media/posts_1.json',
    new TextReader(JSON.stringify(posts)),
  )
  await writer.add(
    'media/posts/synthetic.png',
    new Uint8ArrayReader(ONE_PIXEL_PNG),
  )
  const blob = await writer.close()
  return Buffer.from(await blob.arrayBuffer())
}

async function syntheticEncryptedArchive(): Promise<Buffer> {
  const writer = new ZipWriter(new BlobWriter('application/zip'))
  await writer.add(
    'your_instagram_activity/media/posts_1.json',
    new TextReader(
      JSON.stringify({
        title: 'Synthetic caption',
        media: [
          {
            uri: 'media/posts/synthetic.png',
            creation_timestamp: 1_704_164_645,
          },
        ],
      }),
    ),
  )
  await writer.add(
    'your_instagram_activity/messages/private.json',
    new TextReader('private data'),
    { password: 'synthetic-password' },
  )
  const blob = await writer.close()
  return Buffer.from(await blob.arrayBuffer())
}

test('canonicalizes localhost before creating local OAuth state (AT-003, UT-013)', async ({
  page,
}) => {
  await page.goto('http://localhost:4173/?mode=local#ready')

  await expect.poll(() => new URL(page.url()).hostname).toBe('127.0.0.1')
  expect(new URL(page.url()).search).toBe('?mode=local')
  expect(new URL(page.url()).hash).toBe('#ready')
  await expect(
    page.getByRole('heading', {
      name: 'Import your Instagram posts into CraftSky',
    }),
  ).toBeVisible()
})

test('reviews a synthetic ZIP locally before any external request or persistence (AT-001; browser worker/privacy smoke for AT-002, IT-002, IT-003, IT-017)', async ({
  page,
}) => {
  const requests: string[] = []
  page.on('request', (request) => requests.push(request.url()))
  await page.goto('/')
  await page
    .getByRole('checkbox', {
      name: /selected posts and images will become public/i,
    })
    .check()
  await page.locator('input[type=file]').setInputFiles({
    name: 'private-source-name.zip',
    mimeType: 'application/zip',
    buffer: await syntheticInstagramArchive(),
  })

  await expect(
    page.getByRole('heading', { name: /ready to review/i }),
  ).toBeVisible()
  await expect(page.getByText('1 post · 1 image')).toBeVisible()
  await page
    .getByRole('button', { name: 'Preview image 1' })
    .click()
  await expect(
    page.getByRole('img', { name: 'Image 1 preview' }),
  ).toHaveAttribute('src', /^blob:/u)
  await expect(page.getByText('PRIVATE-MESSAGE-CANARY')).toHaveCount(0)
  await expect(page.getByText('private-source-name.zip')).toHaveCount(0)
  expect(
    requests.filter(
      (url) => new URL(url).origin !== 'http://127.0.0.1:4173',
    ),
  ).toEqual([])
  const inlineStyles = await page
    .locator('[style]')
    .evaluateAll((nodes) =>
      nodes.map((node) => node.getAttribute('style') ?? ''),
    )
  expect(JSON.stringify(inlineStyles)).not.toContain(
    'PRIVATE-MESSAGE-CANARY',
  )
  expect(JSON.stringify(inlineStyles)).not.toContain(
    'private-source-name.zip',
  )
  const browserStorage = await page.evaluate(async () => ({
    local: Object.entries(localStorage),
    session: Object.entries(sessionStorage),
    databases:
      typeof indexedDB.databases === 'function'
        ? await indexedDB.databases()
        : [],
    caches:
      'caches' in globalThis ? await globalThis.caches.keys() : [],
  }))
  expect(JSON.stringify(browserStorage)).not.toContain(
    'PRIVATE-MESSAGE-CANARY',
  )
  expect(JSON.stringify(browserStorage)).not.toContain(
    'private-source-name.zip',
  )
})

test('cancel fences an inspection and a later archive can be selected (AT-004, UT-018)', async ({
  page,
}) => {
  await page.goto('/')
  await page
    .getByRole('checkbox', {
      name: /selected posts and images will become public/i,
    })
    .check()
  const archive = await syntheticInstagramArchive()
  await page.locator('input[type=file]').setInputFiles({
    name: 'first.zip',
    mimeType: 'application/zip',
    buffer: await syntheticSlowInspectionArchive(),
  })
  const cancel = page.getByRole('button', { name: 'Cancel' })
  await expect(cancel).toBeVisible()
  await cancel.click()
  await expect(
    page.getByRole('heading', { name: 'Choose your Instagram export' }),
  ).toBeVisible()
  await expect(page.getByRole('alert')).toHaveCount(0)

  await page.locator('input[type=file]').setInputFiles({
    name: 'second.zip',
    mimeType: 'application/zip',
    buffer: archive,
  })
  await expect(
    page.getByRole('heading', { name: /ready to review/i }),
  ).toBeVisible()
})

test('sanitizes a selected image inside the archive worker (browser worker smoke for AT-005, IT-004)', async ({
  page,
}) => {
  await page.goto('/')
  const archiveBase64 = (await syntheticInstagramArchive()).toString(
    'base64',
  )
  const result = await page.evaluate(async (encodedArchive) => {
    const workerModulePath = '/src/worker/client.ts'
    const { createArchiveWorkerClient } = (await import(
      /* @vite-ignore */ workerModulePath
    )) as WorkerClientModule
    const bytes = Uint8Array.from(atob(encodedArchive), (character) =>
      character.charCodeAt(0),
    )
    const client = createArchiveWorkerClient()
    try {
      const manifest = await client.inspect(
        new File([bytes], 'synthetic.zip', {
          type: 'application/zip',
        }),
      )
      const token = manifest.posts[0]?.media[0]?.token
      if (!token) throw new Error('missing synthetic media')
      const sanitized = await client.sanitize(token)
      return {
        mime: sanitized.mime,
        width: sanitized.width,
        height: sanitized.height,
        signature: Array.from(
          new Uint8Array(sanitized.buffer).slice(0, 4),
        ),
      }
    } finally {
      client.dispose()
    }
  }, archiveBase64)

  expect(result).toEqual({
    mime: 'image/png',
    width: 1,
    height: 1,
    signature: [0x89, 0x50, 0x4e, 0x47],
  })
})

test('rejects an encrypted entry across the real archive-worker boundary (IT-002)', async ({
  page,
}) => {
  await page.goto('/')
  const archiveBase64 = (await syntheticEncryptedArchive()).toString(
    'base64',
  )
  const errorCode = await page.evaluate(async (encodedArchive) => {
    const workerModulePath = '/src/worker/client.ts'
    const { createArchiveWorkerClient } = (await import(
      /* @vite-ignore */ workerModulePath
    )) as WorkerClientModule
    const bytes = Uint8Array.from(atob(encodedArchive), (character) =>
      character.charCodeAt(0),
    )
    const client = createArchiveWorkerClient()
    try {
      return await client
        .inspect(
          new File([bytes], 'encrypted.zip', {
            type: 'application/zip',
          }),
        )
        .then(
          () => 'unexpectedSuccess',
          (error: unknown) =>
            error instanceof Error ? error.message : 'unknownError',
        )
    } finally {
      client.dispose()
    }
  }, archiveBase64)

  expect(errorCode).toBe('archiveUnsupported')
})
