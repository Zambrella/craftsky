import '../src/styles/app.css'

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import {
  App,
  type ImporterUiServices,
  type PublishOutcome,
} from '../src/app/App'
import type { ReviewManifest } from '../src/domain/types'

const calls = {
  authorize: 0,
  beginImport: 0,
  publish: 0,
  finishImport: 0,
  rollback: 0,
  clearHistory: 0,
  signOut: 0,
}

declare global {
  interface Window {
    __importerHarness: typeof calls
  }
}

window.__importerHarness = calls

const manifest: ReviewManifest = {
  schemaVersion: 1,
  fingerprint: 'browser-harness-manifest',
  posts: [
    {
      itemKey: 'browser-item',
      rkey: '3kexampletid2',
      createdAt: '2024-01-02T03:04:05.000Z',
      caption: 'Synthetic browser caption #making',
      initialCaption: 'Synthetic browser caption #making',
      media: [
        {
          token: 'browser-media',
          kind: 'image',
          mime: 'image/png',
          width: 1,
          height: 1,
          selected: true,
        },
      ],
      warnings: [],
      selected: true,
      needsTextOnlyConfirmation: false,
      textOnlyConfirmed: false,
    },
  ],
  skipped: [],
  counts: {
    selectedPosts: 1,
    selectedImages: 1,
    transformedPosts: 0,
    warningPosts: 0,
    skippedPosts: 0,
  },
}

const onePixelPng = Uint8Array.from(
  atob(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
  ),
  (character) => character.charCodeAt(0),
)

const services: ImporterUiServices = {
  runtime: {
    mode: 'local',
    origin: window.location.origin,
    redirectUri: `${window.location.origin}/oauth/callback`,
    clientId: 'http://localhost',
    writesEnabled: true,
  },
  inspect: () => Promise.resolve(manifest),
  previewMedia: () =>
    Promise.resolve(new Blob([onePixelPng], { type: 'image/png' })),
  cancelInspection: () => undefined,
  authorize: () => {
    calls.authorize += 1
    return Promise.resolve({ did: 'did:plc:browser-harness' })
  },
  beginImport: () => {
    calls.beginImport += 1
    return Promise.resolve({ ownedCreated: 0 })
  },
  publish: (): Promise<PublishOutcome> => {
    calls.publish += 1
    return Promise.resolve({
      status: calls.publish === 1 ? 'failed' : 'created',
    })
  },
  finishImport: () => {
    calls.finishImport += 1
    return Promise.resolve()
  },
  pauseImport: () => Promise.resolve(),
  rollback: () => {
    calls.rollback += 1
    return Promise.resolve({
      deleted: 0,
      absent: 1,
      conflicts: 0,
      failed: 0,
      remainingOwned: 0,
      itemStatuses: {
        'browser-item': 'rollbackAbsent',
      },
    })
  },
  clearHistory: () => {
    calls.clearHistory += 1
    return Promise.resolve()
  },
  signOut: () => {
    calls.signOut += 1
    return Promise.resolve()
  },
}

const root = document.getElementById('root')
if (!root) throw new Error('rootElementMissing')
createRoot(root).render(
  <StrictMode>
    <App services={services} />
  </StrictMode>,
)
