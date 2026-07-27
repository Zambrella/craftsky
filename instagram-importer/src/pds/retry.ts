import { XRPCError } from '@atproto/api'

export interface RetryOptions {
  readonly maxAttempts?: number
  readonly maxDelayMs?: number
  readonly sleep?: (
    milliseconds: number,
    signal?: AbortSignal,
  ) => Promise<void>
  readonly random?: () => number
  readonly signal?: AbortSignal
}

type RetryableError = Error & {
  readonly status?: number
  readonly retryAfterMs?: number
  readonly headers?: Readonly<Record<string, string | undefined>>
}

function isTransient(error: unknown): error is RetryableError {
  if (error instanceof TypeError) return true
  if (!(error instanceof Error)) return false
  const status = (error as RetryableError).status
  return (
    (error instanceof XRPCError && status === 1) ||
    status === 408 ||
    status === 425 ||
    status === 429 ||
    (status !== undefined && status >= 500 && status <= 599)
  )
}

function defaultSleep(
  milliseconds: number,
  signal?: AbortSignal,
): Promise<void> {
  const abortError = (): Error =>
    signal?.reason instanceof Error
      ? signal.reason
      : new DOMException('Aborted', 'AbortError')
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(abortError())
      return
    }
    const timer = window.setTimeout(resolve, milliseconds)
    signal?.addEventListener(
      'abort',
      () => {
        window.clearTimeout(timer)
        reject(abortError())
      },
      { once: true },
    )
  })
}

function retryAfterMilliseconds(error: RetryableError): number | undefined {
  if (
    typeof error.retryAfterMs === 'number' &&
    Number.isFinite(error.retryAfterMs)
  ) {
    return error.retryAfterMs
  }
  const entry = Object.entries(error.headers ?? {}).find(
    ([name]) => name.toLocaleLowerCase('en-US') === 'retry-after',
  )?.[1]
  if (!entry) return undefined
  const seconds = Number(entry)
  if (Number.isFinite(seconds) && seconds >= 0) return seconds * 1_000
  const date = Date.parse(entry)
  return Number.isFinite(date) ? Math.max(0, date - Date.now()) : undefined
}

export async function withTransientRetry<T>(
  operation: (attempt: number) => Promise<T>,
  options: RetryOptions = {},
): Promise<T> {
  const maxAttempts = options.maxAttempts ?? 3
  const maxDelayMs = options.maxDelayMs ?? 30_000
  const sleep = options.sleep ?? defaultSleep
  const random = options.random ?? Math.random

  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    options.signal?.throwIfAborted()
    try {
      return await operation(attempt)
    } catch (error) {
      if (attempt === maxAttempts || !isTransient(error)) throw error
      const exponential = 1_000 * 2 ** (attempt - 1)
      const jittered = exponential + Math.floor(random() * exponential)
      const requested = retryAfterMilliseconds(error) ?? jittered
      await sleep(Math.min(Math.max(0, requested), maxDelayMs), options.signal)
    }
  }
  throw new Error('retryExhausted')
}
