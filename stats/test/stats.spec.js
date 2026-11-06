const { test, expect } = require('@playwright/test');

const API = 'https://appview.craftsky.social/v1/public/stats';
const craft = (name) => `social.craftsky.feed.defs#${name}`;
const snapshot = {
  generatedAt: '2026-11-06T12:00:00Z',
  accounts: 1234,
  posts: 32001,
  projectPosts: 1000,
  finishedProjectPosts: 607,
  // Deliberately unsorted: the page should rank the chart by count.
  projectsByCraft: [
    { craft: craft('sewing'), count: 340 },
    { craft: craft('embroidery'), count: 0 },
    { craft: craft('knitting'), count: 660 },
  ],
};

async function respond(page, payload = snapshot) {
  await page.route(API, (route) => route.fulfill({ json: payload }));
}

async function ready(page) {
  await expect(page.locator('#stats-content')).toHaveAttribute('data-state', 'ready');
}

// Report numbers in their final state, independently of animation timing.
async function expectNumber(page, field, text) {
  await expect(page.locator(`[data-stat="${field}"]`)).toHaveText(text);
}

test('shows the fetched snapshot, ranks crafts, and links the join CTA', async ({ page }) => {
  await respond(page);
  await page.goto('/');
  await ready(page);
  await expect(page.getByRole('heading', { name: /CraftSky\s+in numbers\./ })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Headline numbers', exact: true })).toBeVisible();
  await expectNumber(page, 'accounts', '1,234');
  await expectNumber(page, 'posts', '32,001');
  await expectNumber(page, 'projectPosts', '1,000');
  await expectNumber(page, 'finishedProjectPosts', '607');
  const chart = page.getByTestId('craft-chart');
  await expect(page.getByRole('heading', { name: 'Projects by craft', exact: true })).toBeVisible();
  const rows = chart.locator('[data-craft]');
  await expect(rows.nth(0)).toHaveAttribute('data-craft', craft('knitting'));
  await expect(rows.nth(1)).toHaveAttribute('data-craft', craft('sewing'));
  await expect(chart.locator(`[data-craft="${craft('knitting')}"]`)).toContainText('660');
  await expect(chart.locator(`[data-craft="${craft('sewing')}"]`)).toContainText('340');
  const join = page.getByRole('link', { name: /join craftsky/i });
  await expect(join.last()).toHaveAttribute('href', 'https://app.craftsky.social');
});

test('renders open craft tokens safely, including inherited object names', async ({ page }) => {
  const markup = '<img src=x onerror="window.pwned=true">';
  await respond(page, {
    ...snapshot,
    projectsByCraft: [
      { craft: craft('knitting'), count: 500 },
      { craft: craft('constructor'), count: 500 },
      { craft: craft(markup), count: 0 },
    ],
  });
  await page.goto('/');
  await ready(page);
  const chart = page.getByTestId('craft-chart');
  await expect(chart.locator('[data-craft]').nth(0)).toHaveAttribute('data-craft', craft('constructor'));
  await expect(chart.getByText('Constructor', { exact: true })).toBeVisible();
  await expect(chart.getByText(markup, { exact: true })).toBeVisible();
  await expect(chart.locator('img, script')).toHaveCount(0);
  expect(await page.evaluate(() => window.pwned)).toBeUndefined();
});

test('keeps loading separate from genuine zero data', async ({ page }) => {
  let release;
  const gate = new Promise((resolve) => { release = resolve; });
  await page.route(API, async (route) => {
    await gate;
    await route.fulfill({ json: snapshot });
  });
  await page.goto('/');
  await expect(page.locator('#stats-content')).toHaveAttribute('data-state', 'loading');
  await expect(page.locator('[data-stat="accounts"]')).not.toHaveText('0');
  release();
  await ready(page);
});

test('renders an honest empty network without invalid bars or fake numbers', async ({ page }) => {
  await respond(page, {
    generatedAt: snapshot.generatedAt,
    accounts: 0, posts: 0, projectPosts: 0, finishedProjectPosts: 0,
    projectsByCraft: [],
  });
  await page.goto('/');
  await ready(page);
  for (const field of ['accounts', 'posts', 'projectPosts', 'finishedProjectPosts']) {
    await expectNumber(page, field, '0');
  }
  await expect(page.getByTestId('craft-chart')).toContainText(/no project posts|first project|yet/i);
  await expect(page.locator('body')).not.toContainText(/NaN|Infinity/);
});

test('shows an error and retries the one public endpoint', async ({ page }) => {
  let attempts = 0;
  await page.route(API, (route) => {
    attempts += 1;
    return attempts === 1
      ? route.fulfill({ status: 503, json: { error: 'unavailable', message: 'Busy', requestId: 'test' } })
      : route.fulfill({ json: snapshot });
  });
  await page.goto('/');
  await expect(page.locator('#stats-content')).toHaveAttribute('data-state', 'error');
  await page.getByRole('button', { name: 'Try again', exact: true }).click();
  await ready(page);
  await expectNumber(page, 'accounts', '1,234');
  expect(attempts).toBe(2);
  await expect(page.locator('#headline-numbers')).toBeFocused();
});

test('rejects malformed data instead of displaying partial statistics', async ({ page }) => {
  await respond(page, { ...snapshot, projectPosts: -1 });
  await page.goto('/');
  await expect(page.locator('#stats-content')).toHaveAttribute('data-state', 'error');
  await expect(page.getByRole('button', { name: 'Try again', exact: true })).toBeVisible();
  await expect(page.locator('body')).not.toContainText('NaN');
});

test('respects reduced motion and reads statistics without cookies or credentials', async ({ page, context }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await context.addCookies([{ name: 'private-session', value: 'canary', domain: 'appview.craftsky.social', path: '/' }]);
  let requestHeaders;
  await page.route(API, async (route) => {
    requestHeaders = await route.request().allHeaders();
    await route.fulfill({ json: snapshot });
  });
  await page.goto('/');
  await ready(page);
  await expectNumber(page, 'posts', '32,001');
  expect(requestHeaders.authorization).toBeUndefined();
  expect(requestHeaders.cookie).toBeUndefined();
  expect(requestHeaders['x-craftsky-device-id']).toBeUndefined();
  expect(await page.evaluate(() => document.getAnimations().some((animation) => animation.playState === 'running'))).toBe(false);
});

for (const width of [320, 390, 768, 1440]) {
  test(`fits a ${width}px viewport without horizontal overflow`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await respond(page);
    await page.goto('/');
    await ready(page);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await expect(page.getByRole('heading', { name: 'Projects by craft', exact: true })).toBeVisible();
  });
}
