import { Agent } from '@atproto/api'
import {
  BrowserOAuthClient,
  type OAuthSession,
} from '@atproto/oauth-client-browser'

import {
  HANDLE_RESOLVER,
  OAUTH_SCOPE,
  validateDynamicServiceUrl,
  type RuntimePolicy,
} from '../config/runtime'

export interface ProfileRecordReader {
  getRecord(input: {
    repo: string
    collection: string
    rkey: string
  }): Promise<unknown>
}

export interface CraftskyAccountProfile {
  readonly displayName?: string
  readonly avatarUrl?: string
}

function recordValue(value: unknown): Record<string, unknown> | undefined {
  return typeof value === 'object' && value !== null
    ? (value as Record<string, unknown>)
    : undefined
}

function hasCustomStringifier(
  value: unknown,
): value is { toString(): string } {
  if (typeof value !== 'object' || value === null) return false
  const stringifier = (value as { readonly toString?: unknown })
    .toString
  return (
    typeof stringifier === 'function' &&
    stringifier !== Object.prototype.toString
  )
}

function blobCid(value: unknown): string | undefined {
  const reference = recordValue(value)?.ref
  const referenceRecord = recordValue(reference)
  if (typeof referenceRecord?.$link === 'string') {
    return referenceRecord.$link
  }
  return hasCustomStringifier(reference)
    ? reference.toString()
    : undefined
}

export async function readCraftskyAccountProfile(
  reader: ProfileRecordReader,
  did: string,
): Promise<CraftskyAccountProfile> {
  const response = (await reader.getRecord({
    repo: did,
    collection: 'app.bsky.actor.profile',
    rkey: 'self',
  })) as {
    readonly data?: {
      readonly value?: unknown
    }
  }
  const profile = recordValue(response.data?.value)
  if (profile?.$type !== 'app.bsky.actor.profile') return {}
  const avatar = recordValue(profile.avatar)
  const displayName =
    typeof profile.displayName === 'string'
      ? profile.displayName.trim() || undefined
      : undefined
  const avatarCid = blobCid(avatar)
  const avatarFormat =
    typeof avatar?.mimeType === 'string'
      ? avatar.mimeType.split('/').at(-1)
      : undefined
  return {
    displayName,
    avatarUrl:
      avatarCid && avatarFormat
        ? `https://cdn.bsky.app/img/avatar/plain/${did}/${avatarCid}@${avatarFormat}`
        : undefined,
  }
}

function canonicalScope(scope: string): string[] {
  return [...new Set(scope.trim().split(/\s+/u).filter(Boolean))].sort()
}

export function assertExactAuthority(grantedScope: string): void {
  const requested = canonicalScope(OAUTH_SCOPE)
  const granted = canonicalScope(grantedScope)
  if (
    requested.length !== granted.length ||
    requested.some((permission, index) => permission !== granted[index])
  ) {
    throw new Error('oauthAuthorityMismatch')
  }
}

export async function verifyCraftskyMembership(
  reader: ProfileRecordReader,
  did: string,
): Promise<void> {
  try {
    await reader.getRecord({
      repo: did,
      collection: 'social.craftsky.actor.profile',
      rkey: 'self',
    })
  } catch (error) {
    const protocol = error as {
      readonly error?: string
      readonly status?: number
    }
    if (
      protocol.error === 'RecordNotFound' &&
      protocol.status === 400
    ) {
      throw new Error('notCraftskyMember', { cause: error })
    }
    throw error
  }
}

export interface AuthenticatedPds {
  readonly did: string
  readonly pdsOrigin: string
  readonly displayName?: string
  readonly accountLabel?: string
  readonly avatarUrl?: string
  readonly session: OAuthSession
  readonly agent: Agent
}

export class ImporterAuthService {
  constructor(
    readonly client: BrowserOAuthClient,
    private readonly runtime: RuntimePolicy,
  ) {}

  static async load(runtime: RuntimePolicy): Promise<ImporterAuthService> {
    if (!runtime.clientId) throw new Error('previewWritesDisabled')
    const client = await BrowserOAuthClient.load({
      clientId: runtime.clientId,
      handleResolver: HANDLE_RESOLVER,
      allowHttp: runtime.mode === 'local',
    })
    return new ImporterAuthService(client, runtime)
  }

  async restore(): Promise<AuthenticatedPds | null> {
    if (!this.runtime.writesEnabled) return null
    const result = await this.client.init()
    return result ? this.prepareOrSignOut(result.session) : null
  }

  async signIn(input: string): Promise<AuthenticatedPds> {
    if (!this.runtime.writesEnabled) {
      throw new Error('previewWritesDisabled')
    }
    const session = await this.client.signInPopup(input.trim(), {
      scope: OAUTH_SCOPE,
      popupName: 'craftsky-instagram-importer-oauth',
    })
    return this.prepareOrSignOut(
      session,
      input.startsWith('did:') ? undefined : input.replace(/^@/u, ''),
    )
  }

  async prepare(
    session: OAuthSession,
    accountLabel?: string,
  ): Promise<AuthenticatedPds> {
    const token = await session.getTokenInfo('auto')
    assertExactAuthority(token.scope)
    const pds = validateDynamicServiceUrl(
      token.aud,
      token.aud,
      this.runtime.mode === 'local',
    )
    const agent = new Agent(session)
    await verifyCraftskyMembership(
      {
        getRecord: (input) =>
          agent.com.atproto.repo.getRecord(input),
      },
      session.did,
    )
    const profile = await readCraftskyAccountProfile(
      {
        getRecord: (input) =>
          agent.com.atproto.repo.getRecord(input),
      },
      session.did,
    ).catch((): CraftskyAccountProfile => ({}))
    return {
      did: session.did,
      pdsOrigin: pds.origin,
      displayName: profile.displayName,
      accountLabel,
      avatarUrl: profile.avatarUrl,
      session,
      agent,
    }
  }

  async signOut(authenticated: AuthenticatedPds): Promise<void> {
    await authenticated.session.signOut()
  }

  private async prepareOrSignOut(
    session: OAuthSession,
    accountLabel?: string,
  ): Promise<AuthenticatedPds> {
    try {
      return await this.prepare(session, accountLabel)
    } catch (error) {
      await session.signOut().catch(() => undefined)
      throw error
    }
  }
}
