import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import {
	ZipWriter,
	Uint8ArrayWriter,
	Uint8ArrayReader,
	TextReader
} from '@zip.js/zip.js';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const caption = 'My first knitted scarf, brought over from Instagram.';
const createdAt = '2020-09-13T12:26:40.000Z';
async function archive(mediaCount = 1, videos = 0) {
	const writer = new ZipWriter(new Uint8ArrayWriter(), {
		useWebWorkers: false
	});
	const media = [];
	for (let index = 0; index < mediaCount; index++) {
		const uri = `media/posts/202009/scarf-${index}.png`;
		media.push({ uri, creation_timestamp: 1600000000 });
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
			JSON.stringify([
				{ title: caption, creation_timestamp: 1600000000, media }
			])
		)
	);
	return Buffer.from(await writer.close());
}
async function selectExport(page: Page, bytes: Buffer) {
	await page
		.getByLabel('Choose your Instagram posts ZIP export', { exact: true })
		.setInputFiles({
			name: 'instagram-posts.zip',
			mimeType: 'application/zip',
			buffer: bytes
		});
	await expect(page.locator('.post-import-row')).toHaveCount(1);
	await expect(
		page.getByText('Checking previous imports…', { exact: true })
	).toHaveCount(0);
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

test('verified users review locally, cancel, create a project and see durable duplicate status', async ({
	page
}) => {
	let verified = false;
	let published:
		{ sourceKey: string; uri: string; kind: 'project' } | undefined;
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
		(route) => route.fulfill({ json: { items: published ? [published] : [] } })
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
		page.getByLabel('Choose your Instagram posts ZIP export', { exact: true })
	).toHaveCount(0);
	verified = true;
	await page.reload();
	const bytes = await archive();
	await selectExport(page, bytes);
	await expect(page.locator('.post-thumbnail').first()).toHaveAttribute(
		'src',
		/^blob:/
	);
	expect(
		networkBodies.some(
			(body) => body.includes(caption) || body.includes('scarf-0.png')
		)
	).toBe(false);
	await page
		.locator('.post-import-row')
		.getByRole('button', { name: 'Create post', exact: true })
		.click();
	await expect(page).toHaveURL(/\/compose\?draft=/);
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toHaveValue(caption);
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/instagram$/);
	await expect(page.locator('.post-import-row')).toHaveCount(1);
	await expect(
		page.locator('.post-import-row').getByText('Posted', { exact: true })
	).toHaveCount(0);
	await page
		.locator('.post-import-row')
		.getByRole('button', { name: 'Create project', exact: true })
		.click();
	await expect(page).toHaveURL(/\/compose\/project\?draft=/);
	await expect(page.getByLabel('Project description')).toHaveValue(caption);
	await page
		.getByRole('combobox', { name: 'Primary craft required', exact: true })
		.click();
	await page.getByRole('option', { name: 'Knitting', exact: true }).click();
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/instagram$/, { timeout: 20000 });
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
	await expect(
		page
			.locator('.post-import-row')
			.getByText('Project created', { exact: false })
	).toBeVisible();
	await expect(
		page
			.locator('.post-import-row')
			.getByRole('link', { name: 'View project', exact: true })
	).toHaveAttribute('href', /^\/posts\/did%3Aplc%3Aalice\//);
	expect(creates).toBe(1);
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({ json: { items: [] } })
	);
	await page.getByRole('link', { name: 'Home', exact: true }).first().click();
	await expect(page).toHaveURL(/\/feed$/);
	await expect(
		page.locator('.post-card').filter({ hasText: caption })
	).toHaveCount(0);
});

test('mixed and excess media require an explicit supported selection', async ({
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
	await page.route(
		'**/api/appview/v1/migrations/instagram/post-imports/lookup',
		(route) => route.fulfill({ json: { items: [] } })
	);
	await page.goto('/settings/instagram');
	await selectExport(page, await archive(9, 2));
	await page
		.locator('.post-import-row')
		.getByRole('button', { name: 'Create post', exact: true })
		.click();
	await expect(page).toHaveURL(/\/settings\/instagram$/);
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
	await page
		.getByRole('button', { name: 'Continue to post composer', exact: true })
		.click();
	await expect(page).toHaveURL(/\/compose\?draft=/);
	await expect(page.locator('.video-attachment')).toBeVisible();
	await expect(page.locator('.photo-attachments img')).toHaveCount(0);
});
