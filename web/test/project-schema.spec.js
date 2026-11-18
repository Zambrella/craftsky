const { test, expect } = require('@playwright/test');

test('the built schema page serves its interactive Lexicon field lists', async ({ page }) => {
  test.skip(process.env.WEB_TEST_ROOT !== 'dist', 'The website build generates this page.');
  const response = await page.goto('/project-schema.html');
  expect(response.status()).toBe(200);
  await expect(page).toHaveURL('/project-schema.html');
  await expect(page.getByRole('heading', { name: 'CraftSky project schema' })).toBeVisible();

  const field = page.getByRole('button', { name: /^colors\b/ });
  await field.hover();
  const options = page.getByRole('list', { name: 'Declared values' });
  await expect(options.getByText('teal', { exact: true })).toBeVisible();

  await page.getByRole('button', { name: 'Knitting', exact: true }).click();
  await expect(page.getByRole('button', { name: /^needleSizeMm\b/ })).toBeVisible();
});
