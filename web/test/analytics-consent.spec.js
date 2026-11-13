const { test, expect } = require('@playwright/test');

const publicRoutes = [
  '/',
  '/waitlist',
  '/privacy',
  '/terms',
  '/community-guidelines',
  '/reporting',
  '/copyright',
  '/safety',
  '/instagram-import',
  '/why-instagram-sucks-for-crafters',
];
const postHogHost = 'https://t.craftsky.social';
const consentKey = 'craftsky.analytics-consent';

// Exercise local scripts without depending on fonts, Brevo or Turnstile.
// Individual tests install a more specific PostHog stub after this route.
test.beforeEach(async ({ context }) => {
  await context.route('**/*', route => {
    const url = new URL(route.request().url());
    return url.origin === 'http://127.0.0.1:4173'
      ? route.continue()
      : route.abort('blockedbyclient');
  });
});

async function observePostHog(context) {
  const requests = [];
  context.on('request', request => {
    if (request.url().startsWith(postHogHost)) requests.push(request.url());
  });
  await context.route(`${postHogHost}/**`, route => route.fulfill({
    status: 200,
    contentType: 'application/javascript',
    body: 'window.__craftskyPostHogTestScriptLoaded = true;',
  }));
  return requests;
}

async function postHogStorage(page, context) {
  const storage = await page.evaluate(() => ({
    local: Object.keys(localStorage),
    session: Object.keys(sessionStorage),
  }));
  const cookies = await context.cookies();
  return {
    local: storage.local.filter(key => key !== consentKey && /posthog|ph_/i.test(key)),
    session: storage.session.filter(key => /posthog|ph_/i.test(key)),
    cookies: cookies.filter(cookie => /posthog|ph_/i.test(cookie.name)),
  };
}

for (const route of publicRoutes) {
  test(`withholds PostHog before consent on ${route}`, async ({ page, context }) => {
    const requests = await observePostHog(context);

    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    const response = await page.goto(route);
    expect(response.ok()).toBe(true);
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    expect(errors).toEqual([]);

    expect(requests).toEqual([]);
    expect(await page.locator(`script[src^="${postHogHost}"]`).count()).toBe(0);
    expect(await postHogStorage(page, context)).toEqual({local: [], session: [], cookies: []});
  });
}

test('denial persists a consent choice but creates no PostHog request or identifier', async ({ page, context }) => {
  await context.addInitScript(() => {
    Object.defineProperty(Navigator.prototype, 'doNotTrack', {configurable: true, get: () => '0'});
  });
  const requests = await observePostHog(context);
  await page.goto('/');
  await page.getByRole('button', {name: 'No thanks'}).click();

  expect(await page.evaluate(key => localStorage.getItem(key), consentKey)).toBe('denied');
  expect(requests).toEqual([]);
  expect(await postHogStorage(page, context)).toEqual({local: [], session: [], cookies: []});
});

test('valid consent loads PostHog only from the approved host', async ({ page, context }) => {
  await context.addInitScript(() => {
    Object.defineProperty(Navigator.prototype, 'doNotTrack', {configurable: true, get: () => '0'});
  });
  const requests = await observePostHog(context);
  await page.goto('/');
  await page.getByRole('button', {name: 'Allow analytics'}).click();

  await expect.poll(() => requests.length).toBe(1);
  expect(requests[0]).toBe(`${postHogHost}/static/array.js`);
  expect(await page.evaluate(key => localStorage.getItem(key), consentKey)).toBe('granted');
});

test('Do Not Track overrides a stored grant', async ({ page, context }) => {
  await context.addInitScript(key => {
    Object.defineProperty(Navigator.prototype, 'doNotTrack', {configurable: true, get: () => '1'});
    localStorage.setItem(key, 'granted');
  }, consentKey);
  const requests = await observePostHog(context);
  await page.goto('/');

  expect(requests).toEqual([]);
  expect(await page.locator(`script[src^="${postHogHost}"]`).count()).toBe(0);
  expect(await postHogStorage(page, context)).toEqual({local: [], session: [], cookies: []});
});
