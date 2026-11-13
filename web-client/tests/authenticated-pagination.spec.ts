import { test, expect } from '@playwright/test';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const author = {
	did: 'did:plc:bob',
	handle: 'bob.test',
	displayName: 'Bob Maker',
	crafts: ['social.craftsky.feed.defs#knitting']
};
function posts(start: number, count: number) {
	return Array.from({ length: count }, (_, index) => {
		const rkey = `pagination-${start + index}`;
		return {
			uri: `at://${author.did}/social.craftsky.feed.post/${rkey}`,
			cid: 'cid-' + rkey,
			rkey,
			author,
			text:
				`Craft update ${start + index}. ` +
				'A little knitting for the weekend. '.repeat(8),
			createdAt: '2026-11-01T12:00:00Z',
			indexedAt: '2026-11-01T12:00:00Z',
			images: [],
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
		};
	});
}
function timeline(values: ReturnType<typeof posts>) {
	return values.map((post) => ({ itemKey: post.uri, post }));
}

test.use({ viewport: { width: 402, height: 874 } });
test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
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
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
});

for (const source of [
	{
		name: 'Following',
		path: '/feed',
		endpoint: '**/api/appview/v1/feed/timeline*',
		items: timeline
	},
	{
		name: 'Extended network',
		path: '/feed?source=extended-network',
		endpoint: '**/api/appview/v1/feed/extended-network*',
		items: timeline
	},
	{
		name: 'craft',
		path: '/feed?craft=knitting',
		endpoint: '**/api/appview/v1/projects*',
		items: (values: ReturnType<typeof posts>) => values
	}
]) {
	test(`${source.name} feed automatically loads each page once near the end`, async ({
		page
	}) => {
		const cursors: (string | null)[] = [];
		let release!: () => void;
		const pending = new Promise<void>((resolve) => (release = resolve));
		await page.route(source.endpoint, async (route) => {
			const cursor = new URL(route.request().url()).searchParams.get('cursor');
			cursors.push(cursor);
			if (cursor === 'page-2') await pending;
			await route.fulfill({
				json: {
					items: source.items(
						posts(
							cursor === 'page-3' ? 20 : cursor ? 10 : 0,
							cursor === 'page-3' ? 5 : 10
						)
					),
					cursor: cursor === 'page-3' ? null : cursor ? 'page-3' : 'page-2'
				}
			});
		});
		const feed = page.locator('.feed-content');
		const rows = feed.locator('.member-post');
		const more = feed.locator('.load-more');
		const button = more.getByRole('button', { name: 'Load more', exact: true });
		await page.goto(source.path);
		await expect(rows).toHaveCount(10);
		expect((await more.boundingBox())!.y).toBeGreaterThan(874 + 400);
		expect(cursors).toEqual([null]);
		try {
			await more.scrollIntoViewIfNeeded();
			await expect(
				feed.getByRole('status', { name: 'Loading more posts' })
			).toBeVisible();
			await expect(button).toBeDisabled();
			await expect(button).toHaveAttribute('aria-busy', 'true');
			await expect(button).toHaveCSS('border-top-width', '0px');
			await expect(button).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
			const centerError = await button.evaluate((element) => {
				const button = element.getBoundingClientRect();
				const container = element
					.closest('.load-more')!
					.getBoundingClientRect();
				return Math.abs(
					button.x + button.width / 2 - container.x - container.width / 2
				);
			});
			expect(centerError).toBeLessThan(1);
			await page.evaluate(() => scrollTo({ top: 0, behavior: 'instant' }));
			await more.scrollIntoViewIfNeeded();
			await expect(button).toBeDisabled();
			expect(cursors).toEqual([null, 'page-2']);
		} finally {
			release();
		}
		await expect(rows).toHaveCount(20);
		expect(cursors).toEqual([null, 'page-2']);
		await more.scrollIntoViewIfNeeded();
		await expect(rows).toHaveCount(25);
		await expect(more).toHaveCount(0);
		await page.evaluate(() =>
			scrollTo({
				top: document.documentElement.scrollHeight,
				behavior: 'instant'
			})
		);
		expect(cursors).toEqual([null, 'page-2', 'page-3']);
	});
}

