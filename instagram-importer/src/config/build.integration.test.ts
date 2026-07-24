/// <reference types="node" />
// @vitest-environment node

import { readFile, readdir } from 'node:fs/promises'

import { describe, expect, it } from 'vitest'

import { OAUTH_SCOPE, PRODUCTION_ORIGIN } from './runtime'

const publicFile = (name: string) =>
  readFile(new URL(`../../public/${name}`, import.meta.url), 'utf8')

describe('static deployment contract (IT-016)', () => {
  it('publishes canonical least-authority production OAuth metadata', async () => {
    const metadata = JSON.parse(
      await publicFile('oauth-client-metadata.json'),
    ) as Record<string, unknown>

    expect(metadata).toMatchObject({
      client_id: `${PRODUCTION_ORIGIN}/oauth-client-metadata.json`,
      client_uri: PRODUCTION_ORIGIN,
      redirect_uris: [`${PRODUCTION_ORIGIN}/oauth/callback`],
      grant_types: ['authorization_code', 'refresh_token'],
      response_types: ['code'],
      scope: OAUTH_SCOPE,
      token_endpoint_auth_method: 'none',
      application_type: 'web',
      dpop_bound_access_tokens: true,
    })
    expect(metadata.scope).not.toMatch(
      /repo:\*|action=update|account|identity/,
    )
  })

  it('emits the restrictive Cloudflare Pages header baseline', async () => {
    const headers = await publicFile('_headers')

    expect(headers).toContain("default-src 'self'")
    expect(headers).toContain("script-src 'self'")
    expect(headers).toContain("style-src 'self'")
    expect(headers).toContain("style-src-attr 'unsafe-inline'")
    expect(headers).toContain("connect-src 'self' https:")
    expect(headers).toContain("worker-src 'self' blob:")
    expect(headers).toContain("object-src 'none'")
    expect(headers).toContain("frame-ancestors 'none'")
    expect(headers).toContain('Referrer-Policy: no-referrer')
    expect(headers).toContain(
      'Cross-Origin-Opener-Policy: same-origin-allow-popups',
    )
    expect(headers).toContain('X-Robots-Tag: noindex')
    expect(headers.match(/! Cache-Control/g)).toHaveLength(2)
    expect(headers).not.toMatch(
      /google-analytics|googletagmanager|posthog|sentry|fonts\.googleapis/,
    )
  })

  it('keeps the callback in the SPA and excludes the importer from indexing', async () => {
    await expect(publicFile('_redirects')).resolves.toContain(
      '/oauth/callback  /index.html  200',
    )
    await expect(publicFile('robots.txt')).resolves.toBe(
      'User-agent: *\nDisallow: /\n',
    )
  })

  it('keeps archive import code out of Flutter (REG-008)', async () => {
    const appRoot = new URL('../../../app/', import.meta.url)
    const pubspec = await readFile(new URL('pubspec.yaml', appRoot), 'utf8')
    const appFiles = await readdir(new URL('lib/', appRoot), {
      recursive: true,
    })
    const dartSources = await Promise.all(
      appFiles
        .filter((path) => path.endsWith('.dart'))
        .map((path) => readFile(new URL(`lib/${path}`, appRoot), 'utf8')),
    )
    const flutterSurface = [pubspec, ...dartSources].join('\n')

    expect(flutterSurface).not.toMatch(
      /package:archive\/|ZipDecoder|InstagramPostImporter|instagram-importer/u,
    )
    expect(pubspec).not.toMatch(/^\s+archive:\s/um)
  })
})
