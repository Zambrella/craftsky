const { test, expect } = require('@playwright/test');

test('the built schema page serves its interactive Lexicon field lists', async ({ page }) => {
  test.skip(process.env.WEB_TEST_ROOT !== 'dist', 'The website build generates this page.');
  const response = await page.goto('/project-schema.html');
  expect(response.status()).toBe(200);
  await expect(page).toHaveURL('/project-schema.html');
  await expect(page.getByRole('heading', { name: 'CraftSky project schema' })).toBeVisible();

  const field = page.getByRole('button', { name: /^colors\b/ });
  await expect(field).toHaveCount(0);
  await page.getByRole('checkbox', { name: 'Show legacy fields' }).check();
  await expect(field).toContainText('Legacy');
  await field.hover();
  const options = page.getByRole('list', { name: 'Declared values' });
  await expect(options.getByText('teal', { exact: true })).toBeVisible();

  await page.getByRole('button', { name: 'Knitting', exact: true }).click();
  await expect(page.getByRole('button', { name: /^needleSizeMm\b/ })).toBeVisible();
});

test('the schema page shows up to five shared pattern objects', async ({ page }) => {
  test.skip(process.env.WEB_TEST_ROOT !== 'dist', 'The website build generates this page.');
  await page.goto('/project-schema.html');

  await expect(page.getByRole('button', { name: /^pattern\b/ })).toHaveCount(0);
  const patterns = page.getByRole('button', { name: /^patterns\b/ });
  await expect(patterns).toContainText('pattern[] · ≤5');
  await patterns.click();

  const reference = page.getByRole('dialog');
  await expect(reference.getByRole('heading', { name: 'Pattern', exact: true })).toBeVisible();
  await expect(reference.getByRole('button', { name: /^nameFacets\b/ })).toBeVisible();
  await expect(reference.getByRole('button', { name: /^designerFacets\b/ })).toBeVisible();
  await expect(reference.getByRole('button', { name: /^publisherFacets\b/ })).toBeVisible();
});

test('the schema page shows the proposed craft fields and methods', async ({ page }) => {
  test.skip(process.env.WEB_TEST_ROOT !== 'dist', 'The website build generates this page.');
  await page.goto('/project-schema.html');

  const craft = name => page.getByRole('group', { name: 'Craft' })
    .getByRole('button', { name, exact: true });
  const field = (container, name) => container.getByRole('button', {
    name: new RegExp(`^${name}(?:required)?\\b`),
  });
  const values = page.getByRole('list', { name: 'Declared values' });

  await craft('Knitting').click();
  await expect(field(page, 'projectSubtype')).toBeVisible();
  await expect(field(page, 'needleSizesMm')).toHaveCount(0);
  await field(page, 'methods').hover();
  await expect(values.getByText('hand', { exact: true })).toBeVisible();
  await expect(values.getByText('machine', { exact: true })).toBeVisible();
  await expect(values.getByText('loom', { exact: true })).toBeVisible();

  await craft('Crochet').click();
  await expect(field(page, 'hookSizesMm')).toHaveCount(0);
  await field(page, 'methods').hover();
  await expect(values.getByText('standardCrochet', { exact: true })).toBeVisible();
  await expect(values.getByText('tunisianCrochet', { exact: true })).toBeVisible();

  await craft('Embroidery').click();
  await expect(field(page, 'methods')).toHaveCount(0);
  await expect(field(page, 'method')).toContainText('string');
  await field(page, 'method').hover();
  await expect(values.getByRole('listitem')).toHaveText(['hand', 'machine']);
  await expect(field(page, 'style')).toContainText('string');
  await field(page, 'style').hover();
  await expect(values.getByRole('listitem')).toHaveText(['crossStitch', 'needlepoint']);

  await craft('Felting').click();
  await field(page, 'methods').hover();
  await expect(values.getByText('needle', { exact: true })).toBeVisible();
  await expect(values.getByText('wet', { exact: true })).toBeVisible();
  await expect(field(page, 'projectSubtype')).toBeVisible();

  for (const name of ['Sewing', 'Quilting', 'Punch Needle', 'Macramé']) {
    await craft(name).click();
    await expect(field(page, 'projectType')).toBeVisible();
  }
  await craft('Weaving').click();
  await expect(field(page, 'loomType')).toBeVisible();
  await expect(field(page, 'sett')).toHaveCount(0);

  await craft('Spinning').click();
  await expect(page.getByText('This craft uses the shared project fields for new posts.')).toBeVisible();
  await craft('Dyeing').click();
  await expect(field(page, 'projectType')).toBeVisible();
  await expect(field(page, 'materialForm')).toHaveCount(0);
  await page.getByRole('checkbox', { name: 'Show legacy fields' }).check();
  await expect(craft('Dyeing')).toHaveAttribute('aria-pressed', 'true');
  await expect(field(page, 'materialForm')).toContainText('Legacy');
  await craft('Embroidery').click();
  await expect(field(page, 'methods')).toContainText('Legacy');
});

test('the schema preview groups item subtypes under their project types', async ({ page }) => {
  test.skip(process.env.WEB_TEST_ROOT !== 'dist', 'The website build generates this page.');
  await page.goto('/project-schema.html');
  await page.getByRole('button', { name: /^projectType\b/ }).hover();
  await expect(page.getByRole('list', { name: 'Declared values' }).getByText('Other', { exact: true })).toBeVisible();

  await page.getByRole('button', { name: /^projectSubtype\b/ }).hover();
  const garments = page.getByRole('list', { name: 'Garment', exact: true });
  await expect(garments.getByText('Dungarees', { exact: true })).toBeVisible();
  await expect(garments.getByText('Coat or jacket', { exact: true })).toBeVisible();
  await expect(page.getByRole('list', { name: 'Accessory', exact: true }).getByText('Backpack', { exact: true })).toBeVisible();

  await page.getByRole('button', { name: 'Knitting', exact: true }).click();
  await page.getByRole('button', { name: /^projectSubtype\b/ }).hover();
  await expect(garments.getByText('Jumper', { exact: true })).toBeVisible();

  await page.getByRole('button', { name: 'Punch Needle', exact: true }).click();
  const customItem = page.getByRole('button', { name: /^projectSubtype\b/ });
  await expect(customItem).not.toContainText('values');
  await customItem.click();
  await expect(page.getByRole('region', { name: 'projectSubtype' })).toContainText(/custom/i);
});
