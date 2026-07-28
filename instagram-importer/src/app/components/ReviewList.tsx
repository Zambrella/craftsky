import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'

import type {
  ReviewPost,
  SafeSkipCode,
  SkippedPost,
  UserFacingWarningCode,
} from '../../domain/types'
import { isUserFacingWarningCode } from '../../domain/types'
import {
  confirmTextOnly,
  setPostSelectionByKey,
  updateCaption,
  updateMediaSelection,
  updatePostSelection,
} from '../../review/reviewState'

export type ReviewFilter = 'all' | UserFacingWarningCode

const REVIEW_VIEWPORT_HEIGHT = 720
const SKIPPED_VIEWPORT_HEIGHT = 480
const REVIEW_ROW_ESTIMATE = 190
const SKIPPED_ROW_ESTIMATE = 122
const VIRTUAL_OVERSCAN = 3

interface ReviewListProps {
  readonly posts: readonly ReviewPost[]
  readonly onChange: (posts: ReviewPost[]) => void
  readonly loadPreview?: (
    token: string,
    signal: AbortSignal,
  ) => Promise<Blob>
}

interface LightboxImage {
  readonly itemKey: string
  readonly token: string
  readonly url: string
  readonly label: string
}

const warningCopy: Record<UserFacingWarningCode, string> = {
  captionTruncated: 'Caption was shortened to CraftSky’s limit',
  imagesOmitted: 'Some images were left out',
  videoOmitted: 'Video was left out',
  unsupportedMediaOmitted: 'An unsupported file was left out',
  mediaUnavailable: 'Some media could not be opened',
  textOnlyConfirmationRequired: 'This post has no usable image',
}

const warningFilterCopy: Record<UserFacingWarningCode, string> = {
  captionTruncated: 'Shortened captions',
  imagesOmitted: 'Images left out',
  videoOmitted: 'Videos left out',
  unsupportedMediaOmitted: 'Unsupported files',
  mediaUnavailable: 'Media not opened',
  textOnlyConfirmationRequired: 'No usable image',
}

const warningWhyCopy: Record<UserFacingWarningCode, string> = {
  captionTruncated:
    'CraftSky captions can contain up to 2,000 characters, so the rest was removed.',
  imagesOmitted:
    'CraftSky posts can include up to four images, so any extra images were left out.',
  videoOmitted:
    'This importer currently supports photos but not videos.',
  unsupportedMediaOmitted:
    'This importer can prepare JPEG, PNG, and WebP images. Other file types cannot be added yet.',
  mediaUnavailable:
    'The file was missing from the export or could not be prepared safely in this browser.',
  textOnlyConfirmationRequired:
    'The caption can still be imported, but none of this post’s images can be added. Please confirm that you want the post without an image.',
}

const skipCopy: Record<SafeSkipCode, string> = {
  ambiguousDuplicate: 'Duplicate entries could not be matched safely',
  invalidTimestamp: 'Post date could not be read',
  unpublished: 'Draft or unpublished post',
  videoOnly: 'Video posts are not supported yet',
  emptyPost: 'No caption or supported image to import',
  unsupportedPost: 'This type of post is not supported yet',
  rkeyCollision: 'A safe place could not be found for this post',
}

