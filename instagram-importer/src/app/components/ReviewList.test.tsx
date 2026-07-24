import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReviewPost, SkippedPost } from '../../domain/types'
import { ReviewList, SkippedReviewList } from './ReviewList'

const REVIEW_ROW_HEIGHT = 430
const SKIPPED_ROW_HEIGHT = 122
const originalScrollTo = Object.getOwnPropertyDescriptor(
  HTMLElement.prototype,
  'scrollTo',
)

function post(index: number): ReviewPost {
  return {
    itemKey: `item-${index}`,
    rkey: `rkey-${index}`,
    createdAt: '2024-01-02T03:04:05.000Z',
    caption: `Synthetic caption ${index}`,
    initialCaption: `Synthetic caption ${index}`,
    media: [
      {
        token: `media-${index}`,
        kind: 'image',
        mime: 'image/jpeg',
        width: 1080,
        height: 1350,
        selected: true,
      },
    ],
    warnings: index % 10 === 0 ? ['captionRepaired'] : [],
    selected: true,
    needsTextOnlyConfirmation: false,
    textOnlyConfirmed: false,
  }
}

function ReviewHarness({
  initialPosts,
  loadPreview,
}: {
  readonly initialPosts: readonly ReviewPost[]
  readonly loadPreview?: (
    token: string,
    signal: AbortSignal,
  ) => Promise<Blob>
}): React.JSX.Element {
  const [posts, setPosts] = useState([...initialPosts])
  return (
    <>
      <output data-testid="selected-count">
        {posts.filter((item) => item.selected).length}
      </output>
      <ReviewList
        posts={posts}
        onChange={setPosts}
        loadPreview={loadPreview}
      />
    </>
  )
}

beforeEach(() => {
  vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(
    800,
  )
  vi.spyOn(
    HTMLElement.prototype,
    'offsetHeight',
    'get',
  ).mockImplementation(function (this: HTMLElement) {
    if (this.classList.contains('virtual-review-window--skipped')) {
      return 480
    }
    if (this.classList.contains('virtual-review-window')) return 720
    if (this.classList.contains('virtual-review-row--skipped')) {
      return SKIPPED_ROW_HEIGHT
    }
    if (this.classList.contains('virtual-review-row')) {
      return REVIEW_ROW_HEIGHT
    }
    return 0
  })
  Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
    configurable: true,
    value: function (
      this: HTMLElement,
      options: ScrollToOptions | number,
      y?: number,
    ): void {
      this.scrollTop =
        typeof options === 'number' ? (y ?? 0) : (options.top ?? 0)
      this.dispatchEvent(new Event('scroll'))
    },
  })
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  if (originalScrollTo) {
    Object.defineProperty(
      HTMLElement.prototype,
      'scrollTo',
      originalScrollTo,
    )
  } else {
    Reflect.deleteProperty(HTMLElement.prototype, 'scrollTo')
  }
})

