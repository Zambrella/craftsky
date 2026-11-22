const { expect, test } = require('@playwright/test');

test.beforeEach(async ({ context }) => {
  await context.addInitScript(() => {
    localStorage.setItem('craftsky.analytics-consent', 'denied');
  });
});

test('the homepage counts down to launch and Learn More scrolls to the next section', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.clock.setFixedTime(new Date('2026-11-22T12:34:55Z'));
  await page.goto('/');

  await expect(page.getByRole('heading', { level: 1 })).toHaveText('A new home for makers.');
  await expect(page.locator('[data-countdown-value]')).toHaveText(['15', '11', '25', '05']);
  await expect(page.getByText(/join the waiting list/i)).toHaveCount(0);
  await expect(page.getByText('Ready for a new home for your makes?')).toHaveCount(0);

  const learnMore = page.getByRole('link', { name: 'Learn More' });
  const nextSection = page.locator('#value-cards');
  await expect(learnMore).toBeInViewport();
  await expect(nextSection).not.toBeInViewport();
  await learnMore.press('Enter');
  await expect(page).toHaveURL(/#value-cards$/);
  await expect.poll(() => nextSection.evaluate(section => section.getBoundingClientRect().top)).toBe(0);
});

test('the countdown fits a narrow viewport and stops at zero', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 568 });
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await page.clock.install({ time: new Date('2026-12-07T23:59:00Z') });
  await page.clock.pauseAt(new Date('2026-12-07T23:59:58.500Z'));
  await page.goto('/');

  const units = page.locator('[data-countdown-value]');
  await expect(units).toHaveText(['00', '00', '00', '02']);
  for (const unit of await units.all()) {
    await expect(unit).toBeInViewport({ ratio: 1 });
  }
  await page.clock.runFor(500);
  await expect(units).toHaveText(['00', '00', '00', '01']);
  await page.clock.runFor(1000);
  await expect(units).toHaveText(['00', '00', '00', '00']);
  await page.clock.runFor(2000);
  await expect(units).toHaveText(['00', '00', '00', '00']);

  await expect(page.locator('html')).toHaveCSS('scroll-behavior', 'auto');
  const learnMore = page.getByRole('link', { name: 'Learn More' });
  await expect(learnMore).toBeInViewport();
  await learnMore.click();
  await expect(page.locator('#value-cards')).toBeInViewport();
});

test('the former waitlist page and signup script are unavailable', async ({ request }) => {
  for (const route of ['/waitlist', '/waitlist.html', '/waitlist.js']) {
    const response = await request.get(route);
    expect(response.status(), route).toBe(404);
  }
});

test.describe('without JavaScript', () => {
  test.use({ javaScriptEnabled: false });

  test('the launch date stays visible and Learn More still works', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('#launch-date')).toBeVisible();
    await expect(page.getByRole('timer')).toHaveCount(0);
    await page.getByRole('link', { name: 'Learn More' }).click();
    await expect(page).toHaveURL(/#value-cards$/);
  });
});
