import { XRPCError } from '@atproto/api'
import { describe, expect, it, vi } from 'vitest'

import { withTransientRetry } from './retry'

describe('bounded publisher retry (UT-011)', () => {
  it('honours a bounded retry delay and stops after success', async () => {
    const operation = vi
      .fn()
      .mockRejectedValueOnce(
        Object.assign(new Error('rate limited'), {
          status: 429,
          retryAfterMs: 2_000,
        }),
      )
      .mockResolvedValue('ok')
    const sleep = vi.fn().mockResolvedValue(undefined)
    await expect(
      withTransientRetry(operation, { sleep, random: () => 0 }),
    ).resolves.toBe('ok')
    expect(operation).toHaveBeenCalledTimes(2)
    expect(sleep).toHaveBeenCalledWith(2_000, undefined)
  })

  it('does not retry validation failures', async () => {
    const operation = vi
      .fn()
      .mockRejectedValue(Object.assign(new Error('invalid'), { status: 400 }))
    await expect(
      withTransientRetry(operation, {
        sleep: vi.fn(),
        random: () => 0,
      }),
    ).rejects.toThrow('invalid')
    expect(operation).toHaveBeenCalledTimes(1)
  })

  it('retries the network-failure shape used by the AT Protocol client', async () => {
    const operation = vi
      .fn()
      .mockRejectedValueOnce(
        new XRPCError(1, undefined, 'network unavailable'),
      )
      .mockResolvedValue('ok')
    const sleep = vi.fn().mockResolvedValue(undefined)

    await expect(
      withTransientRetry(operation, { sleep, random: () => 0 }),
    ).resolves.toBe('ok')

    expect(operation).toHaveBeenCalledTimes(2)
    expect(sleep).toHaveBeenCalledTimes(1)
  })

  it('honours a standard Retry-After response header', async () => {
    const operation = vi
      .fn()
      .mockRejectedValueOnce(
        Object.assign(new Error('busy'), {
          status: 503,
          headers: { 'retry-after': '4' },
        }),
      )
      .mockResolvedValue('ok')
    const sleep = vi.fn().mockResolvedValue(undefined)
    await withTransientRetry(operation, { sleep, random: () => 0 })
    expect(sleep).toHaveBeenCalledWith(4_000, undefined)
  })
})
