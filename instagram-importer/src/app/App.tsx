import { useMemo, useRef, useState } from 'react'

import type {
  ReviewManifest,
  ReviewPost,
} from '../domain/types'
import type { RuntimePolicy } from '../config/runtime'
import { calculateReviewCounts } from '../review/reviewState'
import type { WorkerResponse } from '../worker/protocol'
import {
  ReviewList,
  SkippedReviewList,
} from './components/ReviewList'

type InspectProgress = Extract<WorkerResponse, { type: 'progress' }>

export interface AuthorizedDestination {
  readonly did: string
  readonly ownedCreated?: number
}

export type PublishOutcomeStatus =
  | 'created'
  | 'alreadyExisting'
  | 'collision'
  | 'failed'
  | 'authorizationRequired'

export interface PublishOutcome {
  readonly status: PublishOutcomeStatus
}

export interface ImportRecoveryState {
  readonly ownedCreated: number
  readonly itemStatuses?: Readonly<Record<string, PersistedProgressStatus>>
}

export type PersistedProgressStatus =
  | 'pending'
  | Exclude<PublishOutcomeStatus, 'authorizationRequired'>
  | 'rolledBack'
  | 'rollbackAbsent'
  | 'rollbackConflict'
  | 'rollbackFailed'

type VisibleProgressStatus = PersistedProgressStatus | 'publishing'

export interface ImporterUiServices {
  readonly runtime: RuntimePolicy
  inspect(
    file: File,
    onProgress: (progress: InspectProgress) => void,
  ): Promise<ReviewManifest>
  previewMedia?(token: string, signal: AbortSignal): Promise<Blob>
  cancelInspection(): void
  prepareAuthorization?(): Promise<void>
  authorize(input: string): Promise<AuthorizedDestination>
  publish(post: ReviewPost, signal: AbortSignal): Promise<PublishOutcome>
  beginImport?(
    posts: readonly ReviewPost[],
  ): Promise<ImportRecoveryState | void>
  finishImport?(): Promise<void>
  pauseImport?(): Promise<void>
  signOut?(): Promise<void>
  rollback?(): Promise<{
    readonly deleted: number
    readonly absent: number
    readonly conflicts: number
    readonly failed: number
    readonly remainingOwned?: number
    readonly itemStatuses?: Readonly<
      Record<string, PersistedProgressStatus>
    >
  }>
  clearHistory?(): Promise<void>
}

type AppStage =
  | 'select'
  | 'processing'
  | 'review'
  | 'authorize'
  | 'confirm'
  | 'publishing'
  | 'paused'
  | 'complete'

interface ResultCounts {
  readonly created: number
  readonly alreadyExisting: number
  readonly collision: number
  readonly failed: number
}

const emptyResults: ResultCounts = {
  created: 0,
  alreadyExisting: 0,
  collision: 0,
  failed: 0,
}

const PROGRESS_WINDOW_SIZE = 24

const progressCopy: Record<VisibleProgressStatus, string> = {
  pending: 'Waiting',
  publishing: 'Publishing',
  created: 'Created',
  alreadyExisting: 'Already there',
  collision: 'Left unchanged',
  failed: 'Failed',
  rolledBack: 'Removed',
  rollbackAbsent: 'Already absent',
  rollbackConflict: 'Changed and kept',
  rollbackFailed: 'Removal failed',
}

const phaseCopy: Record<InspectProgress['phase'], string> = {
  directory: 'Checking the archive',
  metadata: 'Reading post information',
  mediaPreflight: 'Checking images',
  sanitizing: 'Preparing an image',
}