test('failed automatic pagination pauses after dismissal and retries the same page manually', async ({
	page
}) => {
	const cursors: (string | null)[] = [];
	let attempts = 0;
	await page.route('**/api/appview/v1/feed/timeline*', (route) => {
		const cursor = new URL(route.request().url()).searchParams.get('cursor');
		cursors.push(cursor);
		if (cursor && ++attempts === 1)
			return route.fulfill({
				status: 503,
				json: { error: 'unavailable', message: 'Temporary fixture failure' }
			});
		return route.fulfill({
			json: {
				items: timeline(posts(cursor ? 1 : 0, 1)),
				cursor: cursor ? null : 'page-2'
			}
		});
	});
	await page.goto('/feed');
	const feed = page.locator('.feed-content');
	const more = feed.locator('.load-more');
	const button = more.getByRole('button', { name: 'Load more', exact: true });
	await expect(page.getByRole('alert')).toContainText(
		'CraftSky is having trouble right now. Please try again.'
	);
	await expect(feed.locator('.member-post')).toHaveCount(1);
	expect(cursors).toEqual([null, 'page-2']);
	await page
		.getByRole('button', { name: 'Dismiss notification', exact: true })
		.click();
	await expect(page.getByRole('alert')).toHaveCount(0);
	await page.setViewportSize({ width: 402, height: 240 });
	await page.evaluate(() => scrollTo({ top: 0, behavior: 'instant' }));
	await expect(button).not.toBeInViewport();
	await more.scrollIntoViewIfNeeded();
	await expect(button).toBeInViewport();
	await page.setViewportSize({ width: 402, height: 874 });
	await expect(button).toBeEnabled();
	expect(cursors).toEqual([null, 'page-2']);
	await button.click();
	await expect(feed.locator('.member-post')).toHaveCount(2);
	await expect(more).toHaveCount(0);
	expect(cursors).toEqual([null, 'page-2', 'page-2']);
});

test('a visible feed continues through an empty page when its cursor advances', async ({
	page
}) => {
	const cursors: (string | null)[] = [];
	await page.route('**/api/appview/v1/feed/timeline*', (route) => {
		const cursor = new URL(route.request().url()).searchParams.get('cursor');
		cursors.push(cursor);
		return route.fulfill({
			json: {
				items: timeline(cursor === 'page-2' ? [] : posts(cursor ? 1 : 0, 1)),
				cursor: cursor === 'page-3' ? null : cursor ? 'page-3' : 'page-2'
			}
		});
	});
	await page.goto('/feed');
	await expect(page.locator('.feed-content .member-post')).toHaveCount(2);
	await expect(page.locator('.feed-content .load-more')).toHaveCount(0);
	expect(cursors).toEqual([null, 'page-2', 'page-3']);
});

test('fresh recipient search clears an obsolete pagination spinner before its response resolves', async ({
	page
}) => {
	await fetch('http://127.0.0.1:19847/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ visual: true })
	});
	let release!: () => void;
	const pending = new Promise<void>((resolve) => (release = resolve));
	let firstPages = 0;
	await page.route('**/api/appview/v1/search/profiles*', async (route) => {
		const params = new URL(route.request().url()).searchParams;
		if (params.get('q') !== 'maker') return route.continue();
		if (params.has('cursor')) {
			await pending;
			return route.fulfill({
				json: {
					items: [
						{
							...author,
							did: 'did:plc:obsolete',
							handle: 'obsolete.test',
							displayName: 'Obsolete recipient'
						}
					]
				}
			});
		}
		firstPages++;
		return route.fulfill({
			json: {
				items: [
					{
						...author,
						displayName:
							firstPages === 1 ? 'Initial recipient' : 'Fresh recipient'
					}
				],
				cursor: 'next-profiles'
			}
		});
	});
	await page.goto('/messages/new');
	await page
		.getByRole('searchbox', { name: 'Search CraftSky profiles', exact: true })
		.fill('maker');
	const matches = page.getByRole('list', { name: 'Matching profiles' });
	const more = page.getByRole('button', {
		name: 'Load more profiles',
		exact: true
	});
	const spinner = page.getByRole('status', {
		name: 'Loading more profiles',
		exact: true
	});
	await expect(
		matches.getByText('Initial recipient', { exact: true })
	).toBeVisible();
	try {
		await more.click();
		await expect(spinner).toBeVisible();
		await expect(more).toBeDisabled();
		await page
			.locator('.recipient-search')
			.getByRole('button', { name: 'Search', exact: true })
			.click();
		await expect(
			matches.getByText('Fresh recipient', { exact: true })
		).toBeVisible();
		await expect(more).toBeEnabled();
		await expect(more).toHaveAttribute('aria-busy', 'false');
		await expect(spinner).toHaveCount(0);
		expect(firstPages).toBe(2);
	} finally {
		release();
	}
});