describe('virtualized review lists (AT-004, AC-006)', () => {
  it(
    'keeps thousands of importable posts bounded while preserving filtering and controls',
    async () => {
      const ui = userEvent.setup()
      render(
        <ReviewHarness
          initialPosts={Array.from({ length: 2_000 }, (_, index) =>
            post(index),
          )}
        />,
      )

      const reviewList = screen.getByRole('list', {
        name: 'Posts to review',
      })
      expect(reviewList).toHaveAttribute('tabindex', '0')
      expect(screen.getByText('Showing 2,000 posts')).toBeVisible()
      expect(
        within(reviewList).getAllByRole('article').length,
      ).toBeLessThan(12)
      expect(
        screen.queryByDisplayValue('Synthetic caption 1,000'),
      ).not.toBeInTheDocument()

      reviewList.scrollTop = REVIEW_ROW_HEIGHT * 1_000
      fireEvent.scroll(reviewList)
      await screen.findByDisplayValue('Synthetic caption 1000')
      expect(
        within(reviewList).getAllByRole('article').length,
      ).toBeLessThan(12)

      await ui.click(
        screen.getByRole('button', { name: 'With warnings' }),
      )
      await waitFor(() => expect(reviewList.scrollTop).toBe(0))
      expect(screen.getByText('Showing 200 posts')).toBeVisible()
      expect(
        within(reviewList).getAllByRole('article').length,
      ).toBeLessThan(12)
      expect(
        screen.queryByDisplayValue('Synthetic caption 1'),
      ).not.toBeInTheDocument()

      await ui.click(
        screen.getByRole('button', { name: 'Deselect shown' }),
      )
      expect(screen.getByTestId('selected-count')).toHaveTextContent(
        '1800',
      )

      const firstVisibleSelection = within(reviewList).getAllByRole(
        'checkbox',
      )[0]
      await ui.click(firstVisibleSelection)
      expect(screen.getByTestId('selected-count')).toHaveTextContent(
        '1801',
      )
    },
    30_000,
  )

  it('virtualizes and filters skipped posts without rendering every item', async () => {
    const ui = userEvent.setup()
    const skipped: SkippedPost[] = Array.from(
      { length: 2_000 },
      (_, index) => ({
        itemKey: `skipped-${index}`,
        code:
          index % 2 === 0
            ? ('invalidTimestamp' as const)
            : ('unpublished' as const),
      }),
    )
    render(<SkippedReviewList skipped={skipped} />)

    const skippedList = screen.getByRole('list', {
      name: 'Skipped posts',
    })
    expect(skippedList).toHaveAttribute('tabindex', '0')
    expect(
      within(skippedList).getAllByRole('article').length,
    ).toBeLessThan(14)
    expect(screen.getByText('Showing 2,000 of 2,000 skipped posts')).toBeVisible()

    await ui.selectOptions(
      screen.getByLabelText('Filter skipped posts'),
      'invalidTimestamp',
    )
    expect(
      screen.getByText('Showing 1,000 of 2,000 skipped posts'),
    ).toBeVisible()
    expect(
      within(skippedList).queryByText('Draft or unpublished post'),
    ).not.toBeInTheDocument()
    expect(
      within(skippedList).getAllByRole('article').length,
    ).toBeLessThan(14)

    skippedList.scrollTop = SKIPPED_ROW_HEIGHT * 900
    fireEvent.scroll(skippedList)
    await screen.findByText('Skipped 901')
    expect(
      within(skippedList).getAllByRole('article').length,
    ).toBeLessThan(14)
  })

  it('loads only an explicitly requested visible media preview', async () => {
    const ui = userEvent.setup()
    const loadPreview = vi
      .fn()
      .mockResolvedValue(new Blob(['sanitized'], { type: 'image/jpeg' }))
    const createObjectURL = vi.fn().mockReturnValue('blob:safe-preview')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal(
      'URL',
      class TestURL extends URL {
        static createObjectURL = createObjectURL
        static revokeObjectURL = revokeObjectURL
      },
    )
    render(
      <ReviewHarness initialPosts={[post(0)]} loadPreview={loadPreview} />,
    )

    expect(loadPreview).not.toHaveBeenCalled()
    await ui.click(
      screen.getByRole('button', { name: 'Preview image 1' }),
    )
    expect(
      await screen.findByRole('img', { name: 'Image 1 preview' }),
    ).toHaveAttribute('src', 'blob:safe-preview')
    expect(loadPreview).toHaveBeenCalledWith(
      'media-0',
      expect.any(AbortSignal),
    )

    await ui.click(
      screen.getByRole('button', { name: 'Hide image 1 preview' }),
    )
    await waitFor(() =>
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:safe-preview'),
    )
  })

  it('aborts an in-flight media preview when the preview is hidden', async () => {
    const ui = userEvent.setup()
    const loadPreview = vi.fn(
      (_token: string, signal?: AbortSignal) =>
        new Promise<Blob>((_resolve, reject) => {
          signal?.addEventListener(
            'abort',
            () =>
              reject(
                new DOMException(
                  'The operation was aborted.',
                  'AbortError',
                ),
              ),
            { once: true },
          )
        }),
    )
    render(
      <ReviewHarness initialPosts={[post(0)]} loadPreview={loadPreview} />,
    )

    await ui.click(
      screen.getByRole('button', { name: 'Preview image 1' }),
    )
    expect(loadPreview).toHaveBeenCalledOnce()
    const signal = loadPreview.mock.calls[0]?.[1]
    expect(signal).toBeInstanceOf(AbortSignal)
    expect(signal?.aborted).toBe(false)

    await ui.click(
      screen.getByRole('button', { name: 'Hide image 1 preview' }),
    )
    expect(signal?.aborted).toBe(true)
  })
})
