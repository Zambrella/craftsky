import { afterEach, describe, expect, it, vi } from 'vitest'

import { sanitizeImage } from './sanitize'

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
      | ((bitmap: {
          width: number
          height: number
          close(): void
        }) => void)
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
