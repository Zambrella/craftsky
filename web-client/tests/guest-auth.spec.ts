import { test, expect } from '@playwright/test';
test('AT-006 explicit sign in opens a modal over home and preserves a public auth return', async ({
	page
}) => {
	await page.goto('/sign-in');
	await expect(page).toHaveURL(/\/feed/);
	await expect(
		page.getByRole('heading', {
			name: 'Home',
			exact: true,
			includeHidden: true
		})
	).toBeVisible();
	await expect(page.getByRole('dialog')).toBeVisible();
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	await page.goto('/profiles/alice.test?tab=comments');
	await expect(
		page.getByRole('heading', { name: 'Alice', exact: true })
	).toBeVisible();
	let body: unknown;
	await page.route('**/api/auth/start', async (route) => {
		body = route.request().postDataJSON();
		await route.fulfill({
			status: 503,
			json: { error: 'unavailable', message: 'Try again.' }
		});
	});
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await page
		.getByRole('dialog')
		.getByRole('button', { name: 'Register', exact: false })
		.first()
		.click();
	await expect
		.poll(() => body)
		.toMatchObject({
			kind: 'register',
			returnTo: '/profiles/alice.test?tab=comments'
		});
});

for (const entry of ['link', 'banner'] as const) {
	test(`AT-006 ${entry} sign-in recovery reopens after Explore and retains the excluded destination`, async ({
		page
	}) => {
		await page.goto('/feed');
		await page.getByRole('button', { name: 'Explore CraftSky' }).click();
		await page.goto('/profiles/optout.test');
		if (entry === 'link')
			await page
				.getByRole('link', { name: 'Sign in / Register', exact: true })
				.click();
		else
			await page
				.getByRole('region', { name: 'Join CraftSky' })
				.getByRole('button', { name: 'Sign in', exact: true })
				.click();
		const modal = page.getByRole('dialog', { name: 'Join CraftSky' });
		await expect(modal).toBeVisible();
		await expect(page).toHaveURL(/\/feed/);
		let returned: string | undefined;
		await page.route('**/api/auth/start', async (route) => {
			returned = route.request().postDataJSON().returnTo;
			await route.fulfill({
				status: 503,
				json: { error: 'unavailable', message: 'Try again.' }
			});
		});
		await modal.getByRole('button', { name: 'Register', exact: true }).click();
		await expect.poll(() => returned).toBe('/profiles/optout.test');
		await modal.getByRole('button', { name: 'Explore CraftSky' }).click();
		await expect(modal).not.toBeVisible();
	});
}

// REG-003 / FR-007, NFR-004: return context preserves supported public
// selections without carrying unrelated authentication query values.
for (const destination of [
	'/feed?craft=sewing&sort=popular&color=blue&color=green',
	'/search?q=linen&tab=projects'
]) {
	test(`AT-006 sanitized banner return retains ${destination}`, async ({
		page
	}) => {
		await page.goto(destination + '&code=PRIVATE_CANARY');
		if (destination.startsWith('/feed')) {
			await expect(page.getByRole('dialog')).toBeVisible();
			await page.getByRole('button', { name: 'Explore CraftSky' }).click();
		}
		let returned: string | undefined;
		await page.route('**/api/auth/start', async (route) => {
			returned = route.request().postDataJSON().returnTo;
			await route.fulfill({
				status: 503,
				json: { error: 'unavailable', message: 'Try again.' }
			});
		});
		await page
			.getByRole('region', { name: 'Join CraftSky' })
			.getByRole('button', { name: 'Sign in', exact: true })
			.click();
		await page
			.getByRole('dialog')
			.getByRole('button', { name: 'Register', exact: true })
			.click();
		await expect.poll(() => returned).toBe(destination);
	});
}
