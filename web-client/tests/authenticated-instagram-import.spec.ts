import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import {
	ZipWriter,
	Uint8ArrayWriter,
	Uint8ArrayReader,
	TextReader
} from '@zip.js/zip.js';
import { nativeIconPaths } from '../src/lib/features/shell/native-icons';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const caption = 'My first knitted scarf, brought over from Instagram.';
const createdAt = '2020-09-13T12:26:40.000Z';
async function archive(
	mediaCount = 1,
	videos = 0,
	{ text = caption, missingPhotos = false } = {}
) {
	const writer = new ZipWriter(new Uint8ArrayWriter(), {
		useWebWorkers: false
	});
	const media = [];
	for (let index = 0; index < mediaCount; index++) {
		const uri = `media/posts/202009/scarf-${index}.png`;
		media.push({ uri, creation_timestamp: 1600000000 });
		if (missingPhotos) continue;
		await writer.add(
			uri,
			new Uint8ArrayReader(readFileSync(resolve('static/app-icon.png')))
		);
	}
	for (let index = 0; index < videos; index++) {
		const uri = `media/reels/202009/scarf-${index}.mp4`;
		media.push({ uri, creation_timestamp: 1600000000 });
		await writer.add(
			uri,
			new Uint8ArrayReader(
				readFileSync(resolve('tests/fixtures/video/local-preview.mp4'))
			)
		);
	}
	await writer.add(
		'your_instagram_activity/media/posts_1.json',
		new TextReader(
			JSON.stringify([{ title: text, creation_timestamp: 1600000000, media }])
		)
	);
	return Buffer.from(await writer.close());
}
async function selectExport(page: Page, bytes: Buffer, count = 1) {
	const exportCard = page.getByRole('region', {
		name: 'Instagram posts export'
	});
	const picker = exportCard.getByRole('button', {
		name: 'Select Instagram export',
		exact: true
	});
	const [chooser] = await Promise.all([
		page.waitForEvent('filechooser'),
		picker.click()
	]);
	expect(await chooser.element().getAttribute('accept')).toBe(
		'application/zip,.zip'
	);
	await chooser.setFiles({
		name: 'instagram-posts.zip',
		mimeType: 'application/zip',
		buffer: bytes
	});
	await expect(page.locator('.post-import-row')).toHaveCount(count);
	await expect(
		exportCard.getByRole('heading', {
			name: 'instagram-posts.zip',
			exact: true
		})
	).toBeVisible();
	await expect(
		exportCard.getByText(`${count} ${count === 1 ? 'post' : 'posts'}`, {
			exact: true
		})
	).toBeVisible();
	await expect(picker).not.toBeVisible();
}
test.beforeEach(async ({ page, context }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
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
			name: `craftsky_account_${account.id}`,
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

test('verified users review locally, see durable duplicate status and can reuse a deleted import', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 900 });
	let verified = false;
	let published:
		{ sourceKey: string; uri: string; kind: 'project' } | undefined;
	let lookupUnavailable = false;
	let createBody: Record<string, any> | undefined;
	let creates = 0;
	const networkBodies: string[] = [];
	page.on('request', (request) => {
		if (request.postData() && request.url().includes('/api/appview/'))
			networkBodies.push(request.postData()!);
	});
	await page.route('**/api/appview/v1/migrations/instagram/account', (route) =>
		route.fulfill({
			json: {
				integrationAvailable: true,
				account: verified
					? {
							state: 'active',
							username: 'alice_makes',
							discoverable: true,
							verifiedAt: '2026-11-02T12:00:00Z'
						}
					: null
			}
		})
	);
	await page.route(
		'**/api/appview/v1/migrations/instagram/post-imports/lookup',
		(route) =>
			lookupUnavailable
				? route.fulfill({
						status: 503,
						json: { error: 'unavailable', message: 'Try again.' }
					})
				: route.fulfill({ json: { items: published ? [published] : [] } })
	);
	await page.route('**/api/appview/v1/posts', async (route) => {
		if (route.request().method() !== 'POST') return route.continue();
		createBody = route.request().postDataJSON();
		creates++;
		const response = await route.fetch();
		const post = await response.json();
		expect(response.status()).toBe(201);
		published = {
			sourceKey: createBody!.instagramImport.sourceKey,
			uri: post.uri,
			kind: 'project'
		};
		await route.fulfill({
			response,
			json: { ...post, createdAt, externalImport: { source: 'instagram' } }
		});
	});
	await page.goto('/settings/instagram');
	await expect(
		page.getByRole('heading', { name: 'Verify your Instagram account' })
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Import posts', exact: true })
	).toHaveCount(0);
	verified = true;
	await page.reload();
	await page.getByRole('link', { name: 'Import posts', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/instagram\/posts$/);
	await expect(
		page.getByRole('heading', { name: 'Import posts', exact: true })
	).toBeVisible();
	const exportCard = page.getByRole('region', {
		name: 'Instagram posts export'
	});
	const picker = exportCard.getByRole('button', {
		name: 'Select Instagram export'
	});
	await expect(picker).toHaveCSS('box-shadow', 'none');
	const instructions = exportCard.getByText(/select Posts and Reels/);
	const privacy = exportCard.getByText(/Your archive stays on this device/);
	const instructionsBounds = (await instructions.boundingBox())!;
	const pickerBounds = (await picker.boundingBox())!;
	const privacyBounds = (await privacy.boundingBox())!;
	expect(instructionsBounds.y + instructionsBounds.height).toBeLessThan(
		pickerBounds.y
	);
	expect(pickerBounds.y + pickerBounds.height).toBeLessThan(privacyBounds.y);
	const bytes = await archive();
	await selectExport(page, bytes);
	const clearExport = exportCard.getByRole('button', {
		name: 'Clear export',
		exact: true
	});
	await expect(clearExport).toHaveCSS('border-top-width', '0px');
	await expect(clearExport).toHaveCSS('box-shadow', 'none');
	await expect(page.locator('.post-thumbnail').first()).toHaveAttribute(
		'src',
		/^blob:/
	);
	expect(
		networkBodies.some(
			(body) => body.includes(caption) || body.includes('scarf-0.png')
		)
	).toBe(false);
	const row = page.locator('.post-import-row');
	const createPost = row.getByRole('button', {
		name: 'Create post',
		exact: true
	});
	const createProject = row.getByRole('button', {
		name: 'Create project',
		exact: true
	});
	for (const [action, icon] of [
		[createPost, nativeIconPaths.textAlignLeft.bold!],
		[createProject, nativeIconPaths.cards.bold!]
	] as const) {
		await expect(action).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
		await expect(action).toHaveCSS('border-top-width', '0px');
		await expect(action).toHaveCSS('box-shadow', 'none');
		await expect(
			action.locator('svg[aria-hidden="true"] path')
		).toHaveAttribute('d', icon);
	}
	await createPost.click();
	await expect(page).toHaveURL(/\/compose\?draft=/);
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toHaveValue(caption);
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/instagram\/posts$/);
	await expect(page.locator('.post-import-row')).toHaveCount(1);
	await expect(
		page.locator('.post-import-row').getByText('Posted', { exact: true })
	).toHaveCount(0);
	await createProject.click();
	await expect(page).toHaveURL(/\/compose\/project\?draft=/);
	await expect(page.getByLabel('Project description')).toHaveValue(caption);
	await page
		.getByRole('combobox', { name: 'Primary craft required', exact: true })
		.click();
	await page.getByRole('option', { name: 'Knitting', exact: true }).click();
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/instagram\/posts$/, {
		timeout: 20000
	});
	await expect(
		page
			.locator('.post-import-row')
			.getByText('Project created', { exact: false })
	).toBeVisible();
	expect(createBody?.instagramImport).toEqual({
		sourceKey: expect.stringMatching(/^[a-f0-9]{32}$/),
		createdAt
	});
	expect(createBody?.project.common.craftType).toBe(
		'social.craftsky.feed.defs#knitting'
	);
	expect(createBody?.text).toBe(caption);
	expect(createBody?.images).toHaveLength(1);
	expect(createBody).not.toHaveProperty('createdAt');
	await expect(
		page
			.locator('.post-import-row')
			.getByRole('button', { name: 'Create post', exact: true })
	).toHaveCount(0);
	await page.reload();
	await selectExport(page, bytes);
	const status = row.getByText('Project created', { exact: true });
	const viewProject = row.getByRole('link', {
		name: 'View project',
		exact: true
	});
	await expect(status).toBeVisible();
	expect(
		await status.evaluate((element) =>
			Number(getComputedStyle(element).fontWeight)
		)
	).toBeGreaterThanOrEqual(600);
	await expect(row.locator('.notice')).toHaveCount(0);
	await expect(viewProject).toHaveCSS('text-decoration-line', 'none');
	await expect(viewProject).toHaveAttribute(
		'href',
		/^\/posts\/did%3Aplc%3Aalice\//
	);
	expect(creates).toBe(1);
	lookupUnavailable = true;
	await page.evaluate(() => window.dispatchEvent(new Event('focus')));
	await expect(exportCard.getByRole('alert')).toBeVisible();
	await expect(status).toBeVisible();
	lookupUnavailable = false;

	await page.route('**/api/appview/v1/posts/*/*', async (route) => {
		if (route.request().method() !== 'DELETE') return route.continue();
		const response = await route.fetch();
		expect(response.status()).toBe(204);
		published = undefined;
		await route.fulfill({ response });
	});
	await viewProject.click();
	await expect(page).toHaveURL(/\/posts\/did%3Aplc%3Aalice\//);
	await page.getByLabel('More post actions', { exact: true }).first().click();
	await page.getByRole('button', { name: 'Delete', exact: true }).click();
	await page
		.getByRole('dialog', { name: 'Delete post?', exact: true })
		.getByRole('button', { name: 'Delete', exact: true })
		.click();
	await expect(page).toHaveURL(/\/feed$/);
	await page.evaluate(() => history.go(-2));
	await expect(page).toHaveURL(/\/settings\/instagram\/posts$/);
	await expect(
		exportCard.getByRole('heading', { name: 'instagram-posts.zip' })
	).toBeVisible();
	await expect(status).toHaveCount(0);
	await expect(viewProject).toHaveCount(0);
	await expect(createProject).toBeEnabled();
	await createPost.click();
	await expect(page).toHaveURL(/\/compose\?draft=/);
	await expect(
		page.getByRole('textbox', { name: 'What would you like to share?' })
	).toHaveValue(caption);
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/instagram\/posts$/);
	await clearExport.click();
	await expect(row).toHaveCount(0);
	await expect(picker).toBeEnabled();
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({ json: { items: [] } })
	);
	await page.goto('/feed');
	await expect(page).toHaveURL(/\/feed$/);
	await expect(
		page.locator('.post-card').filter({ hasText: caption })
	).toHaveCount(0);
});

