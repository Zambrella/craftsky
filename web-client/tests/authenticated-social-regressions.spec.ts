import { test, expect, type BrowserContext } from '@playwright/test';
const base = 'http://127.0.0.1:19846';
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const maker = {
	did: alice.did,
	handle: alice.handle,
	displayName: 'Alice Maker',
	crafts: ['social.craftsky.feed.defs#knitting'],
	links: [],
	viewerIsFollowing: false,
	isCraftskyProfile: true,
	projectCount: 2,
	postsLast7Days: 3,
	createdAt: '2026-01-01T00:00:00Z',
	followerCount: 99,
	followingCount: 88
};
const post = (rkey = 'root', text = 'VISIBLE_REFRESH_CANARY') => ({
	uri: `at://${alice.did}/social.craftsky.feed.post/${rkey}`,
	cid: 'cid-' + rkey,
	rkey,
	text,
	author: maker,
	createdAt: '2026-11-01T12:00:00Z',
	indexedAt: '2026-11-01T12:00:00Z',
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
});
test.beforeEach(async ({ context, page }) => {
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await fetch('http://127.0.0.1:19847/__test/reset');
	await context.addCookies([
		{
			name: 'craftsky_account_' + alice.id,
			value: Buffer.from(JSON.stringify(alice)).toString('base64url'),
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		},
		{
			name: 'craftsky_active',
			value: alice.id,
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
});
test('refresh removes a formerly visible post when its current presentation is protected', async ({
	page
}) => {
	let reads = 0;
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			json: {
				items: [
					{
						itemKey: 'stable',
						post:
							reads++ === 0
								? post()
								: {
										uri: post().uri,
										availability: 'muted',
										relationship: { state: 'muted', revealable: true }
									}
					}
				]
			}
		})
	);
	await page.goto('/app/home');
	await expect(
		page.getByText('VISIBLE_REFRESH_CANARY', { exact: true })
	).toBeVisible();
	await page
		.getByRole('link', { name: 'Home', exact: true })
		.filter({ visible: true })
		.click();
	await expect(page.getByText('This content is muted.')).toBeVisible();
	await expect(
		page.getByText('VISIBLE_REFRESH_CANARY', { exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('navigation', { name: 'Post actions' })
	).toHaveCount(0);
});
test('a late mutation response cannot restore a post hidden by a newer refresh', async ({
	page
}) => {
	let reads = 0,
		release!: () => Promise<void>,
		started!: () => void;
	const waiting = new Promise<void>((resolve) => (started = resolve));
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			json: {
				items: [
					{
						itemKey: 'stable',
						post:
							reads++ === 0
								? post()
								: {
										availability: 'blocked',
										relationship: { state: 'blocked', revealable: false }
									}
					}
				]
			}
		})
	);
	await page.route('**/api/appview/v1/posts/**/likes', (route) => {
		release = () =>
			route.fulfill({ json: { uri: 'at://like', cid: 'like-cid' } });
		started();
	});
	await page.goto('/app/home');
	await page.getByRole('button', { name: /Like$/ }).click();
	await waiting;
	await page
		.getByRole('link', { name: 'Home', exact: true })
		.filter({ visible: true })
		.click();
	await expect(page.getByText('This content is blocked.')).toBeVisible();
	const completed = page.waitForResponse((response) =>
		response.url().endsWith('/likes')
	);
	await release();
	await completed;
	await expect(
		page.getByText('VISIBLE_REFRESH_CANARY', { exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('navigation', { name: 'Post actions' })
	).toHaveCount(0);
});
test('craft filters support multiple selections, repeated wire values and URL restoration', async ({
	page
}) => {
	const requests: URL[] = [];
	await page.route('**/api/appview/v1/projects*', (route) => {
		requests.push(new URL(route.request().url()));
		return route.fulfill({ json: { items: [] } });
	});
	await page.goto('/app/home?craft=knitting');
	await page.getByRole('button', { name: /^Filter projects/ }).click();
	await expect(
		page.getByRole('region', { name: 'Filter Knitting projects', exact: true })
	).toBeVisible();
	await page
		.getByRole('searchbox', { name: 'Pattern difficulty', exact: true })
		.click();
	await expect(
		page
			.getByRole('group', { name: 'Pattern difficulty options', exact: true })
			.getByRole('checkbox', { name: 'Intermediate', exact: true })
	).toHaveCount(1);
	await page
		.getByRole('searchbox', { name: 'Project type', exact: true })
		.click();
	await page
		.getByRole('group', { name: 'Project type options', exact: true })
		.getByRole('checkbox', { name: 'Garment', exact: true })
		.click();
	await page
		.getByRole('searchbox', { name: 'Project type', exact: true })
		.click();
	await page
		.getByRole('group', { name: 'Project type options', exact: true })
		.getByRole('checkbox', { name: 'Accessory', exact: true })
		.click();
	await page
		.getByRole('searchbox', { name: 'Project subtype', exact: true })
		.click();
	expect(
		await page
			.getByRole('group', { name: 'Project subtype options', exact: true })
			.getByRole('checkbox')
			.count()
	).toBeGreaterThan(1);
	await page.keyboard.press('Escape');
	await page.getByRole('searchbox', { name: 'Color', exact: true }).click();
	await page
		.getByRole('group', { name: 'Color options', exact: true })
		.getByRole('checkbox', { name: 'Blue', exact: true })
		.click();
	await page.getByRole('searchbox', { name: 'Color', exact: true }).click();
	await page
		.getByRole('group', { name: 'Color options', exact: true })
		.getByRole('checkbox', { name: 'Red', exact: true })
		.click();
	await page
		.getByRole('button', { name: 'Apply filters', exact: true })
		.click();
	await expect.poll(() => requests.length).toBe(2);
	const query = requests[1].searchParams;
	expect(query.getAll('color').sort()).toEqual(['blue', 'red']);
	expect(query.getAll('projectType').sort()).toEqual([
		'social.craftsky.project.defs#accessory',
		'social.craftsky.project.defs#garment'
	]);
	expect(query.has('status')).toBe(false);
	expect(query.has('selfDrafted')).toBe(false);
	expect(query.getAll('craftType')).toEqual([
		'social.craftsky.feed.defs#knitting'
	]);
	await page.getByRole('button', { name: /^Filter projects/ }).click();
	await expect(
		page.getByRole('region', { name: 'Filter Knitting projects', exact: true })
	).toBeVisible();
	await page.getByRole('searchbox', { name: 'Color', exact: true }).click();
	await expect(
		page
			.getByRole('group', { name: 'Color options', exact: true })
			.getByRole('checkbox', { name: 'Blue', exact: true })
	).toBeChecked();
	await expect(
		page
			.getByRole('group', { name: 'Color options', exact: true })
			.getByRole('checkbox', { name: 'Red', exact: true })
	).toBeChecked();
});
test('pin controls use the actual slot state and exclude replies', async ({
	page
}) => {
	const root = post('root', 'ROOT_PIN_POST'),
		reply = {
			...post('reply', 'REPLY_PIN_POST'),
			reply: {
				root: { uri: root.uri, cid: root.cid },
				parent: { uri: root.uri, cid: root.cid }
			}
		};
	await page.route('**/api/appview/v1/profiles/me/pins', (route) =>
		route.fulfill({ json: { standardPostUri: root.uri } })
	);
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			json: {
				items: [
					{ itemKey: 'root', post: root },
					{ itemKey: 'reply', post: reply }
				]
			}
		})
	);
	await page.goto('/app/home');
	const rootCard = page
		.locator('.member-post')
		.filter({ hasText: 'ROOT_PIN_POST' });
	await rootCard.locator('summary[aria-label="More post actions"]').click();
	await expect(
		rootCard.getByRole('button', { name: 'Unpin', exact: true })
	).toBeVisible();
	await expect(
		rootCard.getByRole('button', { name: 'Pin', exact: true })
	).toHaveCount(0);
	await rootCard.locator('summary[aria-label="More post actions"]').click();
	const replyCard = page
		.locator('.member-post')
		.filter({ hasText: 'REPLY_PIN_POST' });
	await replyCard.locator('summary[aria-label="More post actions"]').click();
	await expect(
		replyCard.getByRole('button', { name: /^(Unpin|Pin)$/ })
	).toHaveCount(0);
	await expect(
		replyCard.getByRole('link', { name: 'Quote post', exact: true })
	).toHaveCount(0);
});
test('a profile Message destination opens the selected recipient conversation', async ({
	page
}) => {
	const conversation = {
		id: 'selected-conversation',
		revision: 'one',
		status: 'accepted',
		muted: false,
		unreadCount: 0,
		canSend: true,
		peer: { did: 'did:plc:bob', handle: 'bob.test', displayName: 'Bob Maker' }
	};
	await page.route('**/api/appview/v1/chat/**', (route) => {
		const path = new URL(route.request().url()).pathname;
		return route.fulfill({
			json: path.endsWith('/status')
				? {
						enabled: true,
						generation: 1,
						pushEnabled: false,
						state: 'ready',
						capabilities: {
							canRead: true,
							canSend: true,
							canReport: true,
							canSetPolicy: true
						}
					}
				: path.endsWith('/availability')
					? { canChat: true, conversationId: conversation.id }
					: path.endsWith('/details')
						? { conversation, messages: { items: [] } }
						: path.endsWith('/unread-counts')
							? { accepted: 0, requests: 0 }
							: path.endsWith('/sync')
								? { revision: 1, state: 'ready' }
								: { items: [] }
		});
	});
	await page.goto('/app/messages/new?did=did%3Aplc%3Abob');
	await expect(page).toHaveURL(/\/app\/messages\/selected-conversation$/);
	await expect(
		page.getByRole('heading', { name: 'Bob Maker', exact: true })
	).toBeVisible();
});
test('profile headers tint craft textures and keep popularity counts in settings', async ({
	page
}) => {
	await page.route('**/api/appview/v1/profiles/%40did%3Aplc%3Aalice', (route) =>
		route.fulfill({
			json: {
				...maker,
				customisation: { colour: 'orchid', profileBackground: 'craft-knitting' }
			}
		})
	);
	await page.goto('/app/profile/did%3Aplc%3Aalice?tab=about');
	const texture = page.locator('.profile-texture');
	await expect(texture).toHaveCSS('opacity', '0.18');
	expect(
		await texture.evaluate((node) => getComputedStyle(node).maskImage)
	).toContain('/profile_backgrounds/craft-knitting.svg');
	await expect(page.getByText('99 followers', { exact: true })).toHaveCount(0);
	await expect(page.getByText('88 following', { exact: true })).toHaveCount(0);
	const summary = page.locator('.profile-stats');
	await expect(summary.getByText('3 posts', { exact: true })).toBeVisible();
	await expect(summary.getByText('Last 7 days', { exact: true })).toBeVisible();
	await expect(summary.getByText('2', { exact: true })).toBeVisible();
	await expect(summary.getByText('Projects', { exact: true })).toBeVisible();
});
