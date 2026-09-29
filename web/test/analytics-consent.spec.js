const { test, expect } = require('@playwright/test');

const publicRoutes = [
  '/',
  '/waitlist',
  '/privacy',
  '/terms',
  '/community-guidelines',
  '/reporting',
  '/copyright',
];
const postHogHost = 'https://t.craftsky.social';
const consentKey = 'craftsky.analytics-consent';

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
  test(`withholds PostHog before consent on ${route}`, async ({ browser }) => {
    const context = await browser.newContext();
    const requests = await observePostHog(context);
    const page = await context.newPage();

    await page.goto(route);
    await page.waitForLoadState('networkidle');

    expect(requests).toEqual([]);
    expect(await page.locator(`script[src^="${postHogHost}"]`).count()).toBe(0);
    expect(await postHogStorage(page, context)).toEqual({local: [], session: [], cookies: []});
    await context.close();
  });
}

test('denial persists a consent choice but creates no PostHog request or identifier', async ({ browser }) => {
  const context = await browser.newContext();
  await context.addInitScript(() => {
    Object.defineProperty(Navigator.prototype, 'doNotTrack', {configurable: true, get: () => '0'});
  });
  const requests = await observePostHog(context);
  const page = await context.newPage();
  await page.goto('/');
  await page.getByRole('button', {name: 'No thanks'}).click();

  expect(await page.evaluate(key => localStorage.getItem(key), consentKey)).toBe('denied');
  expect(requests).toEqual([]);
  expect(await postHogStorage(page, context)).toEqual({local: [], session: [], cookies: []});
  await context.close();
});

test('valid consent loads PostHog only from the approved host', async ({ browser }) => {
  const context = await browser.newContext();
  await context.addInitScript(() => {
    Object.defineProperty(Navigator.prototype, 'doNotTrack', {configurable: true, get: () => '0'});
  });
  const requests = await observePostHog(context);
  const page = await context.newPage();
  await page.goto('/');
  await page.getByRole('button', {name: 'Allow analytics'}).click();

  await expect.poll(() => requests.length).toBe(1);
  expect(requests[0]).toBe(`${postHogHost}/static/array.js`);
  expect(await page.evaluate(key => localStorage.getItem(key), consentKey)).toBe('granted');
  await context.close();
});

test('Do Not Track overrides a stored grant', async ({ browser }) => {
  const context = await browser.newContext();
  await context.addInitScript(key => {
    Object.defineProperty(Navigator.prototype, 'doNotTrack', {configurable: true, get: () => '1'});
    localStorage.setItem(key, 'granted');
  }, consentKey);
  const requests = await observePostHog(context);
  const page = await context.newPage();
  await page.goto('/');
  await page.waitForLoadState('networkidle');

  expect(requests).toEqual([]);
  expect(await page.locator(`script[src^="${postHogHost}"]`).count()).toBe(0);
  expect(await postHogStorage(page, context)).toEqual({local: [], session: [], cookies: []});
  await context.close();
});
