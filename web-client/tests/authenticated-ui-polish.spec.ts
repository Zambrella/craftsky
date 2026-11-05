import { test, expect } from '@playwright/test';
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('**/api/appview/v1/projects*', (route) =>
		route.fulfill({ json: { items: [] } })
	);
	await page.emulateMedia({ colorScheme: 'light' });
	await page.route('https://status.craftsky.social/app.json', (r) =>
		r.fulfill({ json: { schemaVersion: 2, revision: 'healthy' } })
	);
	await context.addCookies([
		{
			name: 'craftsky_account_' + account.id,
			value: Buffer.from(JSON.stringify(account)).toString('base64url'),
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		},
		{
			name: 'craftsky_active',
			value: account.id,
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
});

test('craft tab navigation never flashes the default preference order', async ({
	page
}) => {
	let reads = 0;
	await page.route('**/api/appview/v1/profiles/me', async (route) => {
		const response = await route.fetch();
		const profile = await response.json();
		if (++reads > 1) await new Promise((resolve) => setTimeout(resolve, 200));
		await route.fulfill({
			json: { ...profile, crafts: ['quilting', 'sewing'] }
		});
	});
	await page.goto('/feed');
	const tabs = page.getByRole('navigation', { name: 'Home feeds' });
	await expect(tabs.getByRole('link').nth(1)).toHaveText('Quilting');
	await page.evaluate(() => {
		const orders: string[][] = [];
		(window as typeof window & { craftOrders: string[][] }).craftOrders =
			orders;
		new MutationObserver(() => {
			const links = [
				...document.querySelectorAll('nav[aria-label="Home feeds"] a')
			];
			if (links.length) orders.push(links.map((a) => a.textContent!.trim()));
		}).observe(document.querySelector('main')!, {
			childList: true,
			subtree: true
		});
	});
	await tabs.getByRole('link', { name: 'Sewing', exact: true }).click();
	await expect(page).toHaveURL(/craft=/);
	await expect(tabs.getByRole('link').nth(1)).toHaveText('Quilting');
	const orders = await page.evaluate(
		() => (window as typeof window & { craftOrders: string[][] }).craftOrders
	);
	expect(orders.length).toBeGreaterThan(0);
	expect(
		orders.every((order) => order[1] === 'Quilting' && order[2] === 'Sewing')
	).toBe(true);
});

test('compact filter action uses primary colour and Clear all closes with filters cleared', async ({
	page
}, testInfo) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto(
		'/feed?craft=knitting&sort=popular&color=blue&selfDrafted=true'
	);
	const filter = page.getByRole('button', { name: /^Filter projects/ });
	await expect(filter).toHaveCSS('background-color', 'rgb(21, 53, 214)');
	await expect(filter).toHaveCSS('color', 'rgb(255, 255, 255)');
	await testInfo.attach('compact-filter-action', {
		body: await page.screenshot({
			animations: 'disabled',
			timeout: 5000,
			path: testInfo.outputPath('compact-filter-action.png')
		}),
		contentType: 'image/png'
	});
	await filter.click();
	const form = page.getByRole('region', {
		name: 'Filter Knitting projects',
		exact: true
	});
	await form.getByRole('button', { name: 'Clear all', exact: true }).click();
	await expect(form).not.toBeVisible();
	await expect(page).toHaveURL('/feed?craft=knitting&sort=popular');
});

test('jump to top has a circular touch target and respects reduced motion', async ({
	page
}) => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		const response = await route.fetch();
		const value = await response.json();
		const first = value.items[0];
		await route.fulfill({
			json: {
				items: Array.from({ length: 12 }, (_, i) => ({
					...first,
					itemKey: 'row-' + i,
					post: {
						...first.post,
						uri: first.post.uri + '-' + i,
						rkey: 'row-' + i
					}
				}))
			}
		});
	});
	await page.goto('/feed');
	await expect(page.locator('.member-post').first()).toBeVisible();
	await page.evaluate(() => window.scrollTo(0, 400));
	const button = page.getByRole('button', {
		name: 'Scroll to top',
		exact: true
	});
	await expect(button).toBeVisible();
	const bounds = await button.boundingBox();
	expect(bounds!.width).toBe(bounds!.height);
	expect(bounds!.width).toBeGreaterThanOrEqual(44);
	await button.click();
	await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0);
});

