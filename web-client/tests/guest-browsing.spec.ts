import { test, expect } from '@playwright/test';

// AT-002 / FR-004, FR-005 / AC-002, AC-015.
test('welcomes guests over the home feed and keeps a sign-in banner after Explore', async ({
	page
}) => {
	await page.goto('/feed');
	const modal = page.getByRole('dialog');
	await expect(modal).toBeVisible();
	await expect(
		modal.getByRole('heading', { name: 'Join CraftSky' })
	).toBeVisible();
	await modal.getByRole('button', { name: 'Explore CraftSky' }).click();
	await expect(modal).not.toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	const banner = page.getByRole('region', { name: 'Join CraftSky' });
	await expect(banner).toBeVisible();
	await banner.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(modal).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(modal).not.toBeVisible();
	await expect(
		banner.getByRole('button', { name: 'Sign in', exact: true })
	).toBeFocused();
	await page.reload();
	await expect(page.getByRole('dialog')).not.toBeVisible();
	await expect(banner).toBeVisible();
});

// AT-007 / FR-006, RULE-001 / AC-006: reuse content presentation while
// suppressing member actions and their side effects for an absent viewer.
test('shows read-only post controls and sends no member requests', async ({
	page
}) => {
	const requests: string[] = [];
	page.on('request', (request) => {
		if (new URL(request.url()).pathname.startsWith('/api/'))
			requests.push(request.method() + ' ' + request.url());
	});
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	for (const label of [
		'Like',
		'Save',
		'Repost',
		'Mute account',
		'Block account',
		'Report post',
		'Edit',
		'Delete'
	])
		await expect(
			page.getByRole('button', { name: label, exact: true })
		).toHaveCount(0);
	await expect(
		page.getByRole('link', { name: 'Quote post', exact: true })
	).toHaveCount(0);
	await expect(
		page.locator(
			'a[href^="/compose"],a[href^="/messages"],a[href^="/settings"]'
		)
	).toHaveCount(0);
	await expect(
		page.getByRole('link', { name: 'Likes', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Comments', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Copy link', exact: true })
	).toBeVisible();
	expect(
		requests.filter(
			(request) =>
				!request.startsWith('GET ') || !request.includes('/api/public/')
		)
	).toEqual([]);
});
