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

const REVIEW_ROW_HEIGHT = 190
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
    warnings: index % 10 === 0 ? ['captionTruncated'] : [],
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
  it('does not present automatic caption repairs as warnings', () => {
    render(<ReviewHarness initialPosts={[post(1), {
      ...post(2),
      warnings: ['captionRepaired'],
    }]} />)

    expect(
      screen.queryByText('Caption text was repaired'),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('list', { name: 'Post warnings' }),
    ).not.toBeInTheDocument()

    expect(
      screen.queryByRole('button', { name: 'Shortened captions' }),
    ).not.toBeInTheDocument()
  })

  it('keeps a post caption collapsed until the member opens it', async () => {
    const ui = userEvent.setup()
    render(<ReviewHarness initialPosts={[post(0)]} />)

    const captionSummary = screen.getByText('Caption').closest('summary')
    expect(captionSummary).not.toBeNull()
    const captionDetails = captionSummary!.closest('details')
    expect(captionDetails).not.toBeNull()
    expect(captionDetails).not.toHaveAttribute('open')
    expect(
      screen.getByRole('textbox', { name: /Caption/u }),
    ).not.toBeVisible()

    await ui.click(captionSummary!)

    expect(captionDetails).toHaveAttribute('open')
    expect(
      screen.getByRole('textbox', { name: /Caption/u }),
    ).toBeVisible()
  })

  it('expands the caption editor to its full content height', async () => {
    const ui = userEvent.setup()
    vi.spyOn(
      HTMLTextAreaElement.prototype,
      'scrollHeight',
      'get',
    ).mockReturnValue(176)
    render(<ReviewHarness initialPosts={[post(0)]} />)

    await ui.click(screen.getByText('Caption').closest('summary')!)

    expect(
      screen.getByRole('textbox', { name: 'Caption' }),
    ).toHaveStyle({ height: '176px' })
  })

  it('shows the image count and expands images independently', async () => {
    const ui = userEvent.setup()
    const basePost = post(0)
    const reviewPost: ReviewPost = {
      ...basePost,
      media: [
        ...basePost.media,
        {
          token: 'media-0-second',
          kind: 'image',
          mime: 'image/jpeg',
          width: 1080,
          height: 1350,
          selected: true,
        },
      ],
    }
    render(<ReviewHarness initialPosts={[reviewPost]} />)

    const imageSummary = screen
      .getByText('Images (2)')
      .closest('summary')
    expect(imageSummary).not.toBeNull()
    const imageDetails = imageSummary!.closest('details')
    expect(imageDetails).not.toHaveAttribute('open')
    expect(
      screen.getByRole('checkbox', { name: 'Include image 1' }),
    ).not.toBeVisible()

    await ui.click(imageSummary!)

    expect(imageDetails).toHaveAttribute('open')
    expect(
      screen.getByRole('checkbox', { name: 'Include image 1' }),
    ).toBeVisible()
    expect(
      screen.getByRole('textbox', { name: 'Caption' }),
    ).not.toBeVisible()
  })

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
        screen.getByRole('button', { name: 'Shortened captions' }),
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

  it('offers a separate filter and plain-language explanation for each warning type', async () => {
    const ui = userEvent.setup()
    const warningPosts: ReviewPost[] = [
      {
        ...post(1),
        warnings: ['captionTruncated'],
      },
      {
        ...post(2),
        warnings: ['imagesOmitted'],
      },
      {
        ...post(3),
        media: [],
        warnings: ['textOnlyConfirmationRequired'],
        needsTextOnlyConfirmation: true,
      },
    ]
    render(<ReviewHarness initialPosts={warningPosts} />)

    expect(
      screen.getByRole('button', { name: 'Shortened captions' }),
    ).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'Images left out' }),
    ).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'No usable image' }),
    ).toBeVisible()

    await ui.click(
      screen.getByRole('button', { name: 'Images left out' }),
    )
    expect(screen.getByText('Showing 1 post')).toBeVisible()
    expect(screen.getByText('Some images were left out')).toBeVisible()
    expect(
      screen.getByRole('button', {
        name: 'Why?',
        description: /up to four images/u,
      }),
    ).toBeVisible()
  })

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

  it('loads selected media thumbnails when the image section expands', async () => {
    const ui = userEvent.setup()
    const loadPreview = vi
      .fn()
      .mockResolvedValue(new Blob(['sanitized'], { type: 'image/jpeg' }))
    const createObjectURL = vi
      .fn()
      .mockReturnValueOnce('blob:safe-preview-1')
      .mockReturnValueOnce('blob:safe-preview-2')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal(
      'URL',
      class TestURL extends URL {
        static createObjectURL = createObjectURL
        static revokeObjectURL = revokeObjectURL
      },
    )
    const basePost = post(0)
    render(
      <ReviewHarness
        initialPosts={[
          {
            ...basePost,
            media: [
              ...basePost.media,
              {
                ...basePost.media[0],
                token: 'media-0-second',
              },
            ],
          },
        ]}
        loadPreview={loadPreview}
      />,
    )

    expect(loadPreview).not.toHaveBeenCalled()
    await ui.click(screen.getByText('Images (2)'))

    expect(
      await screen.findByRole('button', {
        name: 'View Image 1 full screen',
      }),
    ).toBeVisible()
    expect(
      screen.getByRole('button', {
        name: 'View Image 2 full screen',
      }),
    ).toBeVisible()
    expect(loadPreview).toHaveBeenCalledWith(
      'media-0',
      expect.any(AbortSignal),
    )
    expect(loadPreview).toHaveBeenCalledWith(
      'media-0-second',
      expect.any(AbortSignal),
    )
    expect(loadPreview).toHaveBeenCalledTimes(2)

    await ui.click(screen.getByText('Images (2)'))
    await waitFor(() =>
      expect(revokeObjectURL).toHaveBeenCalledTimes(2),
    )
  })

  it('opens a thumbnail in a full-screen modal and closes it', async () => {
    const ui = userEvent.setup()
    const loadPreview = vi
      .fn()
      .mockResolvedValue(new Blob(['sanitized'], { type: 'image/jpeg' }))
    const createObjectURL = vi.fn().mockReturnValue('blob:safe-preview')
    vi.stubGlobal(
      'URL',
      class TestURL extends URL {
        static createObjectURL = createObjectURL
        static revokeObjectURL = vi.fn()
      },
    )
    render(
      <ReviewHarness initialPosts={[post(0)]} loadPreview={loadPreview} />,
    )

    await ui.click(screen.getByText('Images (1)'))
    const thumbnail = await screen.findByRole('button', {
      name: 'View Image 1 full screen',
    })
    await ui.click(thumbnail)

    const dialog = screen.getByRole('dialog', {
      name: 'Image 1 full-screen preview',
    })
    expect(dialog).toBeVisible()
    expect(dialog.querySelector('img')).toHaveAttribute(
      'src',
      'blob:safe-preview',
    )

    await ui.click(
      within(dialog).getByRole('button', { name: 'Close preview' }),
    )
    expect(
      screen.queryByRole('dialog', {
        name: 'Image 1 full-screen preview',
      }),
    ).not.toBeInTheDocument()
    expect(thumbnail).toHaveFocus()

    await ui.click(thumbnail)
    const reopenedDialog = screen.getByRole('dialog', {
      name: 'Image 1 full-screen preview',
    })
    fireEvent.click(reopenedDialog)
    expect(
      screen.queryByRole('dialog', {
        name: 'Image 1 full-screen preview',
      }),
    ).not.toBeInTheDocument()
  })

  it('aborts in-flight thumbnails when the image section closes', async () => {
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

    await ui.click(screen.getByText('Images (1)'))
    expect(loadPreview).toHaveBeenCalledOnce()
    const signal = loadPreview.mock.calls[0]?.[1]
    expect(signal).toBeInstanceOf(AbortSignal)
    expect(signal?.aborted).toBe(false)

    await ui.click(screen.getByText('Images (1)'))
    expect(signal?.aborted).toBe(true)
  })
})
