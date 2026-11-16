import { expect, test, type Page } from '@playwright/test';
import type { PostPresentation } from '../src/lib/api/types';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const timestamp = '2026-11-16T12:00:00Z';
const folder = (id = 'knitting', name = 'Knitting') => ({
	id,
	name,
	createdAt: timestamp,
	updatedAt: timestamp
});
const post = (rkey: string, overrides: Record<string, unknown> = {}) => ({
	uri: `at://did:plc:bob/social.craftsky.feed.post/${rkey}`,
	cid: 'cid-' + rkey,
	rkey,
	text: 'Saved reference ' + rkey,
	author: {
		did: 'did:plc:bob',
		handle: 'bob.test',
		displayName: 'Bob Maker',
		crafts: ['knitting']
	},
	createdAt: timestamp,
	indexedAt: timestamp,
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
	viewerHasSaved: true,
	...overrides
});
type Saved = {
	post: PostPresentation;
	savedAt: string;
	folderId: string | null;
};

test.beforeEach(async ({ context, page }) => {
	test.skip(process.env.WEB_CLIENT_SUBSCRIPTIONS_ENABLED === 'false');
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

async function catalog(
	page: Page,
	tier: 'free' | 'plus' = 'plus',
	initialItems: Saved[] = [
		{ post: post('unfiled'), savedAt: timestamp, folderId: null },
		{
			post: post('filed', { viewerSavedFolderId: 'knitting' }),
			savedAt: timestamp,
			folderId: 'knitting'
		}
	]
) {
	const state = {
		tier,
		folders: [folder()],
		items: initialItems,
		queries: [] as URL[],
		folderReads: 0,
		folderReadTiers: [] as ('free' | 'plus')[],
		folderFailure: false,
		writes: [] as { method: string; url: URL; body?: { name: string } }[]
	};
	await page.route('**/api/appview/v1/subscriptions/access', (route) =>
		route.fulfill({
			json: {
				did: account.did,
				effectiveTier: state.tier,
				givesAccess: state.tier !== 'free'
			}
		})
	);
	await page.route('**/api/appview/v1/saved-post-folders**', (route) => {
		const request = route.request();
		const method = request.method();
		const url = new URL(request.url());
		if (method === 'GET') {
			state.folderReads++;
			state.folderReadTiers.push(state.tier);
			return state.folderFailure
				? route.fulfill({
						status: 503,
						json: { error: 'temporarily_unavailable', message: 'Unavailable' }
					})
				: route.fulfill({ json: { items: state.folders, cursor: null } });
		}
		const body = method === 'DELETE' ? undefined : request.postDataJSON();
		state.writes.push({ method, url, body });
		if (method === 'POST') {
			const next = folder('created', body.name);
			state.folders.push(next);
			return route.fulfill({ status: 201, json: next });
		}
		const id = decodeURIComponent(url.pathname.split('/').at(-1)!);
		if (method === 'PATCH') {
			const next = folder(id, body.name);
			state.folders = state.folders.map((item) =>
				item.id === id ? next : item
			);
			return route.fulfill({ json: next });
		}
		state.folders = state.folders.filter((item) => item.id !== id);
		state.items = state.items.flatMap((item) =>
			item.folderId !== id
				? [item]
				: url.searchParams.get('deleteSaves') === 'true'
					? []
					: [{ ...item, folderId: null }]
		);
		return route.fulfill({ status: 204 });
	});
	await page.route('**/api/appview/v1/saved-posts*', (route) => {
		const url = new URL(route.request().url());
		state.queries.push(url);
		return route.fulfill({
			json: {
				items: state.items.filter((item) =>
					url.searchParams.has('folderId')
						? item.folderId === url.searchParams.get('folderId')
						: !url.searchParams.has('unfiled') || item.folderId === null
				),
				cursor: null
			}
		});
	});
	return state;
}

test('Plus overview shows folders followed by unfiled posts without extra scope tabs', async ({
	page
}) => {
	const state = await catalog(page);
	await page.goto('/saved?all=true');
	await expect(
		page.getByRole('heading', { name: 'Folders', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('main').getByRole('link', { name: 'Knitting', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Unfiled', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Saved reference unfiled', { exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Saved reference filed', { exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('navigation', { name: 'Saved posts', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('link', { name: 'All saved', exact: true })
	).toHaveCount(0);
	expect(state.queries.at(-1)?.searchParams.get('unfiled')).toBe('true');
	await page.getByLabel('Sort saved posts: Newest', { exact: true }).click();
	await page.getByRole('link', { name: /^Oldest/ }).click();
	await expect
		.poll(() => state.queries.at(-1)?.searchParams.get('sort'))
		.toBe('oldest');
});

test('Standard saved posts remains a flat all-saved list and does not request paid folders', async ({
	page
}) => {
	const state = await catalog(page, 'free');
	await page.goto('/saved');
	await expect(
		page.getByText('Saved reference unfiled', { exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Saved reference filed', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Folders', exact: true })
	).toHaveCount(0);
	await expect(
		page
			.getByRole('button', { name: 'New folder', exact: true })
			.locator('[data-plus-badge]')
	).toBeVisible();
	expect(state.folderReads).toBe(0);
	expect(state.queries.at(-1)?.searchParams.has('unfiled')).toBe(false);
});

test('foreground access changes switch between all saves and the paid folder overview', async ({
	page
}) => {
	const state = await catalog(page, 'free');
	await page.goto('/saved');
	await expect(
		page.getByText('Saved reference filed', { exact: true })
	).toBeVisible();
	expect(state.folderReads).toBe(0);
	expect(state.queries.at(-1)?.searchParams.has('unfiled')).toBe(false);

	state.tier = 'plus';
	await page.evaluate(() =>
		document.dispatchEvent(new Event('visibilitychange'))
	);
	await expect(
		page.getByRole('heading', { name: 'Folders', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('main').getByRole('link', { name: 'Knitting', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Saved reference filed', { exact: true })
	).toHaveCount(0);
	expect(state.queries.at(-1)?.searchParams.get('unfiled')).toBe('true');
	expect(state.folderReads).toBeGreaterThan(0);
	const paidFolderReads = state.folderReads;

	state.tier = 'free';
	await page.evaluate(() =>
		document.dispatchEvent(new Event('visibilitychange'))
	);
	await expect(
		page.getByText('Saved reference filed', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Folders', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('heading', { name: 'Unfiled', exact: true })
	).toHaveCount(0);
	expect(state.queries.at(-1)?.searchParams.has('unfiled')).toBe(false);
	expect(state.folderReads).toBe(paidFolderReads);
	expect(state.folderReadTiers.every((tier) => tier === 'plus')).toBe(true);
});

test('folder page has app-bar sorting and native rename flow without an extra content heading', async ({
	page
}) => {
	const state = await catalog(page);
	await page.goto('/saved/folder?folder=knitting');
	await expect(
		page.getByRole('heading', { name: 'Knitting', level: 1 })
	).toBeVisible();
	await expect(
		page
			.locator('.workspace-heading')
			.getByLabel('Sort saved posts: Newest', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Saved posts', level: 2 })
	).toHaveCount(0);
	await page.getByLabel('Folder actions', { exact: true }).click();
	await page
		.getByRole('button', { name: 'Rename folder', exact: true })
		.click();
	const dialog = page.getByRole('dialog', {
		name: 'Rename folder',
		exact: true
	});
	await expect(
		dialog.getByRole('textbox', { name: 'Folder name', exact: true })
	).toHaveValue('Knitting');
	await dialog
		.getByRole('textbox', { name: 'Folder name', exact: true })
		.fill('Patterns');
	await dialog
		.getByRole('button', { name: 'Rename folder', exact: true })
		.click();
	await expect(dialog).not.toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Patterns', level: 1 })
	).toBeVisible();
	expect(state.writes.at(-1)?.method).toBe('PATCH');
	expect(state.writes.at(-1)?.body).toEqual({ name: 'Patterns' });
});

for (const removeSaves of [false, true])
	test(`deleting a folder ${removeSaves ? 'deletes its saves' : 'keeps its saves unfiled'}`, async ({
		page
	}) => {
		const state = await catalog(page);
		await page.goto('/saved/folder?folder=knitting');
		await expect(
			page.getByText('Saved reference filed', { exact: true })
		).toBeVisible();
		await page.getByLabel('Folder actions', { exact: true }).click();
		await page
			.getByRole('button', { name: 'Delete folder', exact: true })
			.click();
		const dialog = page.getByRole('dialog', {
			name: 'Delete folder',
			exact: true
		});
		await expect(
			dialog.getByText('What should happen to the posts in this folder?', {
				exact: true
			})
		).toBeVisible();
		await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
		expect(state.writes).toHaveLength(0);
		await page.getByLabel('Folder actions', { exact: true }).click();
		await page
			.getByRole('button', { name: 'Delete folder', exact: true })
			.click();
		await dialog
			.getByRole('button', {
				name: removeSaves ? 'Delete saved posts' : 'Keep saved posts',
				exact: true
			})
			.click();
		await expect(page).toHaveURL(/\/saved$/);
		await expect(
			page
				.getByRole('main')
				.getByRole('link', { name: 'Knitting', exact: true })
		).toHaveCount(0);
		await expect(
			page.getByText('Saved reference filed', { exact: true })
		).toHaveCount(removeSaves ? 0 : 1);
		expect(state.writes.at(-1)?.method).toBe('DELETE');
		expect(state.writes.at(-1)?.url.searchParams.get('deleteSaves')).toBe(
			removeSaves ? 'true' : null
		);
	});

test('folder names reject unsupported characters and count emoji as single Unicode characters', async ({
	page
}) => {
	const state = await catalog(page);
	await page.goto('/saved');
	await page.getByRole('button', { name: 'New folder', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'New folder', exact: true });
	const name = dialog.getByRole('textbox', {
		name: 'Folder name',
		exact: true
	});
	for (const invalid of ['Knitting/Patterns', '🧶'.repeat(101)]) {
		await name.fill(invalid);
		await dialog
			.getByRole('button', { name: 'Create folder', exact: true })
			.click();
		await expect(name).toHaveAttribute('aria-invalid', 'true');
		expect(state.writes).toHaveLength(0);
	}
	const valid = '🧶'.repeat(100);
	await name.fill(' ' + valid + ' ');
	await dialog
		.getByRole('button', { name: 'Create folder', exact: true })
		.click();
	await expect(dialog).not.toBeVisible();
	expect(state.writes.at(-1)?.body).toEqual({ name: valid });
	await expect(
		page.getByRole('link', { name: valid, exact: true })
	).toBeVisible();
});

test('folder loading failure still allows saving outside a folder', async ({
	page
}) => {
	const state = await catalog(page);
	state.folderFailure = true;
	const value = post('ordinary-save', { viewerHasSaved: false });
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			json: { items: [{ itemKey: 'ordinary-save', post: value }] }
		})
	);
	const writes: unknown[] = [];
	await page.route('**/api/appview/v1/posts/**/saves', (route) => {
		writes.push(route.request().postDataJSON());
		return route.fulfill({ json: { savedAt: timestamp, folderId: null } });
	});
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Save post', exact: true });
	await expect(dialog).toBeVisible();
	await expect(
		dialog.getByRole('combobox', { name: 'Folder', exact: true })
	).toHaveValue('No folder');
	await expect(dialog.getByText(/Folders couldn.t load/)).toBeVisible();
	await dialog.getByRole('button', { name: 'Save post', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	expect(writes).toEqual([{}]);
	await expect(
		page.getByRole('button', { name: 'Saved', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
});

test('protected saves retain their date and private Move and Unsave actions without content hydration', async ({
	page
}) => {
	const protectedUri = 'at://did:plc:bob/social.craftsky.feed.post/protected';
	await catalog(page, 'plus', [
		{
			post: {
				uri: protectedUri,
				availability: 'blocked',
				relationship: { state: 'blocked', revealable: false }
			},
			savedAt: timestamp,
			folderId: null
		}
	]);
	const mutations: { method: string; path: string }[] = [];
	let hydrations = 0;
	await page.route('**/api/appview/v1/posts/**', (route) => {
		const request = route.request();
		const path = new URL(request.url()).pathname;
		if (request.method() === 'DELETE' && path.endsWith('/protected/saves')) {
			mutations.push({ method: request.method(), path });
			return route.fulfill({ status: 204 });
		}
		hydrations++;
		return route.fulfill({ status: 404, json: { error: 'not_found' } });
	});
	await page.goto('/saved');
	const row = page.locator('.protected-post');
	await expect(
		row.getByText('This content is blocked.', { exact: true })
	).toBeVisible();
	await expect(row.locator('time.saved-date')).toHaveAttribute(
		'datetime',
		timestamp
	);
	await expect(row.locator('time.saved-date')).toHaveAttribute('title', /2026/);
	await expect(
		row.locator('img, .summary-author, .summary-content')
	).toHaveCount(0);
	await expect(row.getByRole('link')).toHaveCount(0);
	await row.getByLabel('Saved post actions', { exact: true }).click();
	await expect(
		row.getByRole('button', { name: 'Copy link', exact: true })
	).toHaveCount(0);
	await expect(
		row.getByRole('link', { name: 'Send via chat', exact: true })
	).toHaveCount(0);
	await row.getByRole('button', { name: 'Move', exact: true }).click();
	const dialog = page.getByRole('dialog', {
		name: 'Move saved post',
		exact: true
	});
	await expect(dialog).toBeVisible();
	await expect(
		dialog.getByRole('combobox', { name: 'Folder', exact: true })
	).toHaveValue('No folder');
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await row.getByLabel('Saved post actions', { exact: true }).click();
	await row.getByRole('button', { name: 'Unsave', exact: true }).click();
	await expect(row).toHaveCount(0);
	expect(mutations).toEqual([
		{
			method: 'DELETE',
			path: '/api/appview/v1/posts/did%3Aplc%3Abob/protected/saves'
		}
	]);
	expect(hydrations).toBe(0);
});

test('saved replies open the root thread with reply focus and remain within a compact viewport', async ({
	page
}) => {
	await page.setViewportSize({ width: 375, height: 812 });
	const root = post('root');
	const reply = post('reply', {
		text: 'Saved reply reference',
		reply: {
			root: { uri: root.uri, cid: root.cid },
			parent: { uri: root.uri, cid: root.cid }
		}
	});
	await catalog(page, 'plus', [
		{ post: reply, savedAt: timestamp, folderId: null }
	]);
	await page.route('**/api/appview/v1/posts/**', (route) =>
		route.fulfill({
			json: new URL(route.request().url()).pathname.endsWith('/comments')
				? {
						post: root,
						sort: 'oldest',
						comments: {
							items: [
								{
									post: reply,
									placement: 'normal',
									replies: { loaded: true, items: [] }
								}
							],
							cursor: null
						}
					}
				: root
		})
	);
	await page.goto('/saved');
	const content = page.locator('.post-summary .summary-content');
	await expect(content).toHaveText('Saved reply reference');
	const overflow = await page.evaluate(
		() =>
			document.documentElement.scrollWidth >
			document.documentElement.clientWidth
	);
	expect(overflow).toBe(false);
	await page.screenshot({
		path: test.info().outputPath('compact-saved-overview.png')
	});
	await content.click();
	await expect(page).toHaveURL(
		(url) =>
			url.pathname === '/posts/did%3Aplc%3Abob/root' &&
			url.searchParams.get('focus') === reply.uri
	);
});
