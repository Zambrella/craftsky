import { describe, expect, it, vi } from 'vitest'
import type {
  BrowserOAuthClient,
  OAuthSession,
} from '@atproto/oauth-client-browser'

import { OAUTH_SCOPE, type RuntimePolicy } from '../config/runtime'
import {
  assertExactAuthority,
  ImporterAuthService,
  verifyCraftskyMembership,
} from './authService'

const runtime: RuntimePolicy = {
  mode: 'local',
  origin: 'http://localhost:5173',
  redirectUri: 'http://localhost:5173/oauth/callback',
  clientId: 'http://localhost',
  writesEnabled: true,
}

function rejectedSession(scope: string): OAuthSession {
  return {
    did: 'did:plc:test',
    getTokenInfo: vi.fn().mockResolvedValue({
      scope,
      aud: 'https://pds.example',
    }),
    signOut: vi.fn().mockResolvedValue(undefined),
  } as unknown as OAuthSession
}

describe('OAuth authority and CraftSky membership preflight (IT-006)', () => {
  it('accepts only the exact requested permission set', () => {
    expect(() => assertExactAuthority(OAUTH_SCOPE)).not.toThrow()
    expect(() =>
      assertExactAuthority(`${OAUTH_SCOPE} repo:*`),
    ).toThrow('oauthAuthorityMismatch')
    expect(() =>
      assertExactAuthority(
        'atproto repo:social.craftsky.feed.post?action=create',
      ),
    ).toThrow('oauthAuthorityMismatch')
  })

  it('reads only the authenticated profile self record', async () => {
    const getRecord = vi.fn().mockResolvedValue({
      success: true,
      data: { uri: 'at://did:plc:test/social.craftsky.actor.profile/self' },
    })
    await expect(
      verifyCraftskyMembership(
        { getRecord },
        'did:plc:test',
      ),
    ).resolves.toBeUndefined()
    expect(getRecord).toHaveBeenCalledWith({
      repo: 'did:plc:test',
      collection: 'social.craftsky.actor.profile',
      rkey: 'self',
    })
  })

  it('fails closed when the profile self record is absent', async () => {
    await expect(
      verifyCraftskyMembership(
        {
          getRecord: vi.fn().mockRejectedValue(
            Object.assign(new Error('not found'), {
              error: 'RecordNotFound',
              status: 400,
            }),
          ),
        },
        'did:plc:test',
      ),
    ).rejects.toThrow('notCraftskyMember')
  })

  it('does not misreport a PDS or network failure as missing membership', async () => {
    const unavailable = Object.assign(new Error('service unavailable'), {
      status: 503,
    })
    await expect(
      verifyCraftskyMembership(
        {
          getRecord: vi.fn().mockRejectedValue(unavailable),
        },
        'did:plc:test',
      ),
    ).rejects.toBe(unavailable)
  })

  it('removes a popup OAuth session when authority preflight fails', async () => {
    const session = rejectedSession(`${OAUTH_SCOPE} repo:*`)
    const client = {
      signInPopup: vi.fn().mockResolvedValue(session),
    } as unknown as BrowserOAuthClient
    const service = new ImporterAuthService(client, runtime)

    await expect(service.signIn('maker.example')).rejects.toThrow(
      'oauthAuthorityMismatch',
    )
    expect(session.signOut).toHaveBeenCalledTimes(1)
  })

  it('removes a restored OAuth session when membership preflight fails', async () => {
    const session = rejectedSession(OAUTH_SCOPE)
    const client = {
      init: vi.fn().mockResolvedValue({ session }),
    } as unknown as BrowserOAuthClient
    const service = new ImporterAuthService(client, runtime)
    vi.spyOn(service, 'prepare').mockRejectedValueOnce(
      new Error('notCraftskyMember'),
    )

    await expect(service.restore()).rejects.toThrow('notCraftskyMember')
    expect(session.signOut).toHaveBeenCalledTimes(1)
  })
})