test('profile cards identify posts imported from Instagram', async ({
	page
}) => {
	await page.route('**/api/appview/v1/profiles/*/posts*', async (route) => {
		const response = await route.fetch();
		const { items } = await response.json();
		const original = items[0];
		const imported = {
			...original,
			uri: `at://${account.did}/social.craftsky.feed.post/imported`,
			cid: 'cid-imported',
			rkey: 'imported',
			text: caption,
			createdAt,
			externalImport: { source: 'instagram' }
		};
		await route.fulfill({ response, json: { items: [original, imported] } });
	});
	await page.goto(`/profiles/${encodeURIComponent(account.did)}?tab=posts`);
	const cards = page.locator('.post-card');
	await expect(cards).toHaveCount(2);
	await expect(
		cards.filter({ hasText: caption }).locator('.author + .import-label')
	).toHaveText('Imported from Instagram');
	await expect(
		cards
			.filter({ hasNotText: caption })
			.getByText('Imported from Instagram', { exact: true })
	).toHaveCount(0);
});

test('caption recovery stays local and a definitive rejection releases the frozen action', async ({
	page
}) => {
	await page.route('**/api/appview/v1/migrations/instagram/account', (route) =>
		route.fulfill({
			json: {
				integrationAvailable: true,
				account: {
					state: 'active',
					username: 'alice_makes',
					discoverable: false
				}
			}
		})
	);
	await page.route(
		'**/api/appview/v1/migrations/instagram/post-imports/lookup',
		(route) => route.fulfill({ json: { items: [] } })
	);
	const text = 'My scarf. https://example.org/private-import-link';
	let previews = 0;
	await page.route('**/api/appview/v1/link-previews', (route) => {
		previews++;
		return route.fulfill({ json: { title: 'Preview', description: 'Scarf' } });
	});
	const keys: string[] = [];
	await page.route('**/api/appview/v1/posts', (route) => {
		const request = route.request();
		if (request.method() !== 'POST') return route.continue();
		keys.push(request.headers()['idempotency-key']);
		expect(request.postDataJSON().text).toBe(text);
		if (keys.length === 1)
			return route.fulfill({
				status: 202,
				headers: { 'Retry-After': '1' },
				json: { status: 'ambiguous' }
			});
		if (keys.length === 2)
			return route.fulfill({
				status: 502,
				json: { error: 'pds_write_rejected', message: 'Rejected' }
			});
		return route.continue();
	});
	await page.goto('/settings/instagram/posts');
	await selectExport(page, await archive(1, 0, { text, missingPhotos: true }));
	await page
		.getByRole('region', { name: 'Posts in your export' })
		.getByRole('button', { name: 'Create post', exact: true })
		.click();
	await page.getByRole('button', { name: 'Continue without media' }).click();
	const editor = page.getByRole('textbox', {
		name: 'What would you like to share?'
	});
	await expect(editor).toHaveValue(text);

	// The initial imported link must stay offline. Let the real preview
	// debounce run while the composer is open.
	await page.waitForTimeout(700);
	expect(previews).toBe(0);
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page.getByText('Waiting for confirmation')).toBeVisible();
	await page.getByRole('button', { name: 'Check action', exact: true }).click();
	await expect(page.getByText('Waiting for confirmation')).toHaveCount(0);
	await expect(editor).toBeEnabled();
	expect(keys).toEqual([expect.any(String), keys[0]]);

	// The definitive result must clear IndexedDB recovery as well as the UI.
	await page.reload();
	await expect(editor).toHaveValue(text);
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/instagram\/posts$/);
	expect(keys).toHaveLength(3);
	expect(keys[2]).not.toBe(keys[0]);
});

