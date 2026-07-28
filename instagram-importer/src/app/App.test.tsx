import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from 'vitest'

import type { ReviewManifest, ReviewPost } from '../domain/types'
import type { RuntimePolicy } from '../config/runtime'
import {
  App,
  type ImporterUiServices,
  type PublishOutcome,
} from './App'

const originalScrollToDescriptor = Object.getOwnPropertyDescriptor(
  HTMLElement.prototype,
  'scrollTo',
)

beforeEach(() => {
  vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(
    800,
  )
  vi.spyOn(
    HTMLElement.prototype,
    'offsetHeight',
    'get',
  ).mockImplementation(function (this: HTMLElement) {
    if (this.classList.contains('virtual-review-window')) return 720
    if (this.classList.contains('virtual-review-row')) return 430
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
  if (originalScrollToDescriptor) {
    Object.defineProperty(
      HTMLElement.prototype,
      'scrollTo',
      originalScrollToDescriptor,
    )
  } else {
    Reflect.deleteProperty(HTMLElement.prototype, 'scrollTo')
  }
})

const runtime: RuntimePolicy = {
  mode: 'local',
  origin: 'http://localhost:5173',
  redirectUri: 'http://localhost:5173/oauth/callback',
  clientId: 'http://localhost',
  writesEnabled: true,
}

function post(index: number): ReviewPost {
  const day = String((index % 27) + 1).padStart(2, '0')
  return {
    itemKey: `item-${index}`,
    rkey: `rkey-${index}`,
    createdAt: `2024-01-${day}T12:00:00.000Z`,
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
    warnings: index === 0 ? ['captionRepaired'] : [],
    selected: true,
    needsTextOnlyConfirmation: false,
    textOnlyConfirmed: false,
  }
}

function manifest(count = 2): ReviewManifest {
  return {
    schemaVersion: 1,
    fingerprint: 'synthetic-fingerprint',
    posts: Array.from({ length: count }, (_, index) => post(index)),
    skipped: [],
    counts: {
      selectedPosts: count,
      selectedImages: count,
      transformedPosts: 1,
      warningPosts: 1,
      skippedPosts: 0,
    },
  }
}

function services(
  reviewManifest: ReviewManifest = manifest(),
): ImporterUiServices {
  return {
    runtime,
    inspect: vi.fn().mockResolvedValue(reviewManifest),
    cancelInspection: vi.fn(),
    authorize: vi.fn().mockResolvedValue({
      did: 'did:plc:synthetic',
      displayName: 'Synthetic Maker',
      accountLabel: 'maker.example',
      avatarUrl:
        'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
    }),
    beginImport: vi.fn().mockResolvedValue(undefined),
    publish: vi
      .fn<
        (
          post: ReviewPost,
          signal: AbortSignal,
        ) => Promise<PublishOutcome>
      >()
      .mockResolvedValue({ status: 'created' }),
    finishImport: vi.fn().mockResolvedValue(undefined),
  }
}

async function loadReview(
  ui: ReturnType<typeof userEvent.setup>,
  appServices: ImporterUiServices,
): Promise<void> {
  render(<App services={appServices} />)
  const fileInput = document.querySelector<HTMLInputElement>(
    'input[type="file"]',
  )
  expect(fileInput).not.toBeNull()
  expect(fileInput).toBeDisabled()

  await ui.click(
    screen.getByRole('checkbox', {
      name: /selected posts and images will become public/i,
    }),
  )
  expect(fileInput).toBeEnabled()
  await ui.upload(
    fileInput!,
    new File(['synthetic'], 'never-render-this-name.zip', {
      type: 'application/zip',
    }),
  )
  await screen.findByRole('heading', {
    name: /ready to review/i,
  })
}

async function reachConfirmation(
  ui: ReturnType<typeof userEvent.setup>,
  appServices: ImporterUiServices,
): Promise<void> {
  await loadReview(ui, appServices)
  await ui.click(
    screen.getByRole('button', {
      name: 'Connect your CraftSky account',
    }),
  )
  await ui.type(
    screen.getByRole('textbox', { name: 'CraftSky handle' }),
    'maker.example',
  )
  await ui.click(
    screen.getByRole('button', {
      name: 'Continue to secure sign-in',
    }),
  )
  await screen.findByRole('heading', {
    name: 'Ready to import to CraftSky',
  })
}

async function startConfirmedImport(
  ui: ReturnType<typeof userEvent.setup>,
  appServices: ImporterUiServices,
): Promise<void> {
  await reachConfirmation(ui, appServices)
  await ui.click(
    screen.getByRole('checkbox', {
      name: /add these selected posts to this CraftSky account/i,
    }),
  )
  await ui.click(
    screen.getByRole('button', { name: /^Import \d+ posts?$/ }),
  )
}

describe('Instagram importer app', () => {
  it('shows CraftSky branding and the public footer links', () => {
    render(<App services={services()} />)

    const homeLink = screen.getByRole('link', {
      name: 'CraftSky importer home',
    })
    expect(homeLink).toHaveAttribute('href', '/')
    expect(homeLink.querySelector('img')).toHaveAttribute(
      'src',
      '/app_icon.png',
    )
    expect(homeLink).toHaveTextContent('CraftSky')

    expect(screen.getByRole('link', { name: 'Privacy' })).toHaveAttribute(
      'href',
      'https://craftsky.social/privacy',
    )
    expect(
      screen.getByRole('link', { name: 'Terms & Conditions' }),
    ).toHaveAttribute('href', 'https://craftsky.social/terms')
    expect(
      screen.getByRole('link', { name: 'CraftSky on GitHub' }),
    ).toHaveAttribute('href', 'https://github.com/Zambrella/craftsky')
    expect(screen.getByText('How it works')).toBeVisible()
    expect(screen.getByText('Review your posts')).toBeVisible()
    expect(
      screen.getByText('Connect your CraftSky account'),
    ).toBeVisible()
    expect(screen.getByText(/takes longer to process/u)).toBeVisible()
    expect(screen.queryByText(/\bPDS\b/u)).not.toBeInTheDocument()
  })

  it('returns to file selection without an error after inspection is cancelled', async () => {
    const ui = userEvent.setup()
    let rejectInspection:
      | ((reason?: unknown) => void)
      | undefined
    const inspect = vi.fn(
      () =>
        new Promise<ReviewManifest>((_resolve, reject) => {
          rejectInspection = reject
        }),
    )
    const cancelInspection = vi.fn(() => {
      rejectInspection?.(new Error('operationCanceled'))
    })
    const appServices = {
      ...services(),
      inspect,
      cancelInspection,
    }
    render(<App services={appServices} />)
    await ui.click(
      screen.getByRole('checkbox', {
        name: /selected posts and images will become public/i,
      }),
    )
    const fileInput = document.querySelector<HTMLInputElement>(
      'input[type="file"]',
    )
    expect(fileInput).not.toBeNull()
    await ui.upload(
      fileInput!,
      new File(['synthetic'], 'cancelled.zip', {
        type: 'application/zip',
      }),
    )

    await ui.click(
      await screen.findByRole('button', { name: 'Cancel' }),
    )

    await waitFor(() => {
      expect(cancelInspection).toHaveBeenCalledTimes(1)
      expect(
        screen.getByRole('heading', {
          name: 'Choose your Instagram export',
        }),
      ).toBeVisible()
      expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    })
  })

  it('reviews locally and authorizes only after the member continues (AT-003, IT-008)', async () => {
    const ui = userEvent.setup()
    const appServices = services()
    await loadReview(ui, appServices)

    expect(screen.queryByText('Local only')).not.toBeInTheDocument()
    expect(
      screen.queryByText('No account connected yet'),
    ).not.toBeInTheDocument()
    expect(appServices.inspect).toHaveBeenCalledTimes(1)
    expect(appServices.authorize).not.toHaveBeenCalled()
    expect(screen.queryByText('never-render-this-name.zip')).not.toBeInTheDocument()
    expect(screen.getByText('2 posts · 2 images')).toBeInTheDocument()
    expect(
      screen.queryByText('Caption text was repaired'),
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('group', { name: '0 need attention' }),
    ).toBeInTheDocument()

    await ui.click(
      screen.getByRole('button', {
        name: 'Connect your CraftSky account',
      }),
    )
    expect(appServices.authorize).not.toHaveBeenCalled()
    await ui.type(
      screen.getByRole('textbox', { name: 'CraftSky handle' }),
      'maker.example',
    )
    await ui.click(
      screen.getByRole('button', {
        name: 'Continue to secure sign-in',
      }),
    )

    await screen.findByRole('heading', {
      name: 'Ready to import to CraftSky',
    })
    expect(appServices.authorize).toHaveBeenCalledWith('maker.example')
    expect(appServices.publish).not.toHaveBeenCalled()
    expect(screen.queryByText('did:plc:synthetic')).not.toBeInTheDocument()
    expect(screen.getByText('Synthetic Maker')).toBeVisible()
    expect(screen.getByText('@maker.example')).toBeVisible()
    expect(
      screen.getByRole('img', {
        name: 'Synthetic Maker profile picture',
      }),
    ).toHaveAttribute('src', expect.stringContaining('data:image/png'))
    expect(screen.getByText('public posts')).toBeInTheDocument()
    expect(screen.getByText('public images')).toBeInTheDocument()
  })

  it('preloads OAuth metadata before enabling the user-activated popup action', async () => {
    const ui = userEvent.setup()
    let finishPreparation: (() => void) | undefined
    const prepareAuthorization = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finishPreparation = resolve
        }),
    )
    const appServices = {
      ...services(),
      prepareAuthorization,
    }
    await loadReview(ui, appServices)
    await ui.click(
      screen.getByRole('button', {
        name: 'Connect your CraftSky account',
      }),
    )
    await ui.type(
      screen.getByRole('textbox', { name: 'CraftSky handle' }),
      'maker.example',
    )

    const authorizeButton = screen.getByRole('button', {
      name: 'Continue to secure sign-in',
    })
    expect(prepareAuthorization).toHaveBeenCalledTimes(1)
    expect(authorizeButton).toBeDisabled()
    expect(
      screen.getByText('Preparing secure sign-in…'),
    ).toBeVisible()

    finishPreparation?.()
    await waitFor(() => expect(authorizeButton).toBeEnabled())
    await ui.click(authorizeButton)
    expect(appServices.authorize).toHaveBeenCalledTimes(1)
  })

  it('publishes only after exact destination confirmation (AT-005, IT-008)', async () => {
    const ui = userEvent.setup()
    const appServices = services(manifest(1))
    await reachConfirmation(ui, appServices)

    const importButton = await screen.findByRole('button', {
      name: 'Import 1 post',
    })
    expect(importButton).toBeDisabled()
    await ui.click(
      screen.getByRole('checkbox', {
        name: /add these selected posts to this CraftSky account/i,
      }),
    )
    await ui.click(importButton)

    await screen.findByRole('heading', {
      name: 'Your historical posts have been processed',
    })
    expect(appServices.beginImport).toHaveBeenCalledTimes(1)
    expect(appServices.publish).toHaveBeenCalledTimes(1)
    expect(appServices.finishImport).toHaveBeenCalledTimes(1)
    expect(screen.getByText('added')).toBeInTheDocument()
  })

  it('updates reviewed media counts and requires text-only confirmation', async () => {
    const ui = userEvent.setup()
    const appServices = services()
    await loadReview(ui, appServices)

    await ui.click(screen.getAllByText('Images (1)')[0])
    const imageOptions = screen.getAllByRole('checkbox', {
      name: /image 1/i,
    })
    await ui.click(imageOptions[0])
    expect(screen.getByText('2 posts · 1 image')).toBeInTheDocument()

    const continueButton = screen.getByRole('button', {
      name: 'Connect your CraftSky account',
    })
    expect(continueButton).toBeDisabled()
    const textOnly = screen.getByRole('checkbox', {
      name: /import this post without an image/i,
    })
    await ui.click(textOnly)
    expect(continueButton).toBeEnabled()

    await ui.click(screen.getAllByText('Caption')[0])
    const captions = screen.getAllByRole('textbox', { name: /caption/i })
    expect(captions[0]).toHaveAttribute('spellcheck', 'false')
    expect(captions[0]).toHaveAttribute('autocorrect', 'off')
    expect(captions[0]).toHaveAttribute('autocapitalize', 'off')
    await ui.clear(captions[0])
    await ui.type(captions[0], 'Edited locally')
    expect(captions[0]).toHaveValue('Edited locally')
  })

  it('shows the first sanitized image for visible posts until review closes', async () => {
    const ui = userEvent.setup()
    const createObjectURL = vi.fn().mockReturnValue('blob:safe-preview')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal(
      'URL',
      class TestURL extends URL {
        static createObjectURL = createObjectURL
        static revokeObjectURL = revokeObjectURL
      },
    )
    const previewMedia = vi
      .fn<(token: string, signal: AbortSignal) => Promise<Blob>>()
      .mockResolvedValue(new Blob(['sanitized'], { type: 'image/jpeg' }))
    const appServices = { ...services(manifest(2)), previewMedia }
    await loadReview(ui, appServices)

    await waitFor(() =>
      expect(previewMedia).toHaveBeenCalledWith(
        'media-0',
        expect.any(AbortSignal),
      ),
    )
    expect(previewMedia).toHaveBeenCalledWith(
      'media-1',
      expect.any(AbortSignal),
    )
    const firstPreviewSignal = previewMedia.mock.calls[0]?.[1]
    expect(firstPreviewSignal?.aborted).toBe(false)
    const secondPreviewSignal = previewMedia.mock.calls[1]?.[1]
    expect(secondPreviewSignal?.aborted).toBe(false)
    const thumbnails = await screen.findAllByRole('button', {
      name: 'View Image 1 full screen',
    })
    expect(thumbnails).toHaveLength(2)
    await ui.click(
      thumbnails[0],
    )
    expect(
      screen.getByRole('dialog', {
        name: 'Image 1 full-screen preview',
      }),
    ).toBeVisible()

    await ui.click(
      screen.getAllByRole('checkbox', { name: 'Include image 1' })[1],
    )
    expect(revokeObjectURL).not.toHaveBeenCalled()
    expect(secondPreviewSignal?.aborted).toBe(false)

    cleanup()
    await waitFor(() =>
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:safe-preview'),
    )
    expect(secondPreviewSignal?.aborted).toBe(true)
    expect(firstPreviewSignal?.aborted).toBe(true)
  })

  it('hard-disables real authorization in preview builds (AT-010, IT-016)', async () => {
    const ui = userEvent.setup()
    const appServices = {
      ...services(),
      runtime: {
        mode: 'preview',
        origin: 'https://branch.pages.dev',
        redirectUri: 'https://branch.pages.dev/oauth/callback',
        clientId: '',
        writesEnabled: false,
      } satisfies RuntimePolicy,
    }
    await loadReview(ui, appServices)
    expect(screen.getByText('Safe preview')).toBeInTheDocument()

    await ui.click(
      screen.getByRole('button', {
        name: 'Connect your CraftSky account',
      }),
    )
    await ui.type(
      screen.getByRole('textbox', { name: 'CraftSky handle' }),
      'maker.example',
    )
    expect(
      screen.getByRole('button', {
        name: 'Continue to secure sign-in',
      }),
    ).toBeDisabled()
    expect(appServices.authorize).not.toHaveBeenCalled()
  })

  it('pauses without scheduling another post (AT-006, AT-007; component slice of IT-018)', async () => {
    const ui = userEvent.setup()
    const pauseImport = vi.fn().mockResolvedValue(undefined)
    const publish = vi.fn(
      (_post: ReviewPost, signal: AbortSignal) =>
        new Promise<PublishOutcome>((resolve) => {
          signal.addEventListener(
            'abort',
            () => resolve({ status: 'failed' }),
            { once: true },
          )
        }),
    )
    const appServices = {
      ...services(manifest(2)),
      publish,
      pauseImport,
    }
    await startConfirmedImport(ui, appServices)
    await ui.click(
      await screen.findByRole('button', { name: 'Pause import' }),
    )

    await screen.findByRole('heading', { name: /of 2 posts processed/i })
    expect(screen.getByText('Import paused')).toBeInTheDocument()
    expect(publish).toHaveBeenCalledTimes(1)
    expect(pauseImport).toHaveBeenCalledTimes(1)
    expect(appServices.beginImport).toHaveBeenCalledTimes(1)
  })

  it('retries only failures and exposes guarded rollback and clear actions (AT-006, AT-007)', async () => {
    const ui = userEvent.setup()
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const rollback = vi.fn().mockResolvedValue({
      deleted: 1,
      absent: 0,
      conflicts: 0,
      failed: 0,
      remainingOwned: 0,
    })
    const clearHistory = vi.fn().mockResolvedValue(undefined)
    const publish = vi
      .fn<
        (
          post: ReviewPost,
          signal: AbortSignal,
        ) => Promise<PublishOutcome>
      >()
      .mockResolvedValueOnce({ status: 'failed' })
      .mockResolvedValueOnce({ status: 'created' })
    const appServices = {
      ...services(manifest(1)),
      publish,
      rollback,
      clearHistory,
    }
    await startConfirmedImport(ui, appServices)
    await screen.findByRole('heading', {
      name: 'Your historical posts have been processed',
    })
    expect(
      within(
        screen.getByLabelText('Per-post import progress'),
      ).getByText('Failed'),
    ).toBeVisible()

    await ui.click(
      screen.getByRole('button', { name: 'Retry failed posts' }),
    )
    await waitFor(() => expect(publish).toHaveBeenCalledTimes(2))
    expect(appServices.beginImport).toHaveBeenCalledTimes(1)
    await screen.findByRole('heading', {
      name: 'Your historical posts have been processed',
    })
    expect(
      within(
        screen.getByLabelText('Per-post import progress'),
      ).getByText('Added'),
    ).toBeVisible()

    await ui.click(
      screen.getByRole('button', { name: 'Remove imported posts' }),
    )
    await screen.findByText(
      '1 removed · 0 already absent · 0 changed and kept · 0 failed',
    )
    expect(rollback).toHaveBeenCalledTimes(1)

    await ui.click(
      screen.getByRole('button', { name: 'Clear import history' }),
    )
    await screen.findByText(
      'Import history cleared from this device.',
    )
    expect(clearHistory).toHaveBeenCalledTimes(1)
    expect(confirm).toHaveBeenCalledTimes(2)
    expect(confirm).toHaveBeenNthCalledWith(
      1,
      expect.stringContaining(
        'Posts you have changed since importing will be kept',
      ),
    )
  })

  it('retains recovery state and reports when local history cannot be cleared', async () => {
    const ui = userEvent.setup()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const clearHistory = vi
      .fn()
      .mockRejectedValue(new Error('unknown'))
    const appServices = {
      ...services(manifest(1)),
      clearHistory,
      rollback: vi.fn(),
    }
    await startConfirmedImport(ui, appServices)
    await screen.findByRole('heading', {
      name: 'Your historical posts have been processed',
    })

    await ui.click(
      screen.getByRole('button', { name: 'Clear import history' }),
    )

    expect(
      await screen.findByText(
        'Your import history could not be cleared. Resume and removal information remains on this device.',
      ),
    ).toBeVisible()
    expect(
      screen.getByRole('button', {
        name: 'Remove imported posts',
      }),
    ).toBeVisible()
    expect(clearHistory).toHaveBeenCalledTimes(1)
  })

  it('offers CID-guarded rollback for a paused partial import and reports each post (AT-007, FR-027, FR-028)', async () => {
    const ui = userEvent.setup()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const pauseImport = vi.fn().mockResolvedValue(undefined)
    const rollback = vi.fn().mockResolvedValue({
      deleted: 0,
      absent: 1,
      conflicts: 0,
      failed: 0,
      remainingOwned: 0,
      itemStatuses: {
        'item-0': 'rollbackAbsent',
        'item-1': 'pending',
      } as const,
    })
    const publish = vi.fn(
      (_post: ReviewPost, signal: AbortSignal) =>
        new Promise<PublishOutcome>((resolve) => {
          signal.addEventListener(
            'abort',
            () => resolve({ status: 'created' }),
            { once: true },
          )
        }),
    )
    const appServices = {
      ...services(manifest(2)),
      beginImport: vi.fn().mockResolvedValue({
        ownedCreated: 1,
        itemStatuses: {
          'item-0': 'created',
          'item-1': 'pending',
        },
      }),
      publish,
      pauseImport,
      rollback,
    }
    await startConfirmedImport(ui, appServices)
    await ui.click(
      await screen.findByRole('button', { name: 'Pause import' }),
    )
    await screen.findByText('Import paused')

    const perPost = screen.getByLabelText(
      'Per-post import progress',
    )
    expect(within(perPost).getByText('Post 1')).toBeVisible()
    expect(within(perPost).getByText('Post 2')).toBeVisible()
    expect(
      screen.getByRole('button', {
        name: 'Remove imported posts',
      }),
    ).toBeVisible()

    await ui.click(
      screen.getByRole('button', {
        name: 'Remove imported posts',
      }),
    )
    await screen.findByText(
      '0 removed · 1 already absent · 0 changed and kept · 0 failed',
    )
    expect(rollback).toHaveBeenCalledTimes(1)
    expect(
      within(
        screen.getByLabelText('Per-post import progress'),
      ).getByText('Already absent'),
    ).toBeVisible()
  })

  it('signs out after completion without clearing local history (FR-028, IT-005)', async () => {
    const ui = userEvent.setup()
    const signOut = vi.fn().mockResolvedValue(undefined)
    const clearHistory = vi.fn().mockResolvedValue(undefined)
    const appServices = {
      ...services(manifest(1)),
      signOut,
      clearHistory,
    }
    await startConfirmedImport(ui, appServices)
    await screen.findByRole('heading', {
      name: 'Your historical posts have been processed',
    })

    await ui.click(
      screen.getByRole('button', {
        name: 'Disconnect CraftSky account',
      }),
    )

    await screen.findByText(
      'CraftSky account disconnected. Import history remains on this device.',
    )
    expect(signOut).toHaveBeenCalledTimes(1)
    expect(clearHistory).not.toHaveBeenCalled()
  })

  it('keeps the confirmed destination when switching accounts cannot sign out', async () => {
    const ui = userEvent.setup()
    const signOut = vi.fn().mockRejectedValue(new Error('unknown'))
    const appServices = {
      ...services(manifest(1)),
      signOut,
    }
    await reachConfirmation(ui, appServices)

    await ui.click(
      screen.getByRole('button', { name: 'Use another account' }),
    )

    expect(
      await screen.findByText(
        'Your CraftSky account could not be disconnected and may still be connected. Import progress remains on this device.',
      ),
    ).toBeVisible()
    expect(
      screen.getByRole('heading', {
        name: 'Ready to import to CraftSky',
      }),
    ).toBeVisible()
    expect(signOut).toHaveBeenCalledTimes(1)
  })
})
