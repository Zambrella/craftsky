import '@fontsource/dm-serif-display/400.css'
import '@fontsource/outfit/400.css'
import '@fontsource/outfit/500.css'
import '@fontsource/outfit/700.css'
import '@fontsource/outfit/800.css'
import './styles/app.css'

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import type { Agent } from '@atproto/api'

import {
  App,
  type AuthorizedDestination,
  type ImporterUiServices,
  type PersistedProgressStatus,
  type PublishOutcome,
} from './app/App'
import {
  ImporterAuthService,
  type AuthenticatedPds,
} from './auth/authService'
import { processOAuthPopupCallback } from './auth/popupCallback'
import {
  canonicalLoopbackUrl,
  classifyRuntime,
} from './config/runtime'
import type { ReviewManifest, ReviewPost } from './domain/types'
import type { RepoClient } from './pds/publisher'
import { importerDatabase } from './progress/database'
import { deriveReviewedManifestFingerprint } from './progress/fingerprint'
import { rollbackImport } from './progress/importRunner'
import { publishStoredItem } from './progress/publishStoredItem'
import { ProgressRepository } from './progress/repository'
import {
  applySessionPublishFailure,
  applySessionPublishOutcome,
  rollbackOwnershipCount,
  sessionPublishOutcomeStatus,
  type ImportItemRow,
  type ImportSessionRow,
} from './progress/state'
import {
  createArchiveWorkerClient,
  type ArchiveWorkerClient,
} from './worker/client'
import type { WorkerResponse } from './worker/protocol'

type InspectProgress = Extract<WorkerResponse, { type: 'progress' }>

function visibleProgressStatus(
  status: ImportItemRow['status'],
): PersistedProgressStatus {
  return status === 'remaining' || status === 'running'
    ? 'pending'
    : status
}

function visibleProgressStatuses(
  items: Iterable<ImportItemRow>,
): Readonly<Record<string, PersistedProgressStatus>> {
  return Object.fromEntries(
    [...items].map((item) => [
      item.itemKey,
      visibleProgressStatus(item.status),
    ]),
  )
}

function repoClient(agent: Agent, signal?: AbortSignal): RepoClient {
  return {
    async getRecord(input) {
      const response = await agent.com.atproto.repo.getRecord(input, {
        signal,
      })
      return {
        data: {
          cid: response.data.cid,
          value: response.data.value,
        },
      }
    },
    async uploadBlob(blob, options) {
      const response = await agent.com.atproto.repo.uploadBlob(blob, {
        ...options,
        signal,
      })
      return { data: { blob: response.data.blob } }
    },
    async createRecord(input) {
      const response = await agent.com.atproto.repo.createRecord({
        ...input,
        record: input.record as Record<string, unknown>,
      }, { signal })
      return {
        data: {
          uri: response.data.uri,
          cid: response.data.cid,
        },
      }
    },
    async deleteRecord(input) {
      return agent.com.atproto.repo.deleteRecord(input, { signal })
    },
  }
}

class BrowserImporterServices implements ImporterUiServices {
  readonly runtime = classifyRuntime(
    window.location.origin,
    import.meta.env,
  )

  private readonly worker: ArchiveWorkerClient =
    createArchiveWorkerClient()
  private readonly progress = new ProgressRepository(importerDatabase)
  private authService?: ImporterAuthService
  private authenticated?: AuthenticatedPds
  private manifest?: ReviewManifest
  private session?: ImportSessionRow
  private items = new Map<string, ImportItemRow>()

  async inspect(
    file: File,
    onProgress: (progress: InspectProgress) => void,
  ): Promise<ReviewManifest> {
    const manifest = await this.worker.inspect(file, onProgress)
    this.manifest = manifest
    this.session = undefined
    this.items.clear()
    return manifest
  }

  cancelInspection(): void {
    this.worker.cancel()
  }

  async previewMedia(token: string, signal: AbortSignal): Promise<Blob> {
    const media = await this.worker.sanitize(token, signal)
    return new Blob([media.buffer], { type: media.mime })
  }

