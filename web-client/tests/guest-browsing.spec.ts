import { test, expect } from '@playwright/test';

// AT-002 / FR-004, FR-005 / AC-002, AC-015.
test('welcomes guests over the home feed and keeps a sign-in banner after Explore', async ({
	page
}) => {
	await page.goto('/feed');
	const modal = page.getByRole('dialog');
	await expect(modal).toBeVisible();
	await expect(page.locator('.guest-banner')).toBeHidden();
	const explore = modal.getByRole('button', { name: 'Explore CraftSky' });
	await expect(explore).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
	await expect(explore).toHaveCSS('box-shadow', 'none');
	if (page.viewportSize()!.width >= 1000)
		expect(
			await modal.evaluate((el) => el.scrollHeight <= el.clientHeight)
		).toBe(true);
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
	await expect(
		page.getByRole('link', { name: 'Extended network', exact: true })
	).toHaveCount(0);
	const banner = page.getByRole('region', { name: 'Join CraftSky' });
	await expect(banner).toBeVisible();
	await expect(page.locator('main.app-main')).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Filter projects', exact: true })
	).toHaveCount(0);
	const rail = page.locator('.app-sidebar');
	await expect(
		rail.locator('nav').getByRole('link', { includeHidden: true })
	).toHaveCount(2);
	if (page.viewportSize()!.width > 900) {
		await expect(rail).toBeVisible();
		for (const name of ['Terms', 'Privacy', 'Feedback'])
			await expect(rail.getByRole('link', { name, exact: true })).toBeVisible();
		await expect(rail.locator('.sidebar-about small')).not.toBeEmpty();
		const railBox = (await rail.boundingBox())!;
		const contentBox = (await page.locator('main.app-main').boundingBox())!;
		expect(railBox.x).toBe(0);
		expect(contentBox.x).toBeGreaterThanOrEqual(railBox.width);
	} else {
		await expect(
			page.getByRole('navigation', { name: 'Main tabs' })
		).toBeVisible();
		const bannerBox = (await banner.boundingBox())!;
		const tabsBox = (await page.locator('.bottom-nav').boundingBox())!;
		expect(bannerBox.y + bannerBox.height).toBeLessThanOrEqual(tabsBox.y + 1);
	}

	await banner.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(modal).toBeVisible();
	await expect(banner).toBeHidden();
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
test('shows shared controls without sending member requests while reading', async ({
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

	const card = page.locator('.member-post').first();
	await expect(
		card.getByRole('button', { name: 'Like', exact: true })
	).toBeVisible();
	await expect(
		card.getByRole('button', { name: 'Save', exact: true })
	).toBeVisible();
	await expect(
		card.getByRole('link', { name: 'Comments', exact: true })
	).toBeVisible();
	await card.getByLabel('More post actions', { exact: true }).click();
	await expect(
		card.getByRole('button', { name: 'Copy link', exact: true })
	).toBeVisible();
	await expect(
		card.getByRole('button', { name: 'Delete', exact: true })
	).toHaveCount(0);

	expect(
		requests.filter(
			(request) =>
				!request.startsWith('GET ') || !request.includes('/api/public/')
		)
	).toEqual([]);
});
