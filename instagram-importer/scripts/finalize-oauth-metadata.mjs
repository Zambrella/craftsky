import { readFile, writeFile } from 'node:fs/promises'
import { isIP } from 'node:net'
import { resolve } from 'node:path'

import { loadEnv } from 'vite'

const mode = process.argv[2]
const projectRoot = process.cwd()
const metadataPath = resolve(
  projectRoot,
  'dist',
  'oauth-client-metadata.json',
)
const canonicalMetadataPath = resolve(
  projectRoot,
  'public',
  'oauth-client-metadata.json',
)

if (mode === 'preview') {
  await writeFile(
    metadataPath,
    `${JSON.stringify(
      {
        preview: true,
        oauthEnabled: false,
      },
      null,
      2,
    )}\n`,
  )
  process.exit(0)
}

if (mode === 'production') {
  const [source, emitted] = await Promise.all([
    readFile(canonicalMetadataPath, 'utf8'),
    readFile(metadataPath, 'utf8'),
  ])
  if (source !== emitted) {
    throw new Error('Emitted production OAuth metadata drifted')
  }
  process.exit(0)
}

if (mode !== 'staging') {
  throw new Error(`Unsupported OAuth metadata mode: ${String(mode)}`)
}

const environment = loadEnv('staging', projectRoot, '')
const rawOrigin = environment.VITE_STAGING_ORIGIN
if (!rawOrigin) {
  throw new Error('VITE_STAGING_ORIGIN is required for a staging build')
}

const parsedOrigin = new URL(rawOrigin)
if (
  parsedOrigin.protocol !== 'https:' ||
  parsedOrigin.origin !== rawOrigin ||
  parsedOrigin.port ||
  parsedOrigin.username ||
  parsedOrigin.password ||
  parsedOrigin.hostname === 'localhost' ||
  isIP(parsedOrigin.hostname.replaceAll(/^\[|\]$/g, '')) !== 0
) {
  throw new Error('VITE_STAGING_ORIGIN must be one canonical HTTPS origin')
}

const clientId =
  environment.VITE_OAUTH_CLIENT_ID ??
  `${parsedOrigin.origin}/oauth-client-metadata.json`
if (clientId !== `${parsedOrigin.origin}/oauth-client-metadata.json`) {
  throw new Error(
    'VITE_OAUTH_CLIENT_ID must be the staging origin metadata document',
  )
}

const metadata = {
  client_id: clientId,
  client_name: 'CraftSky Instagram post importer (staging)',
  client_uri: parsedOrigin.origin,
  logo_uri: `${parsedOrigin.origin}/logo.svg`,
  policy_uri: 'https://craftsky.social/privacy.html',
  tos_uri: 'https://craftsky.social/terms.html',
  redirect_uris: [`${parsedOrigin.origin}/oauth/callback`],
  grant_types: ['authorization_code', 'refresh_token'],
  response_types: ['code'],
  scope:
    'atproto repo:social.craftsky.feed.post?action=create&action=delete ' +
    'blob?accept=image/jpeg&accept=image/png&accept=image/webp',
  token_endpoint_auth_method: 'none',
  application_type: 'web',
  dpop_bound_access_tokens: true,
}

await writeFile(metadataPath, `${JSON.stringify(metadata, null, 2)}\n`)
