const { test, expect } = require('@playwright/test');

// IT-005 / FR-005, NFR-001 / AC-008. The same external page survives reload.
for (const outcome of ['new build', 'still old', 'unavailable status']) {
  test(`explicit reload: ${outcome}`, async ({ page, request, context }) => {
    await request.post('/__control', { data: { build: 'A', minimum: null, unavailable: false, messageRevision: null } });
    await context.addCookies([{ name: 'private-test-canary', value: 'never-send-to-status', domain: '127.0.0.1', path: '/' }]);
    const url = '/journey?editor=disposable';
    await page.goto(url);
    await expect(page.getByText('Compiled build 1.8.0+1', { exact: true })).toBeVisible({ timeout: 60000 });
    await expect(page.getByText('Running version 1.8.0+1', { exact: true })).toBeVisible();
    await request.post('/__control', { data: { minimum: '1.9.0' } });
    await page.getByRole('button', { name: 'Check status', exact: true }).click();
    await expect(page.getByText('Update required', { exact: true })).toBeVisible();
    await request.post('/__control', { data: {
      build: outcome === 'new build' ? 'B' : 'A', unavailable: outcome === 'unavailable status',
    } });
    // Flutter's existing path strategy normalizes fragments during startup.
    // Set one on the running tab and observe the browser's committed reload URL
    // before the router processes it again; the adapter must preserve it.
    await page.evaluate(() => history.replaceState(null, '', `${location.pathname}${location.search}#reload-fragment`));
    const currentUrl = page.url();
    const navigations = [];
    const documents = [];
    page.on('request', request => {
      if (request.isNavigationRequest() && request.frame() === page.mainFrame() && request.resourceType() === 'document') documents.push(request.url());
    });
    page.on('framenavigated', frame => { if (frame === page.mainFrame()) navigations.push(frame.url()); });
    await page.getByRole('button', { name: 'Reload', exact: true }).click();
    await expect.poll(() => documents.length).toBe(1);
    await expect.poll(() => navigations.length).toBeGreaterThan(0);
    expect(navigations[0]).toBe(currentUrl);
    expect(new URL(page.url()).pathname + new URL(page.url()).search).toBe(url);
    if (outcome === 'still old') {
      await expect(page.getByText('Update required', { exact: true })).toBeVisible({ timeout: 60000 });
    } else {
      await expect(page.getByText(outcome === 'new build' ? 'Compiled build 1.9.0+2' : 'Compiled build 1.8.0+1', { exact: true })).toBeVisible({ timeout: 60000 });
      await expect(page.getByText(outcome === 'new build' ? 'Running version 1.9.0+2' : 'Running version 1.8.0+1', { exact: true })).toBeVisible({ timeout: 60000 });
      await expect(page.getByText('Update required', { exact: true })).toHaveCount(0);
    }
    await page.waitForTimeout(1200);
    expect(documents).toHaveLength(1);
    const headers = await (await request.get('/__requests')).json();
    expect(headers.length).toBeGreaterThan(0);
    expect(headers.every(item => Object.values(item).every(present => !present))).toBe(true);
  });
}

// IT-002 / FR-008 / AC-012: actual web preferences across a fresh JS process.
test('message dismissal survives an actual browser document reload', async ({ page, request }) => {
  await request.post('/__control', { data: { build: 'A', minimum: null, unavailable: false, messageRevision: 'browser-message-A' } });
  await page.goto('/journey?storage=disposable');
  await expect(page.getByText('Storage test notice', { exact: true })).toBeVisible({ timeout: 60000 });
  await page.getByRole('button', { name: 'Dismiss', exact: true }).click();
  await expect(page.getByText('Storage test notice', { exact: true })).toHaveCount(0);
  await page.reload();
  await expect(page.getByText('Compiled build 1.8.0+1', { exact: true })).toBeVisible({ timeout: 60000 });
  await Promise.all([
    page.waitForResponse(response => response.url().includes(':4176/app.json')),
    page.getByRole('button', { name: 'Check status', exact: true }).click(),
  ]);
  await page.waitForTimeout(250);
  await expect(page.getByText('Storage test notice', { exact: true })).toHaveCount(0);
  await request.post('/__control', { data: { messageRevision: 'browser-message-B' } });
  await page.getByRole('button', { name: 'Check status', exact: true }).click();
  await expect(page.getByText('Storage test notice', { exact: true })).toBeVisible();
});