function matchesFilter(post: ReviewPost, filter: ReviewFilter): boolean {
  if (filter === 'textOnlyConfirmationRequired') {
    return post.needsTextOnlyConfirmation
  }
  return filter === 'all' || post.warnings.includes(filter)
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('en-GB', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(new Date(value))
}

function graphemeCount(value: string): number {
  return [
    ...new Intl.Segmenter('en', { granularity: 'grapheme' }).segment(
      value,
    ),
  ].length
}

function fitCaptionEditor(textarea: HTMLTextAreaElement): void {
  textarea.style.height = 'auto'
  textarea.style.height = `${textarea.scrollHeight}px`
}

function MediaPreview({
  token,
  enabled,
  label,
  loadPreview,
  onOpen,
}: {
  readonly token: string
  readonly enabled: boolean
  readonly label: string
  readonly loadPreview?: (
    token: string,
    signal: AbortSignal,
  ) => Promise<Blob>
  readonly onOpen: (preview: {
    readonly token: string
    readonly url: string
    readonly label: string
    readonly trigger: HTMLButtonElement
  }) => void
}): React.JSX.Element {
  const [url, setUrl] = useState<string>()

  useEffect(() => {
    let active = true
    let activeUrl: string | undefined
    const controller = new AbortController()
    if (enabled && loadPreview) {
      void loadPreview(token, controller.signal)
        .then((blob) => {
          if (!active) return
          activeUrl = URL.createObjectURL(blob)
          setUrl(activeUrl)
        })
        .catch(() => undefined)
    }
    return () => {
      active = false
      controller.abort()
      if (activeUrl) URL.revokeObjectURL(activeUrl)
    }
  }, [enabled, loadPreview, token])

  return url ? (
    <button
      type="button"
      className="media-thumbnail-button"
      aria-label={`View ${label} full screen`}
      onClick={(event) =>
        onOpen({
          token,
          url,
          label,
          trigger: event.currentTarget,
        })
      }
    >
      <img className="media-preview" src={url} alt="" />
    </button>
  ) : (
    <span aria-hidden="true" className="media-glyph">
      ◩
    </span>
  )
}

export function ReviewList({
  posts,
  onChange,
  loadPreview,
}: ReviewListProps): React.JSX.Element {
  const [filter, setFilter] = useState<ReviewFilter>('all')
  const [expandedImagePosts, setExpandedImagePosts] = useState<
    ReadonlySet<string>
  >(() => new Set())
  const [lightboxImage, setLightboxImage] = useState<LightboxImage>()
  const lightboxRef = useRef<HTMLDialogElement>(null)
  const lightboxTriggerRef = useRef<HTMLButtonElement | undefined>(
    undefined,
  )
  const scrollElementRef = useRef<HTMLDivElement>(null)
  const availableWarningFilters = useMemo(
    () =>
      (
        Object.keys(warningFilterCopy) as UserFacingWarningCode[]
      ).filter((warning) =>
        posts.some((post) =>
          warning === 'textOnlyConfirmationRequired'
            ? post.needsTextOnlyConfirmation
            : post.warnings.includes(warning),
        ),
      ),
    [posts],
  )
  const visiblePosts = useMemo(
    () => posts.filter((post) => matchesFilter(post, filter)),
    [filter, posts],
  )
  const visibleKeys = useMemo(
    () => new Set(visiblePosts.map((post) => post.itemKey)),
    [visiblePosts],
  )
  const getPostKey = useCallback(
    (index: number) => visiblePosts[index]?.itemKey ?? index,
    [visiblePosts],
  )
  // TanStack Virtual intentionally exposes a mutable virtualizer instance.
  // eslint-disable-next-line react-hooks/incompatible-library
  const rowVirtualizer = useVirtualizer<HTMLDivElement, HTMLDivElement>({
    count: visiblePosts.length,
    getScrollElement: () => scrollElementRef.current,
    estimateSize: () => REVIEW_ROW_ESTIMATE,
    getItemKey: getPostKey,
    overscan: VIRTUAL_OVERSCAN,
    useFlushSync: false,
    initialRect: {
      width: 800,
      height: REVIEW_VIEWPORT_HEIGHT,
    },
  })

  useEffect(() => {
    const dialog = lightboxRef.current
    if (!dialog || !lightboxImage || dialog.open) return
    if (typeof dialog.showModal === 'function') {
      dialog.showModal()
    } else {
      dialog.setAttribute('open', '')
    }
  }, [lightboxImage])

  useEffect(() => {
    if (lightboxImage || !lightboxTriggerRef.current) return
    lightboxTriggerRef.current.focus()
    lightboxTriggerRef.current = undefined
  }, [lightboxImage])

  function changeFilter(value: ReviewFilter): void {
    setLightboxImage(undefined)
    rowVirtualizer.scrollToOffset(0)
    setFilter(value)
  }

  function setImageSectionOpen(itemKey: string, open: boolean): void {
    setExpandedImagePosts((current) => {
      const next = new Set(current)
      if (open) {
        next.add(itemKey)
      } else {
        next.delete(itemKey)
      }
      return next
    })
    if (!open && lightboxImage?.itemKey === itemKey) {
      setLightboxImage(undefined)
    }
  }

  function setVisibleSelection(selected: boolean): void {
    if (!selected) setLightboxImage(undefined)
    onChange(setPostSelectionByKey(posts, visibleKeys, selected))
  }

  return (
    <section className="review-list-section" aria-labelledby="review-list-title">
      <div className="review-toolbar">
        <div>
          <h2 id="review-list-title">Choose what to import</h2>
          <p className="muted">
           You can make changes without
            reviewing every post.
          </p>
        </div>
        <div className="review-toolbar__actions">
          <button
            type="button"
            className="text-button"
            onClick={() => setVisibleSelection(true)}
          >
            Select shown
          </button>
          <button
            type="button"
            className="text-button"
            onClick={() => setVisibleSelection(false)}
          >
            Deselect shown
          </button>
        </div>
      </div>

      <div
        className="filter-controls"
        role="group"
        aria-label="Filter posts"
      >
        <button
          type="button"
          aria-pressed={filter === 'all'}
          onClick={() => changeFilter('all')}
        >
          All posts
        </button>
        {availableWarningFilters.map((warning) => (
          <button
            key={warning}
            type="button"
            aria-pressed={filter === warning}
            onClick={() => changeFilter(warning)}
          >
            {warningFilterCopy[warning]}
          </button>
        ))}
      </div>

      <p className="review-result-count" aria-live="polite">
        Showing {visiblePosts.length.toLocaleString('en-GB')}{' '}
        {visiblePosts.length === 1 ? 'post' : 'posts'}
      </p>

      {visiblePosts.length === 0 ? (
        <div className="empty-state">
          <p>No posts match this filter.</p>
          <button
            type="button"
            className="text-button"
            onClick={() => changeFilter('all')}
          >
            Show all posts
          </button>
        </div>
      ) : (
        <div
          ref={scrollElementRef}
          className="review-window virtual-review-window"
          aria-label="Posts to review"
          role="list"
          tabIndex={0}
          style={
            {
              '--review-content-height': `${rowVirtualizer.getTotalSize()}px`,
            } as React.CSSProperties
          }
        >
          <div
            className="virtual-review-window__inner"
            style={{ height: `${rowVirtualizer.getTotalSize()}px` }}
          >
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const post = visiblePosts[virtualRow.index]
              if (!post) return null
              const visibleWarnings = post.warnings.filter(
                isUserFacingWarningCode,
              )
              const imageSectionOpen = expandedImagePosts.has(
                post.itemKey,
              )
              return (
                <div
                  key={virtualRow.key}
                  ref={rowVirtualizer.measureElement}
                  className="virtual-review-row"
                  data-index={virtualRow.index}
                  role="listitem"
                  aria-posinset={virtualRow.index + 1}
                  aria-setsize={visiblePosts.length}
                  style={{
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                >
                  <article
                    className={`review-post ${
                      post.selected ? '' : 'review-post--excluded'
                    }`}
                  >
                    <div className="review-post__heading">
                      <label className="selection-label">
                        <input
                          type="checkbox"
                          checked={post.selected}
                          onChange={(event) => {
                            if (
                              !event.currentTarget.checked &&
                              lightboxImage?.itemKey === post.itemKey
                            ) {
                              setLightboxImage(undefined)
                            }
                            onChange(
                              updatePostSelection(
                                posts,
                                post.itemKey,
                                event.currentTarget.checked,
                              ),
                            )
                          }}
                        />
                        <span>
                          <strong>{formatDate(post.createdAt)}</strong>
                          <small>
                            {post.selected
                              ? 'Included in import'
                              : 'Not included'}
                          </small>
                        </span>
                      </label>
                      <span className="post-number">
                        Post {virtualRow.index + 1}
                      </span>
                    </div>

                    {visibleWarnings.length > 0 && (
                      <ul
                        className="warning-list"
                        aria-label="Post warnings"
                      >
                        {visibleWarnings.map((warning) => {
                          const tooltipId = `warning-${virtualRow.index}-${warning}`
                          return (
                            <li key={warning}>
                              <span>{warningCopy[warning]}</span>
                              <span className="warning-tooltip">
                                <button
                                  type="button"
                                  className="warning-tooltip__trigger"
                                  aria-describedby={tooltipId}
                                >
                                  Why?
                                </button>
                                <span
                                  id={tooltipId}
                                  className="warning-tooltip__bubble"
                                  role="tooltip"
                                >
                                  {warningWhyCopy[warning]}
                                </span>
                              </span>
                            </li>
                          )
                        })}
                      </ul>
                    )}

                    <details
                      className="post-section"
                      onToggle={(event) => {
                        if (!event.currentTarget.open) return
                        const textarea =
                          event.currentTarget.querySelector('textarea')
                        if (textarea) fitCaptionEditor(textarea)
                      }}
                    >
                      <summary className="post-section__summary">
                        <strong>Caption</strong>
                        <small>
                          {graphemeCount(post.caption)} / 2,000
                        </small>
                      </summary>
                      <div className="post-section__content field-label">
                        <textarea
                          aria-label="Caption"
                          value={post.caption}
                          rows={1}
                          spellCheck={false}
                          autoCapitalize="off"
                          autoCorrect="off"
                          onChange={(event) => {
                            fitCaptionEditor(event.currentTarget)
                            onChange(
                              updateCaption(
                                posts,
                                post.itemKey,
                                event.currentTarget.value,
                              ),
                            )
                          }}
                        />
                      </div>
                    </details>

                    {post.media.length > 0 && (
                      <details
                        className="post-section"
                        open={imageSectionOpen}
                        onToggle={(event) =>
                          setImageSectionOpen(
                            post.itemKey,
                            event.currentTarget.open,
                          )
                        }
                      >
                        <summary className="post-section__summary">
                          <strong>Images ({post.media.length})</strong>
                        </summary>
                        <div className="post-section__content">
                          <fieldset
                            className="media-options"
                            aria-label={`Images (${post.media.length})`}
                            disabled={!post.selected}
                          >
                            {post.media.map((media, index) => {
                              const inputId = `media-${post.itemKey}-${index}`
                              const previewEnabled =
                                imageSectionOpen &&
                                post.selected &&
                                media.selected
                              return (
                                <div
                                  key={media.token}
                                  className="media-option"
                                >
                                  <input
                                    id={inputId}
                                    type="checkbox"
                                    aria-label={`Include image ${index + 1}`}
                                    checked={media.selected}
                                    onChange={(event) => {
                                      if (
                                        !event.currentTarget.checked &&
                                        lightboxImage?.token === media.token
                                      ) {
                                        setLightboxImage(undefined)
                                      }
                                      onChange(
                                        updateMediaSelection(
                                          posts,
                                          post.itemKey,
                                          media.token,
                                          event.currentTarget.checked,
                                        ),
                                      )
                                    }}
                                  />
                                  <MediaPreview
                                    token={media.token}
                                    enabled={previewEnabled}
                                    label={`Image ${index + 1}`}
                                    loadPreview={loadPreview}
                                    onOpen={({ trigger, ...preview }) => {
                                      lightboxTriggerRef.current = trigger
                                      setLightboxImage({
                                        itemKey: post.itemKey,
                                        ...preview,
                                      })
                                    }}
                                  />
                                  <label htmlFor={inputId}>
                                    Image {index + 1}
                                    <small>
                                      {media.width} × {media.height}
                                    </small>
                                  </label>
                                </div>
                              )
                            })}
                          </fieldset>
                        </div>
                      </details>
                    )}

                    {post.selected &&
                      post.needsTextOnlyConfirmation && (
                        <label className="confirmation-check confirmation-check--warning">
                          <input
                            type="checkbox"
                            checked={post.textOnlyConfirmed}
                            onChange={(event) => {
                              if (event.currentTarget.checked) {
                                onChange(
                                  confirmTextOnly(posts, post.itemKey),
                                )
                              } else {
                                onChange(
                                  posts.map((candidate) =>
                                    candidate.itemKey === post.itemKey
                                      ? {
                                          ...candidate,
                                          textOnlyConfirmed: false,
                                        }
                                      : candidate,
                                  ),
                                )
                              }
                            }}
                          />
                          <span>
                            Import this post without an image
                            <small>
                              Instagram posts normally include media. This
                              post’s caption is usable, but no selected image
                              can be added.
                            </small>
                          </span>
                        </label>
                      )}
                  </article>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {lightboxImage && (
        <dialog
          ref={lightboxRef}
          className="media-lightbox"
          aria-label={`${lightboxImage.label} full-screen preview`}
          onCancel={() => setLightboxImage(undefined)}
          onClick={(event) => {
            if (event.target === event.currentTarget) {
              setLightboxImage(undefined)
            }
          }}
        >
          <button
            type="button"
            className="media-lightbox__close"
            aria-label="Close preview"
            autoFocus
            onClick={() => setLightboxImage(undefined)}
          >
            <span aria-hidden="true">×</span>
          </button>
          <div className="media-lightbox__content">
            <img src={lightboxImage.url} alt="" />
            <p>{lightboxImage.label}</p>
          </div>
        </dialog>
      )}
    </section>
  )
}

export function SkippedReviewList({
  skipped,
}: {
  readonly skipped: readonly SkippedPost[]
}): React.JSX.Element | null {
  const [filter, setFilter] = useState<SafeSkipCode | 'all'>('all')
  const scrollElementRef = useRef<HTMLDivElement>(null)
  const availableCodes = useMemo(
    () => [...new Set(skipped.map((item) => item.code))],
    [skipped],
  )
  const visibleItems = useMemo(
    () =>
      filter === 'all'
        ? skipped
        : skipped.filter((item) => item.code === filter),
    [filter, skipped],
  )
  const getSkippedKey = useCallback(
    (index: number) => visibleItems[index]?.itemKey ?? index,
    [visibleItems],
  )
  // TanStack Virtual intentionally exposes a mutable virtualizer instance.
  // eslint-disable-next-line react-hooks/incompatible-library
  const rowVirtualizer = useVirtualizer<HTMLDivElement, HTMLDivElement>({
    count: visibleItems.length,
    getScrollElement: () => scrollElementRef.current,
    estimateSize: () => SKIPPED_ROW_ESTIMATE,
    getItemKey: getSkippedKey,
    overscan: VIRTUAL_OVERSCAN,
    useFlushSync: false,
    initialRect: {
      width: 800,
      height: SKIPPED_VIEWPORT_HEIGHT,
    },
  })

  function changeFilter(value: SafeSkipCode | 'all'): void {
    rowVirtualizer.scrollToOffset(0)
    setFilter(value)
  }

  if (skipped.length === 0) return null
  return (
    <section
      className="review-list-section skipped-review-section"
      aria-labelledby="skipped-review-title"
    >
      <div className="review-toolbar">
        <div>
          <h2 id="skipped-review-title">Skipped posts</h2>
          <p className="muted">
            These entries cannot be imported safely in this version.
          </p>
        </div>
        <strong>{skipped.length.toLocaleString('en-GB')} skipped</strong>
      </div>
      <label className="field-label skipped-filter">
        <span>Filter skipped posts</span>
        <select
          value={filter}
          onChange={(event) =>
            changeFilter(
              event.currentTarget.value as SafeSkipCode | 'all',
            )
          }
        >
          <option value="all">All reasons</option>
          {availableCodes.map((code) => (
            <option key={code} value={code}>
              {skipCopy[code]}
            </option>
          ))}
        </select>
      </label>
      <p className="review-result-count" aria-live="polite">
        Showing {visibleItems.length.toLocaleString('en-GB')} of{' '}
        {skipped.length.toLocaleString('en-GB')} skipped posts
      </p>
      {visibleItems.length === 0 ? (
        <div className="empty-state">
          <p>No skipped posts match this filter.</p>
        </div>
      ) : (
        <div
          ref={scrollElementRef}
          className="review-window virtual-review-window virtual-review-window--skipped"
          aria-label="Skipped posts"
          role="list"
          tabIndex={0}
          style={
            {
              '--review-content-height': `${rowVirtualizer.getTotalSize()}px`,
            } as React.CSSProperties
          }
        >
          <div
            className="virtual-review-window__inner"
            style={{ height: `${rowVirtualizer.getTotalSize()}px` }}
          >
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const item = visibleItems[virtualRow.index]
              if (!item) return null
              return (
                <div
                  key={virtualRow.key}
                  ref={rowVirtualizer.measureElement}
                  className="virtual-review-row virtual-review-row--skipped"
                  data-index={virtualRow.index}
                  role="listitem"
                  aria-posinset={virtualRow.index + 1}
                  aria-setsize={visibleItems.length}
                  style={{
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                >
                  <article className="review-post review-post--excluded">
                    <div className="review-post__heading">
                      <span>
                        <strong>
                          {item.createdAt
                            ? formatDate(item.createdAt)
                            : 'Date unavailable'}
                        </strong>
                        <small>{skipCopy[item.code]}</small>
                      </span>
                      <span className="post-number">
                        Skipped {virtualRow.index + 1}
                      </span>
                    </div>
                  </article>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </section>
  )
}