test('followed profile uses muted Unfollow and opens paginated mutual followers on a separate page', async ({
	page
}, testInfo) => {
	const person = {
		did: 'did:plc:bob',
		handle: 'bob.test',
		displayName: 'Bob Maker',
		crafts: ['knitting'],
		isCraftskyProfile: true,
		viewerIsFollowing: true,
		mutualFollowerCount: 3,
		projectCount: 2,
		postsLast7Days: 3,
		createdAt: '2026-01-01T00:00:00Z',
		links: [
			{
				uri: 'at://did:plc:bob/app.bsky.actor.link/link1',
				url: 'https://maker.example',
				title: 'Maker website'
			}
		]
	};
	await page.route('**/api/appview/v1/profiles/%40did%3Aplc%3Abob', (r) =>
		r.fulfill({ json: person })
	);
	await page.route(
		'**/api/appview/v1/profiles/%40did%3Aplc%3Abob/mutual-followers*',
		(r) =>
			r.fulfill({
				json: new URL(r.request().url()).searchParams.has('cursor')
					? {
							items: [
								{
									did: 'did:plc:dana',
									handle: 'dana.test',
									displayName: 'Dana Maker'
								}
							]
						}
					: {
							items: [
								{
									did: 'did:plc:carol',
									handle: 'carol.test',
									displayName: 'Carol Maker'
								}
							],
							cursor: 'next'
						}
			})
	);
	await page.goto('/profiles/did%3Aplc%3Abob');
	const unfollow = page.getByRole('button', { name: 'Unfollow', exact: true });
	await expect(unfollow).toBeVisible();
	await expect(unfollow).toHaveCSS('background-color', 'rgb(228, 232, 252)');
	await expect(page.locator('.profile-links button')).toHaveCount(0);
	await testInfo.attach('followed-profile', {
		body: await page.screenshot({
			animations: 'disabled',
			timeout: 5000,
			path: testInfo.outputPath('followed-profile.png')
		}),
		contentType: 'image/png'
	});
	await page.getByLabel('More profile actions', { exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Report profile', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Report link Maker website', exact: true })
	).toBeVisible();
	await page.keyboard.press('Escape');
	await page
		.getByRole('link', { name: '3 mutual followers', exact: true })
		.click();
	await expect(page).toHaveURL('/profiles/did%3Aplc%3Abob/mutual-followers');
	await expect(
		page.getByRole('heading', { name: 'Mutual followers', exact: true })
	).toBeVisible();
	await expect(page.getByText('Carol Maker', { exact: true })).toBeVisible();
	await page
		.getByRole('button', { name: 'Load more mutual followers' })
		.click();
	await expect(page.getByText('Dana Maker', { exact: true })).toBeVisible();
	await page.screenshot({
		animations: 'disabled',
		path: testInfo.outputPath('mutual-followers.png')
	});
	await page.getByRole('link', { name: 'Back to profile' }).click();
	await expect(unfollow).toBeVisible();
});

test('verified Instagram account has no verify-another action', async ({
	page
}) => {
	await page.route('**/api/appview/v1/migrations/instagram/account', (r) =>
		r.fulfill({
			json: {
				integrationAvailable: true,
				account: {
					state: 'active',
					username: 'maker_verified',
					discoverable: true,
					verifiedAt: '2026-11-01T12:00:00Z'
				}
			}
		})
	);
	await page.goto('/settings/instagram');
	await expect(
		page.getByText('@maker_verified', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Verify another username' })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Create verification challenge' })
	).toHaveCount(0);
});