  async prepareAuthorization(): Promise<void> {
    if (!this.runtime.writesEnabled) return
    this.authService ??= await ImporterAuthService.load(this.runtime)
  }

  async authorize(input: string): Promise<AuthorizedDestination> {
    if (!this.authService) {
      await this.prepareAuthorization()
    }
    if (!this.authService) throw new Error('previewWritesDisabled')
    const authenticated = await this.authService.signIn(input)
    if (
      this.authenticated?.did !== authenticated.did ||
      (this.session && this.session.did !== authenticated.did)
    ) {
      this.session = undefined
      this.items.clear()
    }
    this.authenticated = authenticated
    return {
      did: this.authenticated.did,
      displayName: this.authenticated.displayName,
      accountLabel: this.authenticated.accountLabel,
      avatarUrl: this.authenticated.avatarUrl,
    }
  }

  async beginImport(
    posts: readonly ReviewPost[],
  ): Promise<{
    readonly ownedCreated: number
    readonly itemStatuses: Readonly<
      Record<string, PersistedProgressStatus>
    >
  }> {
    const authenticated = this.requireAuthenticated()
    const manifest = this.manifest
    if (!manifest) throw new Error('publicationFailed')
    const now = new Date().toISOString()
    const fingerprint = await deriveReviewedManifestFingerprint(posts)

    if (
      this.session?.did !== authenticated.did ||
      this.session.manifestFingerprint !== fingerprint
    ) {
      const existing = await this.progress.findResumable(
        authenticated.did,
        fingerprint,
      )
      this.session = existing?.session
      this.items = new Map(
        (existing?.items ?? []).map((item) => [item.itemKey, item]),
      )
    }

    if (!this.session) {
      this.session = {
        id: crypto.randomUUID(),
        schemaVersion: 1,
        manifestFingerprint: fingerprint,
        did: authenticated.did,
        status: 'running',
        createdAt: now,
        updatedAt: now,
      }
      const rows = posts.map<ImportItemRow>((post) => ({
        sessionId: this.session!.id,
        itemKey: post.itemKey,
        rkey: post.rkey,
        status: 'remaining',
        attempts: 0,
      }))
      await this.progress.createImport(this.session, rows)
      this.items = new Map(rows.map((item) => [item.itemKey, item]))
      return {
        ownedCreated: 0,
        itemStatuses: visibleProgressStatuses(this.items.values()),
      }
    }

    this.session = {
      ...this.session,
      status: 'running',
      updatedAt: now,
    }
    await this.progress.putSession(this.session)

    const newRows = posts
      .filter((post) => !this.items.has(post.itemKey))
      .map<ImportItemRow>((post) => ({
        sessionId: this.session!.id,
        itemKey: post.itemKey,
        rkey: post.rkey,
        status: 'remaining',
        attempts: 0,
      }))
    if (newRows.length > 0) {
      await this.progress.putItems(newRows)
      for (const row of newRows) this.items.set(row.itemKey, row)
    }
    return {
      ownedCreated: rollbackOwnershipCount([...this.items.values()]),
      itemStatuses: visibleProgressStatuses(this.items.values()),
    }
  }

