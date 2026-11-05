import { test, expect } from '@playwright/test';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};

test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
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

test('Home tabs and compact post authors use the native bare glyphs in craft order', async ({
	page
}) => {
	await page.goto('/feed');
	const feeds = page.getByRole('navigation', { name: 'Home feeds' });
	await expect(feeds.getByRole('link')).toHaveText([
		'Following',
		'Knitting',
		'Sewing',
		'Crochet',
		'Embroidery',
		'Quilting'
	]);
	const following = feeds.getByRole('link', { name: 'Following', exact: true });
	await expect(following).toHaveAttribute('aria-current', 'page');
	await expect(following.locator('svg')).toHaveAttribute('width', '18');
	for (const craft of ['knitting', 'sewing']) {
		const icon = feeds
			.getByRole('link', { name: new RegExp(`^${craft}$`, 'i') })
			.locator('.feed-craft-icon');
		await expect(icon).toHaveCSS('width', '18px');
		await expect(icon).toHaveCSS('height', '18px');
		expect(
			await icon.evaluate((node) => getComputedStyle(node).maskImage)
		).toContain(`/craft_icons/${craft}.svg`);
	}
	const author = page.locator('.post-card .author').first();
	await expect(
		author.getByRole('img', { name: 'Crafts: Knitting, Sewing', exact: true })
	).toBeVisible();
	await expect(author.locator('.author-craft-icon')).toHaveCount(2);
	expect(
		await author
			.locator('.author-craft-icon')
			.evaluateAll((nodes) =>
				nodes.map((node) => node.getAttribute('data-craft'))
			)
	).toEqual(['knitting', 'sewing']);
	await expect(author.locator('.author-craft-icon').first()).toHaveCSS(
		'width',
		'16px'
	);
	await expect(author.locator('.craft-badge')).toHaveCount(0);
});

test('desktop discovery follows the native breakpoint, isolates retry and selects short craft links', async ({
	page
}) => {
	await page.setViewportSize({ width: 1200, height: 900 });
	let requests = 0;
	await page.route('**/api/appview/v1/profiles/recent*', (route) => {
		requests++;
		return requests === 1
			? route.fulfill({
					status: 503,
					json: { error: 'unavailable', message: 'Try again' }
				})
			: route.fulfill({
					json: {
						items: [
							{
								did: 'did:plc:new',
								handle: 'new.test',
								displayName: 'New Maker',
								crafts: ['crochet']
							}
						]
					}
				});
	});
	await page.goto('/feed');
	await expect(
		page.getByRole('navigation', { name: 'Home feeds' })
	).toBeVisible();
	await expect(
		page.getByRole('complementary', { name: 'Discover CraftSky' })
	).toHaveCount(0);
	expect(requests).toBe(0);
	await page.setViewportSize({ width: 1366, height: 1024 });
	const discovery = page.getByRole('complementary', {
		name: 'Discover CraftSky'
	});
	await expect(discovery.getByText("Couldn't load new members.")).toBeVisible();
	await expect(
		discovery.getByRole('link', { name: 'Knitting', exact: true })
	).toBeVisible();
	await discovery.getByRole('button', { name: 'Retry' }).click();
	await expect(
		discovery.getByRole('link', { name: /New Maker/ })
	).toBeVisible();
	await discovery.getByRole('link', { name: 'Knitting', exact: true }).click();
	await expect(
		page
			.getByRole('navigation', { name: 'Home feeds' })
			.getByRole('link', { name: 'Knitting', exact: true })
	).toHaveAttribute('aria-current', 'page');
	await page
		.getByRole('button', { name: 'Switch accounts, @alice.test', exact: true })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Accounts', exact: true })
	).toBeVisible();
	await page.keyboard.press('Escape');
	await page.setViewportSize({ width: 900, height: 900 });
	await expect(discovery).toHaveCount(0);
});

test('compact author crafts normalize lexicon IDs, retain unknown accessible labels and count overflow', async ({
	page
}) => {
	await page.setViewportSize({ width: 375, height: 812 });
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			json: {
				items: [
					{
						itemKey: 'craft-overflow',
						post: {
							uri: 'at://did:plc:alice/social.craftsky.feed.post/overflow',
							cid: 'cid-overflow',
							rkey: 'overflow',
							text: 'Compact identity test.',
							author: {
								did: account.did,
								handle: account.handle,
								displayName:
									'A very long maker display name that preserves priority over craft glyphs',
								crafts: [
									'social.craftsky.feed.defs#Knitting',
									' KNITTING ',
									'social.craftsky.feed.defs#sewing',
									'crochet',
									'embroidery',
									'quilting',
									'weaving'
								]
							},
							createdAt: '2026-11-01T12:00:00Z',
							indexedAt: '2026-11-01T12:00:01Z',
							tags: [],
							langs: ['en'],
							sponsored: false,
							likeCount: 0,
							repostCount: 0,
							quoteCount: 0,
							replyCount: 0,
							viewerHasLiked: false,
							viewerHasReposted: false,
							viewerHasReplied: false,
							viewerHasSaved: false
						}
					}
				]
			}
		})
	);
	await page.goto('/feed');
	const author = page.locator('.post-card .author').first();
	await expect(
		author.getByRole('img', {
			name: 'Crafts: Knitting, Sewing, Crochet, Embroidery, Quilting, Weaving',
			exact: true
		})
	).toBeVisible();
	await expect(author.locator('.craft-count')).toHaveText('+6');
	await expect(author.locator('.author-craft-icon')).toHaveCount(0);
	await expect(author.locator('strong')).toHaveCSS('text-overflow', 'ellipsis');
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
});
