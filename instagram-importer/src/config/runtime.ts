export const PRODUCTION_ORIGIN = 'https://import.craftsky.social'
export const HANDLE_RESOLVER = 'https://bsky.social'

export const OAUTH_SCOPE =
  'atproto repo:social.craftsky.feed.post?action=create&action=delete ' +
  'blob?accept=image/jpeg&accept=image/png&accept=image/webp'

export type RuntimeMode = 'local' | 'staging' | 'production' | 'preview'

export interface RuntimePolicy {
  readonly mode: RuntimeMode
  readonly origin: string
  readonly redirectUri: string
  readonly clientId: string
  readonly writesEnabled: boolean
}

type RuntimeEnvironment = Readonly<Record<string, string | undefined>>

const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '[::1]'])

export function canonicalLoopbackUrl(rawUrl: string): string | null {
  const parsed = new URL(rawUrl)
  if (parsed.hostname !== 'localhost') return null
  parsed.hostname = '127.0.0.1'
  return parsed.href
}

function isPrivateNetworkHost(hostname: string): boolean {
  const normalized = hostname.toLocaleLowerCase('en-US')
  if (
    normalized.endsWith('.local') ||
    normalized.endsWith('.localhost') ||
    normalized.endsWith('.internal') ||
    normalized.endsWith('.lan')
  ) {
    return true
  }
  const ipv4 = normalized.split('.').map(Number)
  if (
    ipv4.length === 4 &&
    ipv4.every(
      (part) => Number.isInteger(part) && part >= 0 && part <= 255,
    )
  ) {
    const [first, second] = ipv4
    return (
      first === 0 ||
      first === 10 ||
      first === 127 ||
      (first === 100 && second >= 64 && second <= 127) ||
      (first === 169 && second === 254) ||
      (first === 172 && second >= 16 && second <= 31) ||
      (first === 192 && second === 168) ||
      (first === 198 && (second === 18 || second === 19)) ||
      first >= 224
    )
  }
  const ipv6 = normalized.replace(/^\[|\]$/gu, '')
  return (
    ipv6 === '::' ||
    ipv6 === '::1' ||
    ipv6.startsWith('fc') ||
    ipv6.startsWith('fd') ||
    /^fe[89ab]/u.test(ipv6)
  )
}

export function classifyRuntime(
  rawOrigin: string,
  environment: RuntimeEnvironment,
): RuntimePolicy {
  const origin = new URL(
    canonicalLoopbackUrl(rawOrigin) ?? rawOrigin,
  ).origin
  const redirectUri = `${origin}/oauth/callback`

  if (environment.MODE === 'preview') {
    return {
      mode: 'preview',
      origin,
      redirectUri,
      clientId: '',
      writesEnabled: false,
    }
  }

  if (origin === PRODUCTION_ORIGIN) {
    return {
      mode: 'production',
      origin,
      redirectUri,
      clientId: `${PRODUCTION_ORIGIN}/oauth-client-metadata.json`,
      writesEnabled: true,
    }
  }

  const stagingOrigin = environment.VITE_STAGING_ORIGIN
  if (stagingOrigin && origin === new URL(stagingOrigin).origin) {
    return {
      mode: 'staging',
      origin,
      redirectUri,
      clientId:
        environment.VITE_OAUTH_CLIENT_ID ??
        `${origin}/oauth-client-metadata.json`,
      writesEnabled: environment.VITE_ENABLE_PDS_WRITES === 'true',
    }
  }

  const parsed = new URL(origin)
  if (LOOPBACK_HOSTS.has(parsed.hostname)) {
    const scope = encodeURIComponent(OAUTH_SCOPE)
    const redirect = encodeURIComponent(redirectUri)
    return {
      mode: 'local',
      origin,
      redirectUri,
      clientId: `http://localhost?redirect_uri=${redirect}&scope=${scope}`,
      writesEnabled: true,
    }
  }

  return {
    mode: 'preview',
    origin,
    redirectUri,
    clientId: '',
    writesEnabled: false,
  }
}

export function validateDynamicServiceUrl(
  rawUrl: string,
  authenticatedServiceOrigin: string,
  allowLoopback: boolean,
): URL {
  const parsed = new URL(rawUrl)
  const authenticatedOrigin = new URL(authenticatedServiceOrigin).origin

  if (parsed.username || parsed.password || parsed.hash) {
    throw new Error('unsafeServiceUrl')
  }

  const isLoopback = LOOPBACK_HOSTS.has(parsed.hostname)
  if (
    (parsed.protocol !== 'https:' && !(allowLoopback && isLoopback)) ||
    (isLoopback && !allowLoopback)
  ) {
    throw new Error('unsafeServiceUrl')
  }
  if (!isLoopback && isPrivateNetworkHost(parsed.hostname)) {
    throw new Error('unsafeServiceUrl')
  }

  if (parsed.origin !== authenticatedOrigin) {
    throw new Error('serviceOriginMismatch')
  }

  return parsed
}

export function requireWritesEnabled(policy: RuntimePolicy): void {
  if (!policy.writesEnabled) {
    throw new Error('previewWritesDisabled')
  }
}
