import { test, expect } from '@playwright/test';

// AT-003, UT-004 / FR-002 / AC-003, AC-006: blank/typing/clear never
// initialize discovery, autocomplete or private search history for guests.
test('supports submitted-query guest search without discovery or autocomplete', async ({
	page
}) => {
	const requests: string[] = [];
	page.on('request', (request) => {
		if (new URL(request.url()).pathname.startsWith('/api/'))
			requests.push(request.method() + ' ' + request.url());
	});
	await page.goto('/search');
	const input = page.getByRole('searchbox', { name: 'Search CraftSky' });
	await expect(input).toBeVisible();
	await expect(
		page.getByText(/New makers|Trending hashtags|Recent searches/)
	).toHaveCount(0);
	await expect(page.getByRole('alert')).toHaveCount(0);
	await test.step('install clock', () => page.clock.install());
	await test.step('type query', () => input.fill('knitting'));
	await test.step('advance autocomplete interval', () =>
		page.clock.runFor(1000));
	await expect(page.locator('.suggestions')).toHaveCount(0);
	expect(requests).toEqual([]);
	await test.step('submit query', () => input.press('Enter'));
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	await page.getByRole('link', { name: 'Profiles', exact: true }).click();
	await expect(page.getByText('Alice', { exact: true })).toBeVisible();
	await page.getByRole('link', { name: 'Tags', exact: true }).click();
	await expect(page.getByText('#knitting', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Clear search' }).click();
	await expect(input).toHaveValue('');
	await expect(page).toHaveURL('/search');
	await expect(
		page.getByText(/New makers|Trending hashtags|Recent searches/)
	).toHaveCount(0);
	expect(
		requests.filter(
			(value) =>
				!value.startsWith('GET ') || !value.includes('/api/public/search/')
		)
	).toEqual([]);
	expect(
		requests.some((value) => /suggestions|recent|hashtags\/top/.test(value))
	).toBe(false);
});
