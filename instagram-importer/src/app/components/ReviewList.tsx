import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'

import type {
  ReviewPost,
  SafeSkipCode,
  SafeWarningCode,
  SkippedPost,
} from '../../domain/types'
import {
  confirmTextOnly,
  setPostSelectionByKey,
  updateCaption,
  updateMediaSelection,
  updatePostSelection,
} from '../../review/reviewState'

export type ReviewFilter = 'all' | 'warnings' | 'textOnly'

const REVIEW_VIEWPORT_HEIGHT = 720
const SKIPPED_VIEWPORT_HEIGHT = 480
const REVIEW_ROW_ESTIMATE = 430
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

const warningCopy: Record<SafeWarningCode, string> = {
  captionRepaired: 'Caption text was repaired',
  captionTruncated: 'Caption was shortened to CraftSky’s limit',
  imagesOmitted: 'Some images were omitted',
  videoOmitted: 'Video was omitted',
  unsupportedMediaOmitted: 'Unsupported media was omitted',
  mediaUnavailable: 'Some media could not be read',
  textOnlyConfirmationRequired: 'Confirm this text-only post',
}

const skipCopy: Record<SafeSkipCode, string> = {
  ambiguousDuplicate: 'Conflicting duplicate export entries',
  invalidTimestamp: 'Creation date is outside the supported range',
  unpublished: 'Draft or unpublished post',
  videoOnly: 'Video-only post',
  emptyPost: 'No publishable caption or image',
  unsupportedPost: 'Unsupported post shape',
  rkeyCollision: 'Deterministic record-key collision',
}

function matchesFilter(post: ReviewPost, filter: ReviewFilter): boolean {
  if (filter === 'warnings') return post.warnings.length > 0
  if (filter === 'textOnly') return post.needsTextOnlyConfirmation
  return true
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

function MediaPreview({
  token,
  enabled,
  label,
  loadPreview,
}: {
  readonly token: string
  readonly enabled: boolean
  readonly label: string
  readonly loadPreview?: (
    token: string,
    signal: AbortSignal,
  ) => Promise<Blob>
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
    <img className="media-preview" src={url} alt={`${label} preview`} />
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
  const [activePreviewToken, setActivePreviewToken] =
    useState<string>()
  const scrollElementRef = useRef<HTMLDivElement>(null)
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

  function changeFilter(value: ReviewFilter): void {
    rowVirtualizer.scrollToOffset(0)
    setFilter(value)
  }

  function setVisibleSelection(selected: boolean): void {
    onChange(setPostSelectionByKey(posts, visibleKeys, selected))
  }

  return (
    <section className="review-list-section" aria-labelledby="review-list-title">
      <div className="review-toolbar">
        <div>
          <h2 id="review-list-title">Choose what to import</h2>
          <p className="muted">
            Everything importable is selected. You can make changes without
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

      <div className="segmented-control" aria-label="Filter posts">
        {(
          [
            ['all', 'All posts'],
            ['warnings', 'With warnings'],
            ['textOnly', 'Text only'],
          ] as const
        ).map(([value, label]) => (
          <button
            key={value}
            type="button"
            aria-pressed={filter === value}
            onClick={() => changeFilter(value)}
          >
            {label}
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
        >
          <div
            className="virtual-review-window__inner"
            style={{ height: `${rowVirtualizer.getTotalSize()}px` }}
          >
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const post = visiblePosts[virtualRow.index]
              if (!post) return null
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
                          onChange={(event) =>
                            onChange(
                              updatePostSelection(
                                posts,
                                post.itemKey,
                                event.currentTarget.checked,
                              ),
                            )
                          }
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

                    {post.warnings.length > 0 && (
                      <ul
                        className="warning-list"
                        aria-label="Post warnings"
                      >
                        {post.warnings.map((warning) => (
                          <li key={warning}>{warningCopy[warning]}</li>
                        ))}
                      </ul>
                    )}

                    <label className="field-label">
                      <span>
                        Caption
                        <small>
                          {graphemeCount(post.caption)} / 2,000
                        </small>
                      </span>
                      <textarea
                        value={post.caption}
                        rows={4}
                        spellCheck={false}
                        autoCapitalize="off"
                        autoCorrect="off"
                        onChange={(event) =>
                          onChange(
                            updateCaption(
                              posts,
                              post.itemKey,
                              event.currentTarget.value,
                            ),
                          )
                        }
                      />
                    </label>

                    {post.media.length > 0 && (
                      <fieldset
                        className="media-options"
                        disabled={!post.selected}
                      >
                        <legend>Images</legend>
                        {post.media.map((media, index) => {
                          const inputId = `media-${post.itemKey}-${index}`
                          const previewActive =
                            post.selected &&
                            media.selected &&
                            activePreviewToken === media.token
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
                                    activePreviewToken === media.token
                                  ) {
                                    setActivePreviewToken(undefined)
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
                                key={`${media.token}:${
                                  previewActive
                                    ? 'selected'
                                    : 'deselected'
                                }`}
                                token={media.token}
                                enabled={previewActive}
                                label={`Image ${index + 1}`}
                                loadPreview={loadPreview}
                              />
                              <label htmlFor={inputId}>
                                Image {index + 1}
                                <small>
                                  {media.width} × {media.height}
                                </small>
                              </label>
                              {loadPreview &&
                                post.selected &&
                                media.selected && (
                                  <button
                                    type="button"
                                    className="text-button media-preview-button"
                                    onClick={() =>
                                      setActivePreviewToken(
                                        (current) =>
                                          current === media.token
                                            ? undefined
                                            : media.token,
                                      )
                                    }
                                  >
                                    {previewActive
                                      ? `Hide image ${index + 1} preview`
                                      : `Preview image ${index + 1}`}
                                  </button>
                                )}
                            </div>
                          )
                        })}
                      </fieldset>
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
                            Import this as a text-only post
                            <small>
                              No selected image will be attached to this
                              post.
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