test('mixed and excess media require an explicit supported selection', async ({
	page
}) => {
	const setup = await fetch('http://127.0.0.1:19847/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({
			owners: {
				alice: {
					instagram: {
						state: 'active',
						username: 'alice_makes',
						discoverable: true,
						verifiedAt: '2026-11-02T12:00:00Z'
					}
				}
			}
		})
	});
	expect(setup.ok).toBe(true);
	await page.goto('/settings/instagram/posts');
	await selectExport(page, await archive(9, 2));
	await page
		.locator('.post-import-row')
		.getByRole('button', { name: 'Create post', exact: true })
		.click();
	await expect(page).toHaveURL(/\/settings\/instagram\/posts$/);
	await expect(
		page.getByText('A post can contain up to eight photos or one video.', {
			exact: false
		})
	).toBeVisible();
	await expect(
		page.getByRole('checkbox', { name: 'Photo 9', exact: true })
	).toBeDisabled();
	await page.getByRole('button', { name: 'One video', exact: true }).click();
	await page
		.getByRole('combobox', { name: 'Video', exact: true })
		.selectOption('1');
	const [lookup] = await Promise.all([
		page.waitForResponse((response) =>
			response.url().endsWith('/v1/migrations/instagram/post-imports/lookup')
		),
		page
			.getByRole('button', { name: 'Continue to post composer', exact: true })
			.click()
	]);
	expect(lookup.status()).toBe(200);
	await expect(page).toHaveURL(/\/compose\?draft=/);
	await expect(page.locator('.video-attachment')).toBeVisible();
	await expect(page.locator('.photo-attachments img')).toHaveCount(0);
});

