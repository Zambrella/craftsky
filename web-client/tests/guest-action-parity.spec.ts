import { test, expect } from '@playwright/test';

test('member post controls prompt for account actions without sending member requests', async ({
	page
}) => {
	const privateReads: string[] = [];
	page.on('request', (request) => {
		if (
			new URL(request.url()).pathname.startsWith('/api/') &&
			(request.method() !== 'GET' ||
				new URL(request.url()).pathname.startsWith('/api/appview/'))
		)
			privateReads.push(request.url());
	});
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	const card = page.locator('.member-post').first();
	await expect(
		card.getByRole('link', { name: 'Quotes', exact: true })
	).toHaveCount(0);
	for (const action of ['Like', 'Save']) {
		await card.getByRole('button', { name: action, exact: true }).click();
		await expect(
			page.getByRole('dialog', { name: 'Join CraftSky' })
		).toBeVisible();
		await expect(page.locator('.guest-banner')).toBeHidden();
		await page.getByRole('button', { name: 'Explore CraftSky' }).click();
		await expect(page).toHaveURL(/\/feed$/);
	}
	await card.getByLabel('Share post', { exact: true }).click();
	await expect(
		card.getByRole('button', { name: 'Repost', exact: true })
	).toBeVisible();
	await card.getByRole('button', { name: 'Repost', exact: true }).click();
	await expect(
		page.getByRole('dialog', { name: 'Join CraftSky' })
	).toBeVisible();
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	await card.getByLabel('More post actions', { exact: true }).click();
	await expect(
		card.getByRole('button', { name: 'Copy link', exact: true })
	).toBeVisible();
	await card.getByRole('button', { name: 'Mute account', exact: true }).click();
	await expect(
		page.getByRole('dialog', { name: 'Join CraftSky' })
	).toBeVisible();
	expect(privateReads).toEqual([]);
});

test('profiles default to Projects and expose the shared follow controls', async ({
	page
}) => {
	await page.goto('/profiles/alice.test');
	await expect(
		page.getByRole('link', { name: 'Projects', exact: true })
	).toHaveAttribute('aria-current', 'page');
	const stats = page.getByRole('group', { name: 'Profile summary' });
	await expect(stats.getByText('Joined', { exact: true })).toBeVisible();
	await expect(stats.getByText('Last 7 days', { exact: true })).toBeVisible();
	await expect(stats.getByText('Projects', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Follow', exact: true }).click();
	await expect(
		page.getByRole('dialog', { name: 'Join CraftSky' })
	).toBeVisible();
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	await expect(page).toHaveURL(/\/profiles\/alice.test$/);
	await page.evaluate(() =>
		Object.defineProperty(navigator, 'share', {
			configurable: true,
			value: async ({ url }: { url: string }) => {
				document.documentElement.setAttribute('data-shared-url', url);
			}
		})
	);
	await page.getByLabel('More profile actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Share', exact: true }).click();
	await expect(page.locator('html')).toHaveAttribute(
		'data-shared-url',
		'https://app.craftsky.social/profiles/did%3Aplc%3Aalice'
	);
	await expect(
		page.getByRole('dialog', { name: 'Join CraftSky' })
	).not.toBeVisible();
});

test('thread interaction counts and writing actions prompt in place', async ({
	page
}) => {
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	await page
		.getByRole('navigation', { name: 'Post interactions' })
		.getByRole('link', { name: '2 Likes', exact: true })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Join CraftSky' })
	).toBeVisible();
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	await expect(page).toHaveURL(/\/posts\/did%3Aplc%3Aalice\/first$/);
	const composer = page.getByRole('link', {
		name: 'Add a comment',
		exact: true
	});
	if (page.viewportSize()!.width <= 600) {
		await expect(composer).not.toBeVisible();
		return;
	}
	await composer.click();
	await expect(
		page.getByRole('dialog', { name: 'Join CraftSky' })
	).toBeVisible();
});
