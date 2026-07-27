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
    return this.prepareOrSignOut(session)
  }

  async prepare(session: OAuthSession): Promise<AuthenticatedPds> {
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
    return {
      did: session.did,
      pdsOrigin: pds.origin,
      session,
      agent,
    }
  }

  async signOut(authenticated: AuthenticatedPds): Promise<void> {
    await authenticated.session.signOut()
  }

  private async prepareOrSignOut(
    session: OAuthSession,
  ): Promise<AuthenticatedPds> {
    try {
      return await this.prepare(session)
    } catch (error) {
      await session.signOut().catch(() => undefined)
      throw error
    }
  }
}
