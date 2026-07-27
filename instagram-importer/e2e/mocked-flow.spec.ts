import { expect, test } from '@playwright/test'

declare global {
  interface Window {
    __importerHarness: {
      authorize: number
      beginImport: number
      publish: number
      finishImport: number
      rollback: number
      clearHistory: number
      signOut: number
    }
  }
}

test('completes the mocked importer UI flow (IT-008; browser UI slices for AT-003, AT-005, AT-006, AT-007)', async ({
  page,
}) => {
  const externalRequests: string[] = []
  page.on('request', (request) => {
    if (new URL(request.url()).origin !== 'http://127.0.0.1:4173') {
      externalRequests.push(request.url())
    }
  })
  page.on('dialog', (dialog) => dialog.accept())
  await page.goto('/test-harness.html')

  await page
    .getByRole('checkbox', {
      name: /selected posts and images will become public/i,
    })
    .check()
  await page.locator('input[type=file]').setInputFiles({
    name: 'synthetic.zip',
    mimeType: 'application/zip',
    buffer: Buffer.from('synthetic'),
  })
  await page.getByText('Images (1)').click()
  const thumbnail = page.getByRole('button', {
    name: 'View Image 1 full screen',
  })
  await expect(thumbnail).toHaveCSS('width', '144px')
  await expect(thumbnail).toHaveCSS('height', '144px')
  await thumbnail.click()
  await expect(
    page.getByRole('dialog', {
      name: 'Image 1 full-screen preview',
    }),
  ).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(
    page.getByRole('dialog', {
      name: 'Image 1 full-screen preview',
    }),
  ).toHaveCount(0)
  await expect(thumbnail).toBeFocused()

  await page.getByRole('button', { name: 'Connect your PDS' }).click()
  await page
    .getByRole('textbox', { name: 'Handle or DID' })
    .fill('maker.example')
  await page
    .getByRole('button', { name: 'Authorize with your PDS' })
    .click()
  await expect(page.getByText('did:plc:browser-harness')).toBeVisible()
  await expect(page.getByLabel('1 public posts')).toBeVisible()
  await expect(page.getByLabel('1 public images')).toBeVisible()

  await page
    .getByRole('checkbox', {
      name: /publish these selected posts to this account/i,
    })
    .check()
  await page.getByRole('button', { name: 'Import 1 post' }).click()
  await expect(
    page.getByRole('heading', {
      name: 'Your historical posts have been processed',
    }),
  ).toBeVisible()
  await page.getByRole('button', { name: 'Retry failed posts' }).click()
  await expect(page.getByLabel('1 created')).toBeVisible()

  expect(
    await page.evaluate(() => ({
      beginImport: window.__importerHarness.beginImport,
      publish: window.__importerHarness.publish,
    })),
  ).toEqual({ beginImport: 1, publish: 2 })

  await page
    .getByRole('button', { name: 'Roll back this import' })
    .click()
  await expect(
    page.getByText(
      '0 removed · 1 already absent · 0 changed and kept · 0 failed',
    ),
  ).toBeVisible()
  await expect(
    page.getByText('Already absent', { exact: true }),
  ).toBeVisible()
  await page.getByRole('button', { name: 'Sign out of PDS' }).click()
  await expect(
    page.getByText(
      'Signed out. Local progress and rollback history remain on this device.',
    ),
  ).toBeVisible()
  expect(await page.evaluate(() => window.__importerHarness)).toMatchObject(
    {
      authorize: 1,
      beginImport: 1,
      publish: 2,
      finishImport: 2,
      rollback: 1,
      clearHistory: 0,
      signOut: 1,
    },
  )
  expect(externalRequests).toEqual([])
})