  async publish(
    post: ReviewPost,
    signal: AbortSignal,
  ): Promise<PublishOutcome> {
    const authenticated = this.requireAuthenticated()
    const session = this.session
    const stored = this.items.get(post.itemKey)
    if (!session || !stored) throw new Error('publicationFailed')

    if (signal.aborted) throw new DOMException('Aborted', 'AbortError')

    const running: ImportItemRow = {
      ...stored,
      status: 'running',
      attempts: stored.attempts + 1,
      safeCode: undefined,
    }
    await this.progress.putItem(running)
    this.items.set(post.itemKey, running)

    try {
      const outcome = await publishStoredItem({
        stored,
        repo: repoClient(authenticated.agent, signal),
        did: authenticated.did,
        post,
        sanitize: (token, sanitizeSignal) =>
          this.worker.sanitize(token, sanitizeSignal),
        signal,
      })
      const completed = applySessionPublishOutcome(
        stored,
        running,
        outcome,
      )
      await this.progress.putItem(completed)
      this.items.set(post.itemKey, completed)
      return {
        status: sessionPublishOutcomeStatus(
          completed,
          outcome.status,
        ),
      }
    } catch (caught) {
      const protocol = caught as {
        readonly status?: number
        readonly error?: string
      }
      const authorizationRequired =
        protocol.status === 401 ||
        protocol.error === 'AuthenticationRequired' ||
        protocol.error === 'InvalidToken'
      const failed = applySessionPublishFailure(
        stored,
        running,
        authorizationRequired ? 'oauthDenied' : 'publicationFailed',
      )
      await this.progress.putItem(failed)
      this.items.set(post.itemKey, failed)
      return {
        status: authorizationRequired
          ? 'authorizationRequired'
          : 'failed',
      }
    }
  }

  async pauseImport(): Promise<void> {
    await this.updateSessionStatus('paused')
  }

  async finishImport(): Promise<void> {
    await this.updateSessionStatus('complete')
  }

  async signOut(): Promise<void> {
    if (!this.authenticated || !this.authService) return
    await this.authService.signOut(this.authenticated)
    this.authenticated = undefined
  }

  async rollback(): Promise<{
    readonly deleted: number
    readonly absent: number
    readonly conflicts: number
    readonly failed: number
    readonly remainingOwned: number
    readonly itemStatuses: Readonly<
      Record<string, PersistedProgressStatus>
    >
  }> {
    const authenticated = this.requireAuthenticated()
    const session = this.session
    if (!session) {
      return {
        deleted: 0,
        absent: 0,
        conflicts: 0,
        failed: 0,
        remainingOwned: 0,
        itemStatuses: {},
      }
    }
    if (session.did !== authenticated.did) {
      throw new Error('publicationFailed')
    }
    const result = await rollbackImport({
      repository: this.progress,
      repo: repoClient(authenticated.agent),
      stored: {
        session,
        items: [...this.items.values()],
      },
    })
    const remainingOwned = rollbackOwnershipCount(result.items)
    const itemStatuses = visibleProgressStatuses(result.items)
    if (result.stored) {
      this.session = result.stored.session
      this.items = new Map(
        result.stored.items.map((item) => [item.itemKey, item]),
      )
    } else {
      this.session = undefined
      this.items.clear()
    }
    return {
      ...result.summary,
      remainingOwned,
      itemStatuses,
    }
  }

  async clearHistory(): Promise<void> {
    if (!this.session) return
    await this.progress.clearSession(this.session.id)
    this.session = undefined
    this.items.clear()
  }

  private requireAuthenticated(): AuthenticatedPds {
    if (!this.authenticated) throw new Error('publicationFailed')
    return this.authenticated
  }

  private async updateSessionStatus(
    status: ImportSessionRow['status'],
  ): Promise<void> {
    if (!this.session) return
    this.session = {
      ...this.session,
      status,
      updatedAt: new Date().toISOString(),
    }
    await this.progress.putSession(this.session)
  }
}

async function bootstrap(): Promise<void> {
  const canonicalUrl = canonicalLoopbackUrl(window.location.href)
  if (canonicalUrl) {
    window.location.replace(canonicalUrl)
    return
  }
  const rootElement = document.getElementById('root')
  if (!rootElement) throw new Error('rootElementMissing')
  const runtime = classifyRuntime(window.location.origin, import.meta.env)
  if (
    await processOAuthPopupCallback(
      runtime,
      {
        search: window.location.search,
        hash: window.location.hash,
      },
    )
  ) {
    return
  }
  createRoot(rootElement).render(
    <StrictMode>
      <App services={new BrowserImporterServices()} />
    </StrictMode>,
  )
}

void bootstrap()
