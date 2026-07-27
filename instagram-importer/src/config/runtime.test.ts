import { describe, expect, it } from 'vitest'

import {
  OAUTH_SCOPE,
  canonicalLoopbackUrl,
  classifyRuntime,
  validateDynamicServiceUrl,
} from './runtime'

describe('runtime policy (UT-013)', () => {
  it('requests the exact approved granular OAuth authority', () => {
    expect(OAUTH_SCOPE).toBe(
      'atproto repo:social.craftsky.feed.post?action=create&action=delete ' +
        'blob?accept=image/jpeg&accept=image/png&accept=image/webp',
    )
  })

  it('allows production, explicit staging, and localhost but disables previews', () => {
    expect(
      classifyRuntime('https://import.craftsky.social', {}),
    ).toMatchObject({ mode: 'production', writesEnabled: true })

    expect(
      classifyRuntime('https://import-stage.craftsky.social', {
        VITE_STAGING_ORIGIN: 'https://import-stage.craftsky.social',
        VITE_ENABLE_PDS_WRITES: 'true',
      }),
    ).toMatchObject({ mode: 'staging', writesEnabled: true })

    expect(classifyRuntime('http://127.0.0.1:5173', {})).toMatchObject({
      mode: 'local',
      writesEnabled: true,
    })

    expect(
      classifyRuntime('https://branch.pages.dev', {}),
    ).toMatchObject({ mode: 'preview', writesEnabled: false })

    expect(
      classifyRuntime('https://import.craftsky.social', {
        MODE: 'preview',
      }),
    ).toMatchObject({
      mode: 'preview',
      clientId: '',
      writesEnabled: false,
    })
  })

  it('moves localhost to an RFC 8252 loopback IP before OAuth state is created', () => {
    expect(
      canonicalLoopbackUrl(
        'http://localhost:5173/review?mode=local#selected',
      ),
    ).toBe('http://127.0.0.1:5173/review?mode=local#selected')
    expect(
      canonicalLoopbackUrl('http://127.0.0.1:5173/review'),
    ).toBeNull()
    expect(
      canonicalLoopbackUrl('https://import.craftsky.social/review'),
    ).toBeNull()

    const runtime = classifyRuntime('http://localhost:5173', {})
    expect(runtime).toMatchObject({
      mode: 'local',
      origin: 'http://127.0.0.1:5173',
      redirectUri: 'http://127.0.0.1:5173/oauth/callback',
    })
    expect(
      new URL(runtime.clientId).searchParams.get('redirect_uri'),
    ).toBe('http://127.0.0.1:5173/oauth/callback')
  })

  it('fails closed for unsafe dynamic service URLs', () => {
    expect(
      validateDynamicServiceUrl(
        'https://pds.example',
        'https://pds.example',
        false,
      ).origin,
    ).toBe('https://pds.example')

    for (const value of [
      'http://pds.example',
      'https://user:pass@pds.example',
      'https://127.0.0.1',
      'https://10.0.0.2',
      'https://192.168.1.5',
      'https://[fd00::1]',
      'https://service.local',
      'https://pds.example#secret',
      'https://other.example',
    ]) {
      expect(() =>
        validateDynamicServiceUrl(
          value,
          'https://pds.example',
          false,
        ),
      ).toThrow()
    }
  })
})
