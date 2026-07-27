import { describe, expect, it, vi } from 'vitest'

import type { RuntimePolicy } from '../config/runtime'
import { processOAuthPopupCallback } from './popupCallback'

const runtime: RuntimePolicy = {
  mode: 'production',
  origin: 'https://import.craftsky.social',
  redirectUri: 'https://import.craftsky.social/oauth/callback',
  clientId:
    'https://import.craftsky.social/oauth-client-metadata.json',
  writesEnabled: true,
}

describe('popup OAuth callback bootstrap (IT-006)', () => {
  it('initializes the OAuth client for the default fragment callback before rendering', async () => {
    const restore = vi
      .fn()
      .mockRejectedValue(new Error('continued in parent'))
    const load = vi.fn().mockResolvedValue({ restore })
    await expect(
      processOAuthPopupCallback(
        runtime,
        {
          search: '',
          hash: '#state=popup-state&code=authorization-code',
        },
        load,
      ),
    ).resolves.toBe(true)
    expect(load).toHaveBeenCalledWith(runtime)
    expect(restore).toHaveBeenCalledTimes(1)
  })

  it('also recognizes an explicitly configured query callback', async () => {
    const restore = vi.fn().mockRejectedValue(new Error('continued'))
    const load = vi.fn().mockResolvedValue({ restore })

    await expect(
      processOAuthPopupCallback(
        runtime,
        {
          search: '?state=popup-state&code=authorization-code',
          hash: '',
        },
        load,
      ),
    ).resolves.toBe(true)
    expect(restore).toHaveBeenCalledTimes(1)
  })

  it('does nothing on an ordinary importer URL', async () => {
    const load = vi.fn()
    await expect(
      processOAuthPopupCallback(
        runtime,
        { search: '', hash: '' },
        load,
      ),
    ).resolves.toBe(false)
    expect(load).not.toHaveBeenCalled()
  })
})
