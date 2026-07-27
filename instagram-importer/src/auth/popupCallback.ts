import type { RuntimePolicy } from '../config/runtime'
import { ImporterAuthService } from './authService'

export interface OAuthCallbackInitializer {
  restore(): Promise<unknown>
}

export async function processOAuthPopupCallback(
  runtime: RuntimePolicy,
  location: {
    readonly search: string
    readonly hash: string
  },
  load: (
    runtime: RuntimePolicy,
  ) => Promise<OAuthCallbackInitializer> = (policy) =>
    ImporterAuthService.load(policy),
): Promise<boolean> {
  const query = new URLSearchParams(location.search)
  const fragment = new URLSearchParams(
    location.hash.startsWith('#')
      ? location.hash.slice(1)
      : location.hash,
  )
  const params =
    fragment.has('state') &&
    (fragment.has('code') || fragment.has('error'))
      ? fragment
      : query
  if (
    !params.has('state') ||
    (!params.has('code') && !params.has('error'))
  ) {
    return false
  }
  try {
    const auth = await load(runtime)
    await auth.restore()
  } catch {
    // signInPopup callbacks intentionally throw after reporting the result
    // through BroadcastChannel and closing the child window.
  }
  return true
}