const errorCopy: Record<string, string> = {
  archiveUnsupported:
    'This export could not be read. Try requesting a Posts-only JSON export from Instagram.',
  archiveEntryLimit:
    'This archive contains too many files. Try a Posts-only Instagram export.',
  centralDirectoryLimit:
    'This archive’s file listing is too large. Try a Posts-only Instagram export.',
  candidateMetadataLimit:
    'The post information is too large to process safely. Try a Posts-only Instagram export.',
  browserUnsupported:
    'This browser cannot safely process the export images. Try the latest Chrome, Edge, or Firefox on a desktop computer.',
  oauthDenied:
    'Authorization was cancelled or denied. Your local review is still here.',
  oauthAuthorityMismatch:
    'This PDS did not grant the exact narrow permissions the importer needs.',
  notCraftskyMember:
    'This account does not contain a CraftSky profile. Join CraftSky before importing.',
  previewWritesDisabled:
    'This preview cannot connect to a real PDS or publish posts.',
  publicationFailed:
    'A post could not be published. You can retry the failed posts.',
  authorizationRequired:
    'Your PDS authorization needs to be renewed. Return to review and connect again; completed posts will not be repeated.',
}

type SafeErrorContext =
  | 'localOnly'
  | 'publication'
  | 'rollback'
  | 'history'
  | 'signOut'

const fallbackErrorCopy: Record<SafeErrorContext, string> = {
  localOnly:
    'Something went wrong locally. No archive content was sent anywhere.',
  publication:
    'Something went wrong while saving import progress. Review the status below before retrying or clearing local history.',
  rollback:
    'Rollback could not be completed. Review the status below before retrying or clearing local history.',
  history:
    'Local history could not be cleared. Resume and rollback information remains on this device.',
  signOut:
    'Your PDS session could not be cleared and may still be active. Local import progress remains on this device.',
}

function safeErrorMessage(
  error: unknown,
  context: SafeErrorContext = 'localOnly',
): string {
  const code = error instanceof Error ? error.message : ''
  return errorCopy[code] ?? fallbackErrorCopy[context]
}

function isInspectionCancellation(error: unknown): boolean {
  return (
    error instanceof Error &&
    error.message === 'operationCanceled'
  )
}

function shortDid(did: string): string {
  if (did.length <= 34) return did
  return `${did.slice(0, 21)}…${did.slice(-10)}`
}

function counted(
  value: number,
  singular: string,
  plural = `${singular}s`,
): string {
  return `${value.toLocaleString('en-GB')} ${
    value === 1 ? singular : plural
  }`
}

function Header(): React.JSX.Element {
  return (
    <header className="site-header">
      <a href="/" className="brand" aria-label="CraftSky importer home">
        <img src="/logo.svg" alt="" />
      </a>
      <span className="site-header__product">Instagram post importer</span>
    </header>
  )
}

function StepStrip({ stage }: { readonly stage: AppStage }): React.JSX.Element {
  const active =
    stage === 'select' || stage === 'processing'
      ? 1
      : stage === 'review'
        ? 2
        : stage === 'authorize' || stage === 'confirm'
          ? 3
          : 4
  return (
    <ol className="step-strip" aria-label="Import progress">
      {['Choose export', 'Review', 'Connect', 'Import'].map(
        (label, index) => (
          <li
            key={label}
            className={
              index + 1 === active
                ? 'step-strip__active'
                : index + 1 < active
                  ? 'step-strip__complete'
                  : ''
            }
            aria-current={index + 1 === active ? 'step' : undefined}
          >
            <span>{index + 1}</span>
            {label}
          </li>
        ),
      )}
    </ol>
  )
}

function Stat({
  value,
  label,
}: {
  readonly value: number
  readonly label: string
}): React.JSX.Element {
  return (
    <div className="stat" aria-label={`${value} ${label}`}>
      <strong>{value.toLocaleString('en-GB')}</strong>
      <span>{label}</span>
    </div>
  )
}