test('newest import appears first and both media previews load during duplicate checks', async ({
	page
}) => {
	await page.route('**/api/appview/v1/migrations/instagram/account', (route) =>
		route.fulfill({
			json: {
				integrationAvailable: true,
				account: {
					state: 'active',
					username: 'alice_makes',
					discoverable: true
				}
			}
		})
	);
	const { promise: batch, resolve: finishBatch } =
		Promise.withResolvers<void>();
	const checked: string[][] = [];
	await page.route(
		'**/api/appview/v1/migrations/instagram/post-imports/lookup',
		async (route) => {
			const { sourceKeys } = route.request().postDataJSON() as {
				sourceKeys: string[];
			};
			checked.push(sourceKeys);
			if (sourceKeys.length > 1) {
				await batch;
				return route.fulfill({
					status: 503,
					json: { error: 'unavailable', message: 'Try again.' }
				});
			}
			return route.fulfill({ json: { items: [] } });
		}
	);
	const photo = 'media/posts/202009/scarf.png';
	const video = 'media/reels/202010/sweater.mp4';
	const writer = new ZipWriter(new Uint8ArrayWriter(), {
		useWebWorkers: false
	});
	await writer.add(
		photo,
		new Uint8ArrayReader(readFileSync(resolve('static/app-icon.png')))
	);
	await writer.add(
		video,
		new Uint8ArrayReader(
			readFileSync(resolve('tests/fixtures/video/local-preview.mp4'))
		)
	);
	await writer.add(
		'your_instagram_activity/media/posts_1.json',
		new TextReader(
			JSON.stringify([
				{
					title: 'Older scarf photo',
					creation_timestamp: 1600000000,
					media: [{ uri: 'media/posts/202009/missing.png' }, { uri: photo }]
				},
				{
					title: 'Newer sweater video',
					creation_timestamp: 1602592000,
					media: [{ uri: video }]
				}
			])
		)
	);
	try {
		await page.goto('/settings/instagram/posts');
		await selectExport(page, Buffer.from(await writer.close()), 2);
		const rows = page.locator('.post-import-row');
		await expect(rows.locator('.caption')).toHaveText([
			'Newer sweater video',
			'Older scarf photo'
		]);
		const thumbnails = rows.locator('img.post-thumbnail');
		await expect
			.poll(() =>
				thumbnails.evaluateAll((images: HTMLImageElement[]) =>
					images.map(
						(image) =>
							image.complete &&
							image.naturalWidth > 0 &&
							image.src.startsWith('blob:')
					)
				)
			)
			.toEqual([true, true]);
		await expect(
			page.getByText('Checking previous imports…', { exact: true })
		).toBeVisible();
		await expect(
			rows.nth(1).getByRole('button', { name: 'Create project' })
		).toBeEnabled();
		await rows.first().getByRole('button', { name: 'Create post' }).click();
		await expect.poll(() => checked.map((keys) => keys.length)).toEqual([2, 1]);
		expect(checked[1]).toEqual([checked[0][0]]);
	} finally {
		finishBatch();
	}
	await expect(page).toHaveURL(/\/compose\?draft=/);
	await expect(page.locator('.video-attachment')).toBeVisible();
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/instagram\/posts$/);
	const photoRow = page
		.locator('.post-import-row')
		.filter({ hasText: 'Older scarf photo' });
	await photoRow.getByRole('button', { name: 'Create project' }).click();
	await expect(photoRow.getByRole('alert')).toContainText(
		'missing or ambiguous'
	);
	await photoRow.getByRole('button', { name: 'Continue with 1 photo' }).click();
	await expect(page).toHaveURL(/\/compose\/project\?draft=/);
	await expect(page.getByLabel('Project description')).toHaveValue(
		'Older scarf photo'
	);
	await expect(
		page.getByRole('button', { name: /Photos .*0 \/ 1 described/ })
	).toBeVisible();
});
