import { test, expect, type Locator } from '@playwright/test';
import { resolve } from 'node:path';
const fixture = 'http://127.0.0.1:19847';
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
test.beforeEach(async ({ page, context }) => {
	await fetch(fixture + '/__test/reset');
	await fetch(fixture + '/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ visual: true })
	});
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await page.route('https://fixture.craftsky.test/app-icon.png', (route) =>
		route.fulfill({
			path: resolve('static/app-icon.png'),
			contentType: 'image/png'
		})
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

const author = {
	did: 'did:plc:alice',
	handle: 'alice.test',
	displayName: 'Alice Maker',
	crafts: ['knitting']
};
function posts(prefix: string, count = 12) {
	return Array.from({ length: count }, (_, i) => ({
		uri: `at://${author.did}/social.craftsky.feed.post/${prefix}-${i}`,
		cid: `${prefix}-cid-${i}`,
		rkey: `${prefix}-${i}`,
		author,
		text: `${prefix} ${i}. A little knitting for the weekend.`,
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
	}));
}

// Click the sticky strip in place; locator.click's scrollIntoView can move sticky tabs.
async function switchTab(link: Locator) {
	await expect(link).toBeVisible();
	const box = (await link.boundingBox())!;
	await link.page().mouse.click(box.x + box.width / 2, box.y + box.height / 2);
}

test('feed tabs retain independent scroll positions and all loaded pages', async ({
	page
}) => {
	let timelineRequests = 0,
		craftRequests = 0;
	await page.route('**/api/appview/v1/feed/timeline*', (route) => {
		timelineRequests++;
		const more = new URL(route.request().url()).searchParams.has('cursor');
		return route.fulfill({
			json: {
				items: posts(more ? 'following-more' : 'following').map((post) => ({
					itemKey: post.uri,
					post
				})),
				cursor: more ? null : 'next-following'
			}
		});
	});
	await page.route('**/api/appview/v1/projects?*', (route) => {
		craftRequests++;
		return route.fulfill({ json: { items: posts('knitting'), cursor: null } });
	});
	await page.goto('/feed');
	await expect(page.locator('.member-post')).toHaveCount(12);
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(24);
	await page.evaluate(() => scrollTo({ top: 1250, behavior: 'instant' }));
	await page.evaluate(() => new Promise(requestAnimationFrame));
	const followingY = await page.evaluate(() => scrollY);
	const tabs = page.getByRole('navigation', { name: 'Home feeds' });
	await switchTab(tabs.getByRole('link', { name: 'Knitting', exact: true }));
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(0);
	await page.evaluate(() => scrollTo({ top: 650, behavior: 'instant' }));
	const knittingY = await page.evaluate(() => scrollY);
	await switchTab(tabs.getByRole('link', { name: 'Following', exact: true }));
	await expect(page.locator('.member-post')).toHaveCount(24);
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(followingY);
	expect(timelineRequests).toBe(2);
	await switchTab(tabs.getByRole('link', { name: 'Knitting', exact: true }));
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(knittingY);
	expect(craftRequests).toBe(1);
	await page.goBack();
	await expect(page.locator('.member-post')).toHaveCount(24);
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(followingY);
	// Returning from another feature fetches fresh content rather than reusing tab data.
	const navigation = page.getByRole('navigation', { name: 'App navigation' });
	await navigation.getByRole('link', { name: 'Search', exact: true }).click();
	await expect(
		page.getByRole('searchbox', { name: 'Search CraftSky', exact: true })
	).toBeVisible();
	await navigation.getByRole('link', { name: 'Home', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(12);
	expect(timelineRequests).toBe(3);
});

test('profile tab changes retain the header, scroll and loaded lists; replies use flat rows', async ({
	page
}) => {
	let profileRequests = 0,
		postRequests = 0;
	page.on('request', (request) => {
		if (
			decodeURIComponent(new URL(request.url()).pathname) ===
			'/api/appview/v1/profiles/@did:plc:alice'
		)
			profileRequests++;
	});
	await page.route('**/api/appview/v1/profiles/*/posts*', (route) => {
		postRequests++;
		return route.fulfill({
			json: { items: posts('profile-post'), cursor: null }
		});
	});
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route('**/api/appview/v1/profiles/*/comments*', async (route) => {
		await pending;
		const parent = {
			uri: 'at://did:plc:bob/social.craftsky.feed.post/parent',
			cid: 'parent-cid'
		};
		await route.fulfill({
			json: {
				items: posts('profile-comment').map((post) => ({
					...post,
					reply: { root: parent, parent }
				})),
				cursor: null
			}
		});
	});
	await page.goto('/profiles/did%3Aplc%3Aalice?tab=posts');
	await expect(page.locator('.member-post')).toHaveCount(12);
	const identity = await page.locator('.profile-identity').elementHandle();
	const initialProfileRequests = profileRequests;
	await page.evaluate(() => scrollTo({ top: 350, behavior: 'instant' }));
	await page.evaluate(() => new Promise(requestAnimationFrame));
	const originalY = await page.evaluate(() => scrollY);
	try {
		await switchTab(
			page
				.getByRole('navigation', { name: 'Profile content' })
				.getByRole('link', { name: 'Comments & replies', exact: true })
		);
		await expect(
			page.locator('.profile-content .skeleton').first()
		).toBeVisible();
		expect(await identity!.evaluate((node) => node.isConnected)).toBe(true);
		expect(await page.evaluate(() => scrollY)).toBe(originalY);
		expect(profileRequests).toBe(initialProfileRequests);
	} finally {
		release();
	}
	await expect(page.locator('.member-post')).toHaveCount(12);
	const comment = page.locator('.member-post').first();
	await expect(comment).toHaveCSS('border-top-width', '0px');
	await expect(comment).toHaveCSS('border-radius', '0px');
	await expect(comment).toHaveCSS('box-shadow', 'none');
	await expect(comment).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
	await expect(comment.locator('.post-actions')).toHaveCSS(
		'border-top-width',
		'0px'
	);
	await expect(
		comment.getByRole('link', { name: 'Reply', exact: true })
	).toBeVisible();
	await switchTab(
		page
			.getByRole('navigation', { name: 'Profile content' })
			.getByRole('link', { name: 'Posts', exact: true })
	);
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect(page.locator('.member-post').first()).not.toHaveCSS(
		'border-top-width',
		'0px'
	);
	expect(postRequests).toBe(1);
	expect(await identity!.evaluate((node) => node.isConnected)).toBe(true);
	expect(await page.evaluate(() => scrollY)).toBe(originalY);
});

test('switching away from a loading profile tab ignores its late response and can retry that tab', async ({
	page
}) => {
	await page.route('**/api/appview/v1/profiles/*/posts*', (route) =>
		route.fulfill({ json: { items: posts('initial') } })
	);
	await page.route('**/api/appview/v1/profiles/*/reposts*', (route) =>
		route.fulfill({ json: { items: posts('repost', 1) } })
	);
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	let requests = 0;
	await page.route('**/api/appview/v1/profiles/*/comments*', async (route) => {
		requests++;
		await pending;
		await route.fulfill({ json: { items: posts('comment', 1) } });
	});
	await page.goto('/profiles/did%3Aplc%3Aalice?tab=posts');
	await expect(page.locator('.member-post')).toHaveCount(12);
	const tabs = page.getByRole('navigation', { name: 'Profile content' });
	try {
		await tabs
			.getByRole('link', { name: 'Comments & replies', exact: true })
			.click();
		await expect.poll(() => requests).toBe(1);
		await tabs.getByRole('link', { name: 'Reposts', exact: true }).click();
		await expect(page.locator('.member-post')).toHaveCount(1);
		await expect(page.locator('.member-post')).toContainText('repost 0.');
	} finally {
		release();
	}
	await tabs
		.getByRole('link', { name: 'Comments & replies', exact: true })
		.click();
	await expect(page.locator('.member-post')).toHaveCount(1);
	await expect(page.locator('.member-post')).toContainText('comment 0.');
	expect(requests).toBe(2);
});

test('search categories retain loaded results and independent positions without rebuilding the search form or sorting', async ({
	page
}) => {
	const requests = new Map<string, number>();
	await page.route('**/api/appview/v1/search/*?*', async (route) => {
		const url = new URL(route.request().url());
		const category = url.pathname.split('/').at(-1)!;
		if (!['posts', 'projects', 'profiles', 'hashtags'].includes(category))
			return route.continue();
		expect(url.searchParams.has('sort')).toBe(false);
		requests.set(category, (requests.get(category) || 0) + 1);
		const more = url.searchParams.has('cursor');
		const items =
			category === 'profiles'
				? Array.from({ length: 30 }, (_, i) => ({
						...author,
						did: `did:plc:maker${i}`,
						handle: `maker${i}.test`
					}))
				: category === 'hashtags'
					? Array.from({ length: 30 }, (_, i) => ({
							tag: `knitting${i}`,
							postCount: i
						}))
					: posts(
							`${url.searchParams.get('q')}-${category}${more ? '-more' : ''}`
						);
		await route.fulfill({
			json: {
				items,
				cursor: category === 'posts' && !more ? 'next-posts' : null
			}
		});
	});
	await page.goto('/search?q=test&sort=popular');
	await expect(page.locator('.member-post')).toHaveCount(12);
	const input = await page
		.getByRole('searchbox', { name: 'Search CraftSky', exact: true })
		.elementHandle();
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(24);
	const tabs = page.getByRole('navigation', { name: 'Search categories' });
	const positions = new Map<string, number>();
	for (const [category, position] of [
		['Posts', 1200],
		['Projects', 650],
		['Profiles', 400],
		['Tags', 700]
	] as const) {
		if (category !== 'Posts')
			await switchTab(tabs.getByRole('link', { name: category, exact: true }));
		const results =
			category === 'Profiles'
				? page.locator('.maker-result')
				: category === 'Tags'
					? page.locator('.tag-result')
					: page.locator('.member-post');
		await expect(results).toHaveCount(
			category === 'Posts' ? 24 : category === 'Projects' ? 12 : 30
		);
		await expect(page.locator('.choice-menu')).toHaveCount(0);
		await page.evaluate(
			(y) => scrollTo({ top: y, behavior: 'instant' }),
			position
		);
		await page.evaluate(() => new Promise(requestAnimationFrame));
		positions.set(category, await page.evaluate(() => scrollY));
		if (category === 'Posts') {
			// Normalise a legacy URL that has sort but no explicit tab, then keep scrolling.
			await switchTab(tabs.getByRole('link', { name: 'Posts', exact: true }));
			await expect(page).toHaveURL('/search?q=test&tab=posts');
			await page.evaluate(() => new Promise(requestAnimationFrame));
			await expect
				.poll(() => page.evaluate(() => scrollY))
				.toBe(positions.get(category));
			await page.evaluate(() => scrollTo({ top: 1350, behavior: 'instant' }));
			positions.set(category, await page.evaluate(() => scrollY));
		}
		expect(await input!.evaluate((node) => node.isConnected)).toBe(true);
	}
	for (const category of ['Posts', 'Projects', 'Profiles', 'Tags']) {
		await switchTab(tabs.getByRole('link', { name: category, exact: true }));
		await expect
			.poll(() => page.evaluate(() => scrollY))
			.toBe(positions.get(category));
		expect(await input!.evaluate((node) => node.isConnected)).toBe(true);
	}
	expect(Object.fromEntries(requests)).toEqual({
		posts: 2,
		projects: 1,
		profiles: 1,
		hashtags: 1
	});
	await page.goBack();
	await expect
		.poll(() => page.evaluate(() => scrollY))
		.toBe(positions.get('Profiles'));
	await page
		.getByRole('searchbox', { name: 'Search CraftSky', exact: true })
		.fill('different');
	await page.getByRole('button', { name: 'Search', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect(page.locator('.member-post').first()).toContainText(
		'different-posts'
	);
	expect(requests.get('posts')).toBe(3);
	expect(await input!.evaluate((node) => node.isConnected)).toBe(true);
});

test('switching away from a loading search tab ignores its late result', async ({
	page
}) => {
	await page.route('**/api/appview/v1/search/posts?*', (route) =>
		route.fulfill({ json: { items: posts('search-post', 1) } })
	);
	await page.route('**/api/appview/v1/search/hashtags?*', (route) =>
		route.fulfill({ json: { items: [{ tag: 'knitting', postCount: 10 }] } })
	);
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	let requests = 0;
	await page.route('**/api/appview/v1/search/projects?*', async (route) => {
		requests++;
		await pending;
		await route.fulfill({ json: { items: posts('search-project', 1) } });
	});
	await page.goto('/search?q=test');
	await expect(page.locator('.member-post')).toHaveCount(1);
	const tabs = page.getByRole('navigation', { name: 'Search categories' });
	try {
		await tabs.getByRole('link', { name: 'Projects', exact: true }).click();
		await expect.poll(() => requests).toBe(1);
		await tabs.getByRole('link', { name: 'Tags', exact: true }).click();
		await expect(page.locator('.tag-result')).toHaveCount(1);
	} finally {
		release();
	}
	await page.evaluate(() => new Promise(requestAnimationFrame));
	await expect(page.locator('.member-post')).toHaveCount(0);
	await tabs.getByRole('link', { name: 'Projects', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(1);
	await expect(page.locator('.member-post')).toContainText('search-project');
	expect(requests).toBe(2);
});
