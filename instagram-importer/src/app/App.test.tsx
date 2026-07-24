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
    authorize: vi.fn().mockResolvedValue({ did: 'did:plc:synthetic' }),
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
  await ui.click(screen.getByRole('button', { name: 'Connect your PDS' }))
  await ui.type(
    screen.getByRole('textbox', { name: 'Handle or DID' }),
    'maker.example',
  )
  await ui.click(
    screen.getByRole('button', { name: 'Authorize with your PDS' }),
  )
  await screen.findByRole('heading', {
    name: 'Ready to publish to your PDS',
  })
}

async function startConfirmedImport(
  ui: ReturnType<typeof userEvent.setup>,
  appServices: ImporterUiServices,
): Promise<void> {
  await reachConfirmation(ui, appServices)
  await ui.click(
    screen.getByRole('checkbox', {
      name: /publish these selected posts to this account/i,
    }),
  )
  await ui.click(
    screen.getByRole('button', { name: /^Import \d+ posts?$/ }),
  )
}

describe('Instagram importer app', () => {
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

    expect(appServices.inspect).toHaveBeenCalledTimes(1)
    expect(appServices.authorize).not.toHaveBeenCalled()
    expect(screen.queryByText('never-render-this-name.zip')).not.toBeInTheDocument()
    expect(screen.getByText('2 posts · 2 images')).toBeInTheDocument()
    expect(
      screen.getByText('Caption text was repaired'),
    ).toBeInTheDocument()

    await ui.click(screen.getByRole('button', { name: 'Connect your PDS' }))
    expect(appServices.authorize).not.toHaveBeenCalled()
    await ui.type(
      screen.getByRole('textbox', { name: 'Handle or DID' }),
      'maker.example',
    )
    await ui.click(
      screen.getByRole('button', { name: 'Authorize with your PDS' }),
    )

    await screen.findByRole('heading', {
      name: 'Ready to publish to your PDS',
    })
    expect(appServices.authorize).toHaveBeenCalledWith('maker.example')
    expect(appServices.publish).not.toHaveBeenCalled()
    expect(screen.getByText('did:plc:synthetic')).toBeInTheDocument()
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
      screen.getByRole('button', { name: 'Connect your PDS' }),
    )
    await ui.type(
      screen.getByRole('textbox', { name: 'Handle or DID' }),
      'maker.example',
    )

    const authorizeButton = screen.getByRole('button', {
      name: 'Authorize with your PDS',
    })
    expect(prepareAuthorization).toHaveBeenCalledTimes(1)
    expect(authorizeButton).toBeDisabled()
    expect(
      screen.getByText('Preparing the secure PDS connection…'),
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
        name: /publish these selected posts to this account/i,
      }),
    )
    await ui.click(importButton)

    await screen.findByRole('heading', {
      name: 'Your historical posts have been processed',
    })
    expect(appServices.beginImport).toHaveBeenCalledTimes(1)
    expect(appServices.publish).toHaveBeenCalledTimes(1)
    expect(appServices.finishImport).toHaveBeenCalledTimes(1)
    expect(screen.getByText('created')).toBeInTheDocument()
  })

  it('updates reviewed media counts and requires text-only confirmation', async () => {
    const ui = userEvent.setup()
    const appServices = services()
    await loadReview(ui, appServices)

    const imageOptions = screen.getAllByRole('checkbox', {
      name: /image 1/i,
    })
    await ui.click(imageOptions[0])
    expect(screen.getByText('2 posts · 1 image')).toBeInTheDocument()

    const continueButton = screen.getByRole('button', {
      name: 'Connect your PDS',
    })
    expect(continueButton).toBeDisabled()
    const textOnly = screen.getByRole('checkbox', {
      name: /import this as a text-only post/i,
    })
    await ui.click(textOnly)
    expect(continueButton).toBeEnabled()

    const captions = screen.getAllByRole('textbox', { name: /caption/i })
    expect(captions[0]).toHaveAttribute('spellcheck', 'false')
    expect(captions[0]).toHaveAttribute('autocorrect', 'off')
    expect(captions[0]).toHaveAttribute('autocapitalize', 'off')
    await ui.clear(captions[0])
    await ui.type(captions[0], 'Edited locally')
    expect(captions[0]).toHaveValue('Edited locally')
  })

  it('shows only sanitized visible-item previews and revokes them on deselection', async () => {
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

    expect(previewMedia).not.toHaveBeenCalled()
    await ui.click(
      screen.getAllByRole('button', { name: 'Preview image 1' })[0],
    )
    const preview = await screen.findByRole('img', {
      name: 'Image 1 preview',
    })
    expect(preview).toHaveAttribute('src', 'blob:safe-preview')
    expect(previewMedia).toHaveBeenCalledWith(
      'media-0',
      expect.any(AbortSignal),
    )
    const firstPreviewSignal = previewMedia.mock.calls[0]?.[1]
    expect(firstPreviewSignal?.aborted).toBe(false)

    await ui.click(
      screen.getByRole('button', { name: 'Preview image 1' }),
    )
    await waitFor(() =>
      expect(previewMedia).toHaveBeenCalledWith(
        'media-1',
        expect.any(AbortSignal),
      ),
    )
    expect(firstPreviewSignal?.aborted).toBe(true)
    const secondPreviewSignal = previewMedia.mock.calls[1]?.[1]
    expect(secondPreviewSignal?.aborted).toBe(false)
    expect(
      screen.getAllByRole('img', { name: 'Image 1 preview' }),
    ).toHaveLength(1)
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:safe-preview')

    await ui.click(
      screen.getAllByRole('checkbox', { name: 'Include image 1' })[1],
    )
    await waitFor(() =>
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:safe-preview'),
    )
    expect(secondPreviewSignal?.aborted).toBe(true)
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

    await ui.click(screen.getByRole('button', { name: 'Connect your PDS' }))
    await ui.type(
      screen.getByRole('textbox', { name: 'Handle or DID' }),
      'maker.example',
    )
    expect(
      screen.getByRole('button', { name: 'Authorize with your PDS' }),
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
        screen.getByLabelText('Per-post publication progress'),
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
        screen.getByLabelText('Per-post publication progress'),
      ).getByText('Created'),
    ).toBeVisible()

    await ui.click(
      screen.getByRole('button', { name: 'Roll back this import' }),
    )
    await screen.findByText(
      '1 removed · 0 already absent · 0 changed and kept · 0 failed',
    )
    expect(rollback).toHaveBeenCalledTimes(1)

    await ui.click(
      screen.getByRole('button', { name: 'Clear local history' }),
    )
    await screen.findByText(
      'Local progress and rollback history cleared.',
    )
    expect(clearHistory).toHaveBeenCalledTimes(1)
    expect(confirm).toHaveBeenCalledTimes(2)
    expect(confirm).toHaveBeenNthCalledWith(
      1,
      expect.stringContaining(
        'separately recreated identical record',
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
      screen.getByRole('button', { name: 'Clear local history' }),
    )

    expect(
      await screen.findByText(
        'Local history could not be cleared. Resume and rollback information remains on this device.',
      ),
    ).toBeVisible()
    expect(
      screen.getByRole('button', {
        name: 'Roll back this import',
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
      'Per-post publication progress',
    )
    expect(within(perPost).getByText('Post 1')).toBeVisible()
    expect(within(perPost).getByText('Post 2')).toBeVisible()
    expect(
      screen.getByRole('button', {
        name: 'Roll back this partial import',
      }),
    ).toBeVisible()

    await ui.click(
      screen.getByRole('button', {
        name: 'Roll back this partial import',
      }),
    )
    await screen.findByText(
      '0 removed · 1 already absent · 0 changed and kept · 0 failed',
    )
    expect(rollback).toHaveBeenCalledTimes(1)
    expect(
      within(
        screen.getByLabelText('Per-post publication progress'),
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
      screen.getByRole('button', { name: 'Sign out of PDS' }),
    )

    await screen.findByText(
      'Signed out. Local progress and rollback history remain on this device.',
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
        'Your PDS session could not be cleared and may still be active. Local import progress remains on this device.',
      ),
    ).toBeVisible()
    expect(
      screen.getByRole('heading', {
        name: 'Ready to publish to your PDS',
      }),
    ).toBeVisible()
    expect(signOut).toHaveBeenCalledTimes(1)
  })
})
