import { afterEach, describe, expect, it, vi } from 'vitest'

import { sanitizeImage } from './sanitize'
import type { SupportedImageMime } from './inspect'

const ONE_PIXEL_PNG = Uint8Array.from(
  atob(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
  ),
  (character) => character.charCodeAt(0),
)

afterEach(() => vi.unstubAllGlobals())

describe('cancel-aware media sanitizing (UT-018, IT-004)', () => {
  it('stops immediately after a pending browser decode completes', async () => {
    let finishDecode:
      | ((bitmap: { width: number; height: number; close(): void }) => void)
      | undefined
    const close = vi.fn()
    vi.stubGlobal('OffscreenCanvas', class {})
    vi.stubGlobal(
      'createImageBitmap',
      vi.fn(
        () =>
          new Promise((resolve) => {
            finishDecode = resolve
          }),
      ),
    )
    const controller = new AbortController()
    const pending = sanitizeImage(ONE_PIXEL_PNG, controller.signal)

    controller.abort()
    finishDecode?.({ width: 1, height: 1, close })

    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    expect(close).toHaveBeenCalledTimes(1)
  })

  it('does not inspect or return bytes produced after cancellation', async () => {
    let finishEncode: ((blob: Blob) => void) | undefined
    const close = vi.fn()
    vi.stubGlobal(
      'createImageBitmap',
      vi.fn().mockResolvedValue({
        width: 1,
        height: 1,
        close,
      }),
    )
    vi.stubGlobal(
      'OffscreenCanvas',
      class {
        getContext(): { drawImage(): void } {
          return { drawImage: vi.fn() }
        }

        convertToBlob(): Promise<Blob> {
          return new Promise((resolve) => {
            finishEncode = resolve
          })
        }
      },
    )
    const controller = new AbortController()
    const pending = sanitizeImage(ONE_PIXEL_PNG, controller.signal)
    await vi.waitFor(() => expect(finishEncode).toBeTypeOf('function'))

    controller.abort()
    finishEncode?.(new Blob([ONE_PIXEL_PNG], { type: 'image/png' }))

    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    expect(close).toHaveBeenCalledTimes(1)
  })
})

describe('final image geometry', () => {
  it.each<SupportedImageMime>(['image/jpeg', 'image/png', 'image/webp'])(
    'resizes a compressible 5000 by 1000 %s before returning it',
    async (mime) => {
      const canvases = stubImageTransform(mime, 5_000, 1_000)

      const result = await sanitizeImage(imageHeader(mime, 5_000, 1_000))

      expect(canvases[0]).toEqual({ width: 4_000, height: 800 })
      expect(result).toMatchObject({ mime, width: 4_000, height: 800 })
    },
  )

  it.each(
    (['image/jpeg', 'image/png', 'image/webp'] as const).flatMap((mime) => [
      { mime, width: 4_000, height: 1_000, output: [4_000, 1_000] },
      { mime, width: 4_001, height: 1_000, output: [4_000, 999] },
      { mime, width: 1_000, height: 4_001, output: [999, 4_000] },
    ]),
  )(
    'keeps final dimensions within 4000 for $mime $width by $height',
    async ({ mime, width, height, output }) => {
      const canvases = stubImageTransform(mime, width, height)

      const result = await sanitizeImage(imageHeader(mime, width, height))

      expect(canvases[0]).toEqual({ width: output[0], height: output[1] })
      expect(result).toMatchObject({
        mime,
        width: output[0],
        height: output[1],
      })
    },
  )

  it('rejects encoded dimensions outside the final policy', async () => {
    stubImageTransform('image/png', 4_000, 1_000, 4_001, 1_000)

    await expect(
      sanitizeImage(imageHeader('image/png', 4_000, 1_000)),
    ).rejects.toThrow('finalImageDimensionLimit')
  })
})

function stubImageTransform(
  mime: SupportedImageMime,
  width: number,
  height: number,
  encodedWidth?: number,
  encodedHeight?: number,
): Array<{ width: number; height: number }> {
  const canvases: Array<{ width: number; height: number }> = []
  vi.stubGlobal(
    'createImageBitmap',
    vi.fn().mockResolvedValue({ width, height, close: vi.fn() }),
  )
  vi.stubGlobal(
    'OffscreenCanvas',
    class {
      constructor(
        readonly width: number,
        readonly height: number,
      ) {
        canvases.push({ width, height })
      }

      getContext(): { drawImage(): void } {
        return { drawImage: vi.fn() }
      }

      convertToBlob(): Promise<Blob> {
        const header = imageHeader(
          mime,
          encodedWidth ?? this.width,
          encodedHeight ?? this.height,
        )
        const buffer = new ArrayBuffer(header.byteLength)
        new Uint8Array(buffer).set(header)
        return Promise.resolve(new Blob([buffer], { type: mime }))
      }
    },
  )
  return canvases
}

function imageHeader(
  mime: SupportedImageMime,
  width: number,
  height: number,
): Uint8Array {
  if (mime === 'image/jpeg') {
    const bytes = new Uint8Array(25)
    bytes.set([0xff, 0xd8, 0xff, 0xe0, 0x00, 0x02, 0xff, 0xc0, 0x00, 0x11])
    const view = new DataView(bytes.buffer)
    view.setUint16(11, height)
    view.setUint16(13, width)
    return bytes
  }
  if (mime === 'image/png') {
    const bytes = new Uint8Array(33)
    bytes.set([137, 80, 78, 71, 13, 10, 26, 10])
    const view = new DataView(bytes.buffer)
    view.setUint32(8, 13)
    bytes.set(new TextEncoder().encode('IHDR'), 12)
    view.setUint32(16, width)
    view.setUint32(20, height)
    return bytes
  }

  const bytes = new Uint8Array(30)
  bytes.set(new TextEncoder().encode('RIFF'), 0)
  const view = new DataView(bytes.buffer)
  view.setUint32(4, 22, true)
  bytes.set(new TextEncoder().encode('WEBPVP8X'), 8)
  view.setUint32(16, 10, true)
  for (const [offset, value] of [
    [24, width - 1],
    [27, height - 1],
  ] as const) {
    bytes[offset] = value & 0xff
    bytes[offset + 1] = (value >> 8) & 0xff
    bytes[offset + 2] = (value >> 16) & 0xff
  }
  return bytes
}