function PostProgressList({
  posts,
  statuses,
  activeIndex,
}: {
  readonly posts: readonly ReviewPost[]
  readonly statuses: Readonly<Record<string, VisibleProgressStatus>>
  readonly activeIndex: number
}): React.JSX.Element {
  const pageCount = Math.max(
    1,
    Math.ceil(posts.length / PROGRESS_WINDOW_SIZE),
  )
  const activePage = Math.min(
    pageCount - 1,
    Math.max(0, Math.floor(activeIndex / PROGRESS_WINDOW_SIZE)),
  )
  const [page, setPage] = useState<number>()
  const currentPage = Math.min(page ?? activePage, pageCount - 1)
  const start = currentPage * PROGRESS_WINDOW_SIZE
  const visiblePosts = posts.slice(start, start + PROGRESS_WINDOW_SIZE)

  if (posts.length === 0) return <></>

  return (
    <section
      className="post-progress"
      aria-label="Per-post publication progress"
    >
      <div className="post-progress__heading">
        <h2>Post progress</h2>
        {pageCount > 1 && (
          <nav aria-label="Post progress pages">
            <button
              type="button"
              className="text-button"
              disabled={currentPage === 0}
              onClick={() =>
                setPage((value) =>
                  Math.max(0, (value ?? activePage) - 1),
                )
              }
            >
              Previous
            </button>
            <span>
              Page {currentPage + 1} of {pageCount}
            </span>
            <button
              type="button"
              className="text-button"
              disabled={currentPage >= pageCount - 1}
              onClick={() =>
                setPage((value) =>
                  Math.min(
                    pageCount - 1,
                    (value ?? activePage) + 1,
                  ),
                )
              }
            >
              Next
            </button>
          </nav>
        )}
      </div>
      <ol start={start + 1}>
        {visiblePosts.map((post, index) => {
          const status = statuses[post.itemKey] ?? 'pending'
          return (
            <li key={post.itemKey}>
              <span>
                Post {start + index + 1}
                <small>
                  {new Intl.DateTimeFormat('en-GB', {
                    day: 'numeric',
                    month: 'short',
                    year: 'numeric',
                  }).format(new Date(post.createdAt))}
                </small>
              </span>
              <strong data-status={status}>{progressCopy[status]}</strong>
            </li>
          )
        })}
      </ol>
    </section>
  )
}

export function App({
  services,
}: {
  readonly services: ImporterUiServices
}): React.JSX.Element {
  const [stage, setStage] = useState<AppStage>('select')
  const [acknowledged, setAcknowledged] = useState(false)
  const [progress, setProgress] = useState<InspectProgress>()
  const [manifest, setManifest] = useState<ReviewManifest>()
  const [posts, setPosts] = useState<ReviewPost[]>([])
  const [accountInput, setAccountInput] = useState('')
  const [authorizationPrepared, setAuthorizationPrepared] = useState(
    services.prepareAuthorization === undefined,
  )
  const [destination, setDestination] =
    useState<AuthorizedDestination>()
  const [destinationConfirmed, setDestinationConfirmed] = useState(false)
  const [error, setError] = useState<string>()
  const [processed, setProcessed] = useState(0)
  const [publishTotal, setPublishTotal] = useState(0)
  const [results, setResults] = useState<ResultCounts>(emptyResults)
  const [failedKeys, setFailedKeys] = useState<Set<string>>(new Set())
  const [itemStatuses, setItemStatuses] = useState<
    Record<string, VisibleProgressStatus>
  >({})
  const [rollbackMessage, setRollbackMessage] = useState<string>()
  const [hasRollbackOwnership, setHasRollbackOwnership] =
    useState(false)
  const stopBeforeNext = useRef(false)
  const publishController = useRef<AbortController | null>(null)

  const counts = useMemo(
    () => calculateReviewCounts(posts, manifest?.skipped.length ?? 0),
    [manifest?.skipped.length, posts],
  )
  const selectedPosts = useMemo(
    () => posts.filter((post) => post.selected),
    [posts],
  )
  const loadPreview = useMemo(
    () =>
      services.previewMedia
        ? (token: string, signal: AbortSignal) =>
            services.previewMedia!(token, signal)
        : undefined,
    [services],
  )
  const needsConfirmation = selectedPosts.some(
    (post) =>
      post.needsTextOnlyConfirmation && !post.textOnlyConfirmed,
  )

  async function chooseFile(file: File | undefined): Promise<void> {
    if (!file || !acknowledged) return
    setError(undefined)
    setProgress(undefined)
    setStage('processing')
    try {
      const inspected = await services.inspect(file, setProgress)
      setManifest(inspected)
      setPosts([...inspected.posts])
      setStage('review')
    } catch (caught) {
      if (isInspectionCancellation(caught)) {
        setError(undefined)
        setProgress(undefined)
        setStage('select')
        return
      }
      setError(safeErrorMessage(caught))
      setStage('select')
    }
  }

  async function authorize(): Promise<void> {
    if (!accountInput.trim()) return
    setError(undefined)
    try {
      const authenticated = await services.authorize(accountInput.trim())
      setDestination(authenticated)
      setDestinationConfirmed(false)
      setStage('confirm')
    } catch (caught) {
      setError(safeErrorMessage(caught))
    }
  }

  function enterAuthorization(): void {
    setStage('authorize')
    setError(undefined)
    if (!services.prepareAuthorization) {
      setAuthorizationPrepared(true)
      return
    }
    setAuthorizationPrepared(false)
    void services
      .prepareAuthorization()
      .then(() => setAuthorizationPrepared(true))
      .catch((caught) => {
        setAuthorizationPrepared(false)
        setError(safeErrorMessage(caught))
      })
  }

  async function publishBatch(
    batch: readonly ReviewPost[],
    options: {
      readonly reset?: boolean
      readonly retry?: boolean
    } = {},
  ): Promise<void> {
    setStage('publishing')
    setError(undefined)
    stopBeforeNext.current = false
    const controller = new AbortController()
    publishController.current = controller
    if (options.reset || options.retry) {
      setProcessed(0)
      setPublishTotal(batch.length)
    }
    if (options.reset) {
      try {
        const recovery = await services.beginImport?.(batch)
        setItemStatuses(
          Object.fromEntries(
            batch.map((post) => [
              post.itemKey,
              recovery?.itemStatuses?.[post.itemKey] ?? 'pending',
            ]),
          ),
        )
        setHasRollbackOwnership(
          (recovery?.ownedCreated ?? 0) > 0,
        )
      } catch (caught) {
        publishController.current = null
        setError(safeErrorMessage(caught))
        setStage('confirm')
        return
      }
    }

    let nextResults = options.reset
      ? { ...emptyResults }
      : options.retry
        ? { ...results, failed: 0 }
        : { ...results }
    const nextFailed =
      options.reset || options.retry
        ? new Set<string>()
        : new Set(failedKeys)
    if (options.reset) setResults(emptyResults)
    if (options.retry) setResults(nextResults)

    for (const post of batch) {
      if (stopBeforeNext.current || controller.signal.aborted) {
        break
      }
      setItemStatuses((value) => ({
        ...value,
        [post.itemKey]: 'publishing',
      }))
      try {
        const outcome = await services.publish(post, controller.signal)
        if (outcome.status === 'authorizationRequired') {
          nextResults = {
            ...nextResults,
            failed: nextResults.failed + 1,
          }
          nextFailed.add(post.itemKey)
          setItemStatuses((value) => ({
            ...value,
            [post.itemKey]: 'failed',
          }))
          setError(errorCopy.authorizationRequired)
          stopBeforeNext.current = true
        } else {
          const completedStatus: Exclude<
            PublishOutcomeStatus,
            'authorizationRequired'
          > = outcome.status
          nextResults = {
            ...nextResults,
            [completedStatus]: nextResults[completedStatus] + 1,
          }
          if (completedStatus === 'failed') {
            nextFailed.add(post.itemKey)
          } else {
            nextFailed.delete(post.itemKey)
          }
          setItemStatuses((value) => ({
            ...value,
            [post.itemKey]: completedStatus,
          }))
          if (completedStatus === 'created') {
            setHasRollbackOwnership(true)
          }
        }
      } catch {
        nextResults = {
          ...nextResults,
          failed: nextResults.failed + 1,
        }
        nextFailed.add(post.itemKey)
        setItemStatuses((value) => ({
          ...value,
          [post.itemKey]: 'failed',
        }))
      }
      setProcessed((value) => value + 1)
      setResults(nextResults)
      setFailedKeys(new Set(nextFailed))
    }

    if (stopBeforeNext.current || controller.signal.aborted) {
      try {
        await services.pauseImport?.()
      } catch (caught) {
        setError(safeErrorMessage(caught, 'publication'))
      }
      setStage('paused')
      publishController.current = null
      return
    }
    try {
      await services.finishImport?.()
      setStage('complete')
      publishController.current = null
    } catch (caught) {
      publishController.current = null
      setError(safeErrorMessage(caught, 'publication'))
      setStage('paused')
    }
  }

  async function rollbackImport(): Promise<void> {
    if (!services.rollback) return
    if (
      !window.confirm(
        'Remove tracked posts that still match this import? A separately recreated identical record at the same address cannot be distinguished.',
      )
    ) {
      return
    }
    setError(undefined)
    try {
      const outcome = await services.rollback()
      setHasRollbackOwnership((outcome.remainingOwned ?? 0) > 0)
      if (outcome.itemStatuses) {
        setItemStatuses((value) => ({
          ...value,
          ...outcome.itemStatuses,
        }))
      }
      setRollbackMessage(
        `${outcome.deleted} removed · ${outcome.absent} already absent · ${outcome.conflicts} changed and kept · ${outcome.failed} failed`,
      )
      if (stage === 'paused') setStage('complete')
    } catch (caught) {
      setError(safeErrorMessage(caught, 'rollback'))
    }
  }

  function reset(): void {
    services.cancelInspection()
    setStage('select')
    setAcknowledged(false)
    setManifest(undefined)
    setPosts([])
    setProgress(undefined)
    setDestination(undefined)
    setDestinationConfirmed(false)
    setAccountInput('')
    setAuthorizationPrepared(
      services.prepareAuthorization === undefined,
    )
    setError(undefined)
    setProcessed(0)
    setPublishTotal(0)
    setResults(emptyResults)
    setFailedKeys(new Set())
    setItemStatuses({})
    setRollbackMessage(undefined)
    setHasRollbackOwnership(false)
    publishController.current?.abort()
    publishController.current = null
  }

  return (
    <>
      <Header />
      <main className="page-shell">
        <StepStrip stage={stage} />

        {services.runtime.mode === 'preview' && (
          <aside className="notice notice--preview">
            <strong>Safe preview</strong>
            <span>
              You can explore the local review flow, but this build cannot
              connect to a real account or publish.
            </span>
          </aside>
        )}

        {error && (
          <div className="notice notice--error" role="alert">
            <strong>We couldn’t continue</strong>
            <span>{error}</span>
          </div>
        )}

        {stage === 'select' && (
          <section className="hero-grid">
            <div className="hero-copy">
              <p className="eyebrow">Bring your making history with you</p>
              <h1>Import your Instagram posts into CraftSky</h1>
              <p className="hero-copy__lede">
                Review historical posts on this device, then publish only
                what you choose directly to your own PDS.
              </p>
              <ul className="privacy-points">
                <li>Your ZIP is processed in this browser.</li>
                <li>No archive, caption, or image is sent to CraftSky.</li>
                <li>Messages, followers, likes, and profile data are ignored.</li>
              </ul>
            </div>

            <div className="paper-card upload-card">
              <div className="upload-card__number" aria-hidden="true">
                01
              </div>
              <h2>Choose your Instagram export</h2>
              <p>
                For the quickest import, request <strong>Posts only</strong> in
                JSON format. A full export also works, but takes longer to
                inspect.
              </p>

              <label className="confirmation-check">
                <input
                  type="checkbox"
                  checked={acknowledged}
                  onChange={(event) =>
                    setAcknowledged(event.currentTarget.checked)
                  }
                />
                <span>
                  I understand selected posts and images will become public
                  <small>
                    You will review everything before connecting an account.
                  </small>
                </span>
              </label>

              <label
                className={`file-picker ${
                  acknowledged ? '' : 'file-picker--disabled'
                }`}
              >
                <input
                  type="file"
                  accept=".zip,application/zip"
                  disabled={!acknowledged}
                  onChange={(event) => {
                    void chooseFile(event.currentTarget.files?.[0])
                    event.currentTarget.value = ''
                  }}
                />
                <span className="file-picker__icon" aria-hidden="true">
                  ↑
                </span>
                <span>
                  <strong>Select export ZIP</strong>
                  <small>
                    The file stays on this device and is not uploaded.
                  </small>
                </span>
              </label>
            </div>
          </section>
        )}

        {stage === 'processing' && (
          <section className="centred-panel paper-card" aria-live="polite">
            <div className="working-mark" aria-hidden="true">
              <span />
              <span />
              <span />
            </div>
            <p className="eyebrow">Working locally</p>
            <h1>
              {progress ? phaseCopy[progress.phase] : 'Opening your export'}
            </h1>
            <p className="muted">
              {progress
                ? `${progress.completed.toLocaleString('en-GB')}${
                    progress.total
                      ? ` of ${progress.total.toLocaleString('en-GB')}`
                      : ''
                  } checked`
                : 'This can take a moment for a full Instagram export.'}
            </p>
            {progress?.total && (
              <progress
                max={progress.total}
                value={progress.completed}
                aria-label="Archive processing progress"
              />
            )}
            <button
              type="button"
              className="secondary-button"
              onClick={() => {
                services.cancelInspection()
                setStage('select')
              }}
            >
              Cancel
            </button>
          </section>
        )}

        {stage === 'review' && manifest && (
          <section className="review-page">
            <div className="section-heading">
              <div>
                <p className="eyebrow">Review locally</p>
                <h1>Your historical posts are ready to review</h1>
                <p>
                  We found only supported post information. Your raw export
                  remains on this device.
                </p>
              </div>
              <div className="privacy-seal">
                <span aria-hidden="true">⌂</span>
                <strong>Local only</strong>
                <small>No account connected yet</small>
              </div>
            </div>

            <div className="stats-grid" aria-label="Review totals">
              <Stat value={counts.selectedPosts} label="posts selected" />
              <Stat value={counts.selectedImages} label="images selected" />
              <Stat value={counts.transformedPosts} label="transformed" />
              <Stat value={counts.warningPosts} label="with warnings" />
              <Stat value={counts.skippedPosts} label="skipped" />
            </div>

            <ReviewList
              posts={posts}
              onChange={setPosts}
              loadPreview={loadPreview}
            />
            <SkippedReviewList skipped={manifest.skipped} />

            <div className="sticky-action">
              <div>
                <strong>
                  {counted(counts.selectedPosts, 'post')} ·{' '}
                  {counted(counts.selectedImages, 'image')}
                </strong>
                <span>
                  {needsConfirmation
                    ? 'Confirm every selected text-only post before continuing.'
                    : 'Nothing is published until the final confirmation.'}
                </span>
              </div>
              <button
                type="button"
                className="primary-button"
                disabled={
                  counts.selectedPosts === 0 || needsConfirmation
                }
                onClick={enterAuthorization}
              >
                Connect your PDS
              </button>
            </div>
          </section>
        )}

        {stage === 'authorize' && (
          <section className="narrow-panel">
            <p className="eyebrow">Connect your account</p>
            <h1>Where should these posts live?</h1>
            <p className="lede">
              Sign in with the handle or DID for your CraftSky account. You’ll
              authorize only post creation, post deletion, and supported image
              uploads.
            </p>
            <form
              className="paper-card auth-card"
              onSubmit={(event) => {
                event.preventDefault()
                void authorize()
              }}
            >
              <label className="field-label">
                <span>Handle or DID</span>
                <input
                  type="text"
                  autoComplete="username"
                  spellCheck={false}
                  value={accountInput}
                  placeholder="you.example.com"
                  onChange={(event) =>
                    setAccountInput(event.currentTarget.value)
                  }
                />
              </label>
              <p className="helper-copy">
                Handles are resolved through{' '}
                <span className="nowrap">bsky.social</span>. This importer
                never asks for your password or app password.
              </p>
              {!authorizationPrepared &&
                services.runtime.writesEnabled && (
                  <p className="helper-copy" role="status">
                    Preparing the secure PDS connection…
                  </p>
                )}
              <div className="button-row">
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => setStage('review')}
                >
                  Back to review
                </button>
                <button
                  type="submit"
                  className="primary-button"
                  disabled={
                    !accountInput.trim() ||
                    !authorizationPrepared ||
                    !services.runtime.writesEnabled
                  }
                >
                  Authorize with your PDS
                </button>
              </div>
            </form>
          </section>
        )}

        {stage === 'confirm' && destination && (
          <section className="narrow-panel">
            <p className="eyebrow">Final check</p>
            <h1>Ready to publish to your PDS</h1>
            <p className="lede">
              Check the destination and exact public content before the first
              image or post is written.
            </p>

            <div className="paper-card destination-card">
              <div className="destination-card__account">
                <span className="account-avatar" aria-hidden="true">
                  ✓
                </span>
                <span>
                  <small>Authenticated account</small>
                  <strong title={destination.did}>
                    {shortDid(destination.did)}
                  </strong>
                </span>
              </div>
              <div className="destination-card__counts">
                <Stat value={counts.selectedPosts} label="public posts" />
                <Stat value={counts.selectedImages} label="public images" />
              </div>
              <p>
                Posts keep their original dates and show a subtle “Imported
                from Instagram” label. Their original backfill will not appear
                in home timelines or trigger notifications.
              </p>
              <label className="confirmation-check">
                <input
                  type="checkbox"
                  checked={destinationConfirmed}
                  onChange={(event) =>
                    setDestinationConfirmed(event.currentTarget.checked)
                  }
                />
                <span>
                  Publish these selected posts to this account
                  <small>
                    Publication starts only when you press the button below.
                  </small>
                </span>
              </label>
              <div className="button-row">
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => {
                    if (!services.signOut) {
                      setDestination(undefined)
                      setStage('authorize')
                      return
                    }
                    void services
                      .signOut()
                      .then(() => {
                        setDestination(undefined)
                        setStage('authorize')
                      })
                      .catch((caught) =>
                        setError(safeErrorMessage(caught, 'signOut')),
                      )
                  }}
                >
                  Use another account
                </button>
                <button
                  type="button"
                  className="primary-button"
                  disabled={!destinationConfirmed}
                  onClick={() => {
                    setProcessed(0)
                    setResults(emptyResults)
                    setFailedKeys(new Set())
                    setPublishTotal(selectedPosts.length)
                    void publishBatch(selectedPosts, { reset: true })
                  }}
                >
                  Import {counted(counts.selectedPosts, 'post')}
                </button>
              </div>
            </div>
          </section>
        )}

        {(stage === 'publishing' || stage === 'paused') && (
          <section className="centred-panel paper-card" aria-live="polite">
            <div
              className={`working-mark ${
                stage === 'paused' ? 'working-mark--paused' : ''
              }`}
              aria-hidden="true"
            >
              <span />
              <span />
              <span />
            </div>
            <p className="eyebrow">
              {stage === 'paused' ? 'Import paused' : 'Publishing'}
            </p>
            <h1>
              {processed.toLocaleString('en-GB')} of{' '}
              {counted(publishTotal, 'post')} processed
            </h1>
            <p className="muted">
              Posts are processed one at a time. Completed posts will not be
              repeated.
            </p>
            <progress
              max={Math.max(publishTotal, 1)}
              value={processed}
              aria-label="Publication progress"
            />
            <div className="mini-results">
              <span>{results.created} created</span>
              <span>{results.alreadyExisting} already there</span>
              <span>{results.collision} conflicts</span>
              <span>{results.failed} failed</span>
            </div>
            <PostProgressList
              posts={selectedPosts}
              statuses={itemStatuses}
              activeIndex={Math.max(0, processed - 1)}
            />
            {stage === 'publishing' ? (
              <button
                type="button"
                className="secondary-button"
                onClick={() => {
                  stopBeforeNext.current = true
                  publishController.current?.abort()
                }}
              >
                Pause import
              </button>
            ) : (
              <div className="button-row">
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => setStage('review')}
                >
                  Return to review
                </button>
                <button
                  type="button"
                  className="primary-button"
                  onClick={() =>
                    void publishBatch(
                      selectedPosts.slice(processed),
                    )
                  }
                >
                  Continue import
                </button>
                {services.rollback &&
                  destination &&
                  hasRollbackOwnership && (
                    <button
                      type="button"
                      className="danger-button"
                      onClick={() => void rollbackImport()}
                    >
                      Roll back this partial import
                    </button>
                  )}
              </div>
            )}
          </section>
        )}

        {stage === 'complete' && (
          <section className="narrow-panel completion-panel">
            <div className="completion-mark" aria-hidden="true">
              ✓
            </div>
            <p className="eyebrow">Import finished</p>
            <h1>Your historical posts have been processed</h1>
            <p className="lede">
              Created posts are visible on your CraftSky profile with their
              original dates.
            </p>
            <div className="stats-grid stats-grid--results">
              <Stat value={results.created} label="created" />
              <Stat value={results.alreadyExisting} label="already there" />
              <Stat value={results.collision} label="left unchanged" />
              <Stat value={results.failed} label="failed" />
            </div>
            <PostProgressList
              posts={selectedPosts}
              statuses={itemStatuses}
              activeIndex={Math.max(0, selectedPosts.length - 1)}
            />
            {failedKeys.size > 0 && (
              <div className="notice notice--warning">
                <strong>
                  {counted(failedKeys.size, 'post')} need another try
                </strong>
                <span>
                  Successfully created posts will not be published twice.
                </span>
                <button
                  type="button"
                  className="text-button"
                  onClick={() => {
                    setProcessed(0)
                    setResults((value) => ({ ...value, failed: 0 }))
                    void publishBatch(
                      selectedPosts.filter((post) =>
                        failedKeys.has(post.itemKey),
                      ),
                      { retry: true },
                    )
                  }}
                >
                  Retry failed posts
                </button>
              </div>
            )}
            {rollbackMessage && (
              <div className="notice" role="status">
                {rollbackMessage}
              </div>
            )}
            <div className="button-row button-row--centre">
              <button
                type="button"
                className="primary-button"
                onClick={reset}
              >
                Import another export
              </button>
              {services.rollback &&
                destination &&
                hasRollbackOwnership && (
                <button
                  type="button"
                  className="danger-button"
                  onClick={() => void rollbackImport()}
                >
                  Roll back this import
                </button>
              )}
              {services.clearHistory && (
                <button
                  type="button"
                  className="text-button"
                  onClick={() => {
                    if (!services.clearHistory) return
                    if (
                      !window.confirm(
                        'Clear local progress? You will lose resume and bulk rollback for this import.',
                      )
                    ) {
                      return
                    }
                    void services
                      .clearHistory()
                      .then(() => {
                        setHasRollbackOwnership(false)
                        setRollbackMessage(
                          'Local progress and rollback history cleared.',
                        )
                      })
                      .catch((caught) =>
                        setError(safeErrorMessage(caught, 'history')),
                      )
                  }}
                >
                  Clear local history
                </button>
              )}
              {services.signOut && destination && (
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => {
                    if (!services.signOut) return
                    void services
                      .signOut()
                      .then(() => {
                        setDestination(undefined)
                        setHasRollbackOwnership(false)
                        setRollbackMessage(
                          'Signed out. Local progress and rollback history remain on this device.',
                        )
                      })
                      .catch((caught) =>
                        setError(safeErrorMessage(caught, 'signOut')),
                      )
                  }}
                >
                  Sign out of PDS
                </button>
              )}
            </div>
            <p className="history-note">
              Local progress is retained so interrupted imports can be resumed
              and unchanged created posts can be rolled back. Clearing it also
              removes those recovery options.
            </p>
          </section>
        )}
      </main>
      <footer className="site-footer">
        <span>CraftSky</span>
        <span>Archive processing stays on this device.</span>
      </footer>
    </>
  )
}
