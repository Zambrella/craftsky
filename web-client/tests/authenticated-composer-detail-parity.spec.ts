import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import type { Post } from '../src/lib/api/types';

const base = 'http://127.0.0.1:19846';
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const rootRef = {
	uri: 'at://did:plc:bob/social.craftsky.feed.post/root',
	cid: 'root-cid'
};
const longText =
	'🧶 A complete comment target.\n' +
	'Every line remains readable when writing a response.\n'.repeat(12) +
	'The final line is visible too.';
function target(rkey = 'root', extra: Partial<Post> = {}): Post {
	return {
		uri: `at://did:plc:bob/social.craftsky.feed.post/${rkey}`,
		cid: rkey + '-cid',
		rkey,
		text: longText,
		author: {
			did: 'did:plc:bob',
			handle: 'bob.test',
			displayName: 'Bob Maker',
			crafts: ['knitting']
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
		viewerHasSaved: false,
		...extra
	};
}
async function mockTargets(page: Page) {
	await page.route('**/api/appview/v1/posts/**', (route) => {
		if (route.request().method() !== 'GET') return route.continue();
		const path = new URL(route.request().url()).pathname;
		if (path.endsWith('/root'))
			return route.fulfill({
				json: target('root', {
					images: [
						{
							cid: 'current-image',
							alt: 'Original selected image',
							thumb: 'https://fixture.craftsky.test/app-icon.png',
							fullsize: 'https://fixture.craftsky.test/app-icon.png'
						}
					]
				})
			});
		if (path.endsWith('/nested'))
			return route.fulfill({
				json: target('nested', { reply: { root: rootRef, parent: rootRef } })
			});
		return route.continue();
	});
}
async function imageFixture(page: Page) {
	const value = await page.evaluate(() => {
		const canvas = document.createElement('canvas');
		canvas.width = 8;
		canvas.height = 10;
		canvas.getContext('2d')!.fillRect(0, 0, 8, 10);
		return canvas.toDataURL('image/png').split(',')[1];
	});
	return Buffer.from(value, 'base64');
}
async function savedDrafts(
	page: Page
): Promise<{ text: string; quoteImageCid?: string }[]> {
	return page.evaluate(async () => {
		if (
			!(await indexedDB.databases()).some(
				(value) => value.name === 'craftsky-private-drafts'
			)
		)
			return [];
		return new Promise<{ text: string; quoteImageCid?: string }[]>(
			(resolve, reject) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onerror = () =>
					reject(new Error('Could not inspect fixture drafts'));
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readonly'),
						result = tx.objectStore('drafts').getAll();
					result.onsuccess = () =>
						resolve(
							result.result
								.filter((value) => value.owner === 'did:plc:alice')
								.map(({ text, quoteImageCid }) => ({ text, quoteImageCid }))
						);
					tx.oncomplete = () => db.close();
				};
			}
		);
	});
}
async function calls() {
	return (await (
		await fetch('http://127.0.0.1:19847/__test/requests')
	).json()) as {
		method: string;
		path: string;
		body: Record<string, unknown>;
	}[];
}

test.beforeEach(async ({ page, context }) => {
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
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
});

test('root comments and nested replies show native identity and the full plain-text target without media or draft actions', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await mockTargets(page);
	await page.goto('/compose/reply?did=did%3Aplc%3Abob&rkey=root');
	await expect(
		page.getByRole('heading', { name: 'Comment', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('textbox', { name: 'Write your comment', exact: true })
	).toBeVisible();
	const context = page.getByRole('blockquote', { name: 'Replying to' });
	await expect(context).toContainText('Bob Maker');
	await expect(context).toContainText('@bob.test');
	await expect(context.locator('p')).toHaveText(longText);
	await expect(context.locator('p')).toHaveCSS('-webkit-line-clamp', 'none');
	await expect(context.locator('video, .image-carousel')).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Save draft', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Comment', exact: true })
	).toBeVisible();
	await page.goto('/compose/reply?did=did%3Aplc%3Abob&rkey=nested');
	await expect(
		page.getByRole('heading', { name: 'Reply', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('textbox', { name: 'Write your reply', exact: true })
	).toBeVisible();
	expect(
		(await calls()).filter(
			(call) => call.method === 'POST' && /\/v1\/posts(?:\/|$)/.test(call.path)
		)
	).toEqual([]);
});

test('quotes stay compact and accept only the selected CID present on the freshly loaded target', async ({
	page
}) => {
	await mockTargets(page);
	await page.goto(
		'/compose/quote?did=did%3Aplc%3Abob&rkey=root&imageCid=current-image'
	);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Selected image quote');
	await expect(
		page.getByText('Draft saved on this device', { exact: true })
	).toBeVisible();
	expect(
		(await savedDrafts(page)).find(
			(value) => value.text === 'Selected image quote'
		)?.quoteImageCid
	).toBe('current-image');

	await expect(
		page.getByRole('blockquote', { name: 'Quoted post' }).locator('p')
	).toHaveCSS('-webkit-line-clamp', '3');
	await expect(
		page.getByRole('button', { name: 'Save draft', exact: true })
	).toHaveCount(0);
	await page.goto(
		'/compose/quote?did=did%3Aplc%3Abob&rkey=root&imageCid=unknown-stale-image'
	);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Unknown image quote');
	await expect(
		page.getByText('Draft saved on this device', { exact: true })
	).toBeVisible();
	expect(
		(await savedDrafts(page)).find(
			(value) => value.text === 'Unknown image quote'
		)?.quoteImageCid
	).toBeUndefined();
	await expect(
		page.getByText(
			'The selected image is no longer available. Your quote will use the original post.'
		)
	).toBeVisible();
});

test('post languages use the selected-language menu and a separate searchable add dialog', async ({
	page
}) => {
	await page.goto('/compose');
	await page
		.getByRole('button', { name: 'Post languages', exact: true })
		.click();
	const menu = page.getByRole('menu', { name: 'Post languages', exact: true });
	await expect(
		menu.getByRole('menuitemcheckbox', { name: 'Remove English' })
	).toBeDisabled();
	await menu.getByRole('menuitem', { name: 'Add language' }).click();
	const search = page.getByRole('dialog', { name: 'Add post language' });
	await expect(
		search.getByRole('searchbox', { name: 'Search languages' })
	).toBeFocused();
	await search
		.getByRole('searchbox', { name: 'Search languages' })
		.fill('French');
	await search.getByRole('button', { name: /^French\s+fr$/ }).click();
	await expect(search).not.toBeVisible();
	await page
		.getByRole('button', { name: 'Post languages', exact: true })
		.click();
	await expect(
		menu.getByRole('menuitemcheckbox', { name: 'Remove French' })
	).toBeChecked();
	await menu.getByRole('menuitemcheckbox', { name: 'Remove French' }).click();
	await page
		.getByRole('button', { name: 'Post languages', exact: true })
		.click();
	await expect(
		menu.getByRole('menuitemcheckbox', { name: 'Remove English' })
	).toBeDisabled();
	await page.keyboard.press('Escape');
});

test('When preserves the local scheduling instant through a saved draft and Now clears it', async ({
	page
}) => {
	await page.goto('/compose');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Private schedule review');
	await page.getByRole('button', { name: 'When', exact: true }).click();
	await page
		.getByRole('menuitemradio', { name: 'Schedule for later', exact: true })
		.click();
	const dialog = page.getByRole('dialog', {
		name: 'Schedule for later',
		exact: true
	});
	const tomorrow = new Date(Date.now() + 86_400_000),
		local = new Date(tomorrow.getTime() - tomorrow.getTimezoneOffset() * 60_000)
			.toISOString()
			.slice(0, 16);
	await dialog.getByLabel('Publish date and time').fill(local);
	await dialog.getByRole('button', { name: 'Set time' }).click();
	await expect(dialog).not.toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Save schedule', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page).toHaveURL(/\/feed$/);
	await expect(
		page.getByRole('status').filter({ hasText: 'Draft saved' })
	).toBeVisible();
	await page.goto('/drafts');
	await page
		.getByRole('link', { name: 'Continue editing', exact: true })
		.click();
	await page.getByRole('button', { name: 'When', exact: true }).click();
	await expect(
		page.getByRole('menuitemradio', { name: 'Schedule for later', exact: true })
	).toBeChecked();
	await page
		.getByRole('menuitemradio', { name: 'Schedule for later', exact: true })
		.click();
	await expect(dialog.getByLabel('Publish date and time')).toHaveValue(local);
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await page.getByRole('button', { name: 'When', exact: true }).click();
	await page.getByRole('menuitemradio', { name: 'Now', exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Post', exact: true })
	).toBeVisible();
	expect(
		(await calls()).filter(
			(call) =>
				call.method === 'POST' &&
				/\/v1\/(posts|scheduled-posts)(?:\/|$)/.test(call.path)
		)
	).toEqual([]);
});

test('crop editing opens a full compact viewport, traps edits and returns focus without changing the original on Escape', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/compose');
	await page.locator('input[type=file][multiple]').setInputFiles({
		name: 'private.png',
		mimeType: 'image/png',
		buffer: await imageFixture(page)
	});
	await page.getByLabel('Image description').fill('Original private image');
	const trigger = page.getByRole('button', {
		name: 'Edit photo',
		exact: true
	});
	await trigger.click();
	const dialog = page.getByRole('dialog', { name: 'Edit photo', exact: true });
	await expect(dialog).toBeVisible();
	const bounds = await dialog.boundingBox();
	expect(bounds?.width).toBe(390);
	expect(bounds?.height).toBe(844);
	await expect(page.locator('.save-draft')).toBeDisabled();
	await dialog
		.getByRole('button', { name: 'Crop bottom right corner' })
		.press('Shift+ArrowLeft');
	await dialog.getByRole('button', { name: 'Rotate photo' }).click();
	const corner = await dialog
		.getByRole('button', { name: 'Crop bottom right corner' })
		.boundingBox();
	expect(corner).not.toBeNull();
	await page.mouse.move(
		corner!.x + corner!.width / 2,
		corner!.y + corner!.height / 2
	);
	await page.mouse.down();
	await page.mouse.move(
		corner!.x + corner!.width / 2 + 30,
		corner!.y + corner!.height / 2,
		{ steps: 5 }
	);
	await page.mouse.up();
	await expect(dialog.locator('.crop-selection')).not.toHaveAttribute(
		'style',
		/height: 100%/
	);
	await dialog.screenshot({ path: test.info().outputPath('crop-editor.png') });
	await page.keyboard.press('Escape');
	await expect(dialog).not.toBeVisible();
	await expect(trigger).toBeFocused();
	await expect(
		page.getByRole('img', { name: 'Original private image', exact: true })
	).toHaveJSProperty('naturalWidth', 8);
	await trigger.click();
	await expect(dialog.locator('.crop-selection')).toHaveAttribute(
		'style',
		/width: 100%/
	);
	await dialog.getByRole('button', { name: 'Apply edit', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	expect(
		(await calls()).filter(
			(call) => call.method === 'POST' && call.path.includes('/blobs/')
		)
	).toEqual([]);
});

test('Send via chat preserves the account conversation draft and adds the controlled public link once for review', async ({
	page
}) => {
	await fetch('http://127.0.0.1:19847/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ visual: true })
	});
	await page.goto('/messages/conversations/visual-accepted');
	await page
		.getByRole('textbox', { name: 'Message', exact: true })
		.fill('My existing private draft');
	const link = 'https://app.craftsky.social/posts/did%3Aplc%3Aalice/first';
	await page.goto('/messages/new?link=' + encodeURIComponent(link));
	await expect(
		page.getByText(
			'Choose someone to send this link to. It will be added to your draft for review.'
		)
	).toBeVisible();
	await page.locator('.conversation-list button.recipient').first().click();
	await expect(page).toHaveURL(/\/messages\/conversations\/visual-accepted$/);
	await expect(
		page.getByRole('textbox', { name: 'Message', exact: true })
	).toHaveValue('My existing private draft\n\n' + link);
	await page.goto('/messages/new?link=' + encodeURIComponent(link));
	await page.locator('.conversation-list button.recipient').first().click();
	await expect(
		page.getByRole('textbox', { name: 'Message', exact: true })
	).toHaveValue('My existing private draft\n\n' + link);
	await page.reload();
	await expect(
		page.getByRole('textbox', { name: 'Message', exact: true })
	).toHaveValue('My existing private draft\n\n' + link);
	expect(
		(await calls()).filter(
			(call) =>
				call.method === 'POST' &&
				/\/chat\/conversations\/[^/]+\/messages$/.test(call.path)
		)
	).toEqual([]);
});

test('full scheduled capacity disables a new scheduled slot while keeping Post now and draft review available', async ({
	page
}) => {
	await page.route('**/api/appview/v1/scheduled-posts', (route) =>
		route.fulfill({
			json: { items: [{ id: 'one' }, { id: 'two' }, { id: 'three' }] }
		})
	);
	await page.goto('/compose');
	await expect(
		page.getByText('you already have 3 scheduled posts', { exact: false })
	).toBeVisible();
	await page.getByRole('button', { name: 'When', exact: true }).click();
	await expect(
		page.getByRole('menuitemradio', { name: 'Schedule for later', exact: true })
	).toBeDisabled();
	await expect(
		page.getByRole('menuitemradio', { name: 'Now', exact: true })
	).toBeEnabled();
	await page.keyboard.press('Escape');
	await expect(
		page.getByRole('button', { name: 'Post', exact: true })
	).toBeEnabled();
	await expect(
		page.getByRole('link', { name: 'Manage scheduled posts', exact: true })
	).toHaveAttribute('href', '/scheduled');
});

test('Save closes only after persistence, and Close keeps or restores an existing autosaved draft on explicit choice', async ({
	page
}) => {
	await page.goto('/compose');
	const text = page.getByRole('textbox', {
		name: 'What would you like to share?',
		exact: true
	});
	await text.fill('Original saved draft');
	await page.locator('input[type=file][multiple]').setInputFiles({
		name: 'private.png',
		mimeType: 'image/png',
		buffer: await imageFixture(page)
	});
	await page.getByLabel('Image description').fill('Original image description');
	await page.getByRole('button', { name: 'Back to post', exact: true }).click();
	await expect(
		page.getByText('Draft saved on this device', { exact: true })
	).toBeVisible();
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	const close = page.getByRole('dialog', {
		name: 'Save your draft?',
		exact: true
	});
	await expect(close).toBeVisible();
	await close
		.getByRole('button', { name: 'Keep editing', exact: true })
		.click();
	await expect(text).toHaveValue('Original saved draft');
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await close.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page).toHaveURL(/\/feed$/);
	await expect(
		page.getByRole('status').filter({ hasText: 'Draft saved' })
	).toBeVisible();
	await page.goto('/drafts');
	await page
		.getByRole('link', { name: 'Continue editing', exact: true })
		.click();
	await expect(text).toHaveValue('Original saved draft');
	await expect(
		page.getByRole('button', { name: 'Save changes', exact: true })
	).toBeDisabled();
	await text.fill('Autosaved changes to discard');
	await page.getByRole('button', { name: /Photos.*Reorder photos/ }).click();
	await page.getByLabel('Image description').fill('Changed image description');
	await page.getByRole('button', { name: 'Back to post', exact: true }).click();
	await expect(
		page.getByText('Draft saved on this device', { exact: true })
	).toBeVisible();
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await close
		.getByRole('button', { name: 'Discard changes', exact: true })
		.click();
	await expect(page).toHaveURL(/\/drafts$/);
	await page
		.getByRole('link', { name: 'Continue editing', exact: true })
		.click();
	await expect(text).toHaveValue('Original saved draft');
	await page.getByRole('button', { name: /Photos.*Reorder photos/ }).click();
	await expect(page.getByLabel('Image description')).toHaveValue(
		'Original image description'
	);
	await expect(
		page.getByRole('img', { name: 'Original image description', exact: true })
	).toHaveJSProperty('naturalWidth', 8);
});

test('explicit Discard removes a new autosaved draft and leaves another account’s recovery intact', async ({
	page,
	context
}) => {
	await page.goto('/compose');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Alice explicitly discarded');
	await expect(
		page.getByText('Draft saved on this device', { exact: true })
	).toBeVisible();
	await page.evaluate(
		() =>
			new Promise<void>((resolve, reject) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readwrite'),
						store = tx.objectStore('drafts'),
						get = store.getAll();
					get.onsuccess = () => {
						const original = get.result.find(
							(value) => value.owner === 'did:plc:alice'
						);
						store.put({
							...original,
							owner: 'did:plc:bob',
							text: 'Bob protected draft'
						});
					};
					tx.oncomplete = () => {
						db.close();
						resolve();
					};
					tx.onerror = () => reject(new Error('Could not seed owner fixture'));
				};
			})
	);
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await page
		.getByRole('dialog', { name: 'Save your draft?', exact: true })
		.getByRole('button', { name: 'Discard', exact: true })
		.click();
	await expect(page).toHaveURL(/\/feed$/);
	await page.goto('/drafts');
	await expect(
		page.getByText('Alice explicitly discarded', { exact: true })
	).toHaveCount(0);
	const bob = {
		id: 'c6dc7d6a-4bff-45f0-87ee-d42634646546',
		did: 'did:plc:bob',
		handle: 'bob.test',
		token: Buffer.alloc(32, 2).toString('base64url')
	};
	await context.addCookies([
		{
			name: 'craftsky_account_' + bob.id,
			value: Buffer.from(JSON.stringify(bob)).toString('base64url'),
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		},
		{
			name: 'craftsky_active',
			value: bob.id,
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	await page.goto('/drafts');
	await expect(
		page.getByText('Bob protected draft', { exact: true })
	).toBeVisible();
});

test('closing an uncertain write keeps its exact frozen recovery and never offers destructive discard', async ({
	page
}) => {
	await page.goto('/compose');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Frozen review fixture');
	await expect(
		page.getByText('Draft saved on this device', { exact: true })
	).toBeVisible();
	const frozen = await page.evaluate(
		() =>
			new Promise<{ id: string; key: string }>((resolve, reject) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readwrite'),
						store = tx.objectStore('drafts'),
						get = store.getAll();
					let result: { id: string; key: string };
					get.onsuccess = () => {
						const draft = get.result.find(
							(value) => value.owner === 'did:plc:alice'
						);
						draft.pending = {
							key: crypto.randomUUID(),
							kind: 'create',
							body: { text: draft.text, langs: ['en'], sponsored: false }
						};
						store.put(draft);
						result = { id: draft.id, key: draft.pending.key };
					};
					tx.oncomplete = () => {
						db.close();
						resolve(result);
					};
					tx.onerror = () =>
						reject(new Error('Could not seed uncertain fixture'));
				};
			})
	);
	await page.goto('/compose?draft=' + frozen.id);
	await expect(
		page.getByRole('heading', { name: 'Waiting for confirmation', exact: true })
	).toBeVisible();
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await expect(page).toHaveURL(/\/drafts$/);
	await expect(
		page.getByRole('dialog', { name: 'Save your draft?', exact: true })
	).toHaveCount(0);
	const record = await page.evaluate(
		(id) =>
			new Promise<{ key: string; body: { text: string } }>((resolve) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readonly'),
						get = tx.objectStore('drafts').get(['did:plc:alice', id]);
					get.onsuccess = () => resolve(get.result.pending);
					tx.oncomplete = () => db.close();
				};
			}),
		frozen.id
	);
	expect(record.key).toBe(frozen.key);
	expect(record.body.text).toBe('Frozen review fixture');
	expect(
		(await calls()).filter(
			(call) => call.method === 'POST' && /\/v1\/posts(?:\/|$)/.test(call.path)
		)
	).toEqual([]);
});

test('media export blocks Done, cancellation and Save until the exact image edit completes', async ({
	page
}) => {
	await page.goto('/compose');
	await page.locator('input[type=file][multiple]').setInputFiles({
		name: 'private.png',
		mimeType: 'image/png',
		buffer: await imageFixture(page)
	});
	await expect(
		page.getByText('Draft saved on this device', { exact: true })
	).toBeAttached();
	await page.evaluate(() => {
		const original = HTMLCanvasElement.prototype.toBlob;
		HTMLCanvasElement.prototype.toBlob = function (callback, ...args) {
			original.call(
				this,
				(blob) => {
					(window as unknown as { releaseCrop: () => void }).releaseCrop = () =>
						callback(blob);
				},
				...args
			);
		};
	});
	await page.getByRole('button', { name: 'Edit photo', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Edit photo', exact: true });
	await dialog.getByRole('button', { name: 'Apply edit', exact: true }).click();
	await expect(
		dialog.getByRole('button', { name: 'Cancel edit', exact: true })
	).toBeDisabled();
	await expect(
		dialog.getByRole('button', { name: 'Apply edit', exact: true })
	).toBeDisabled();
	await expect(page.locator('.save-draft')).toBeDisabled();
	await page.keyboard.press('Escape');
	await expect(dialog).toBeVisible();
	await expect
		.poll(() =>
			page.evaluate(
				() =>
					typeof (window as unknown as { releaseCrop?: () => void }).releaseCrop
			)
		)
		.toBe('function');
	await page.evaluate(() =>
		(window as unknown as { releaseCrop: () => void }).releaseCrop()
	);
	await expect(dialog).not.toBeVisible();
	await expect(page.locator('.save-draft')).toBeEnabled();
	expect(
		(await calls()).filter(
			(call) => call.method === 'POST' && call.path.includes('/blobs/')
		)
	).toEqual([]);
});

test('Sponsored opens the native compact bottom modal and applies its switch for review', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/compose');
	await page
		.getByRole('button', { name: 'Sponsored options', exact: true })
		.click();
	const dialog = page.getByRole('dialog', { name: 'Sponsored', exact: true });
	await expect(dialog).toBeVisible();
	const bounds = await dialog.boundingBox();
	expect(bounds && bounds.y + bounds.height).toBeCloseTo(844, 0);
	await expect(dialog).toContainText(
		'This post includes sponsorship or other commercial consideration'
	);
	await dialog
		.getByRole('switch', { name: 'Sponsored content', exact: true })
		.check();
	await expect(dialog).not.toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Sponsored options', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	await page
		.getByRole('button', { name: 'Sponsored options', exact: true })
		.click();
	await expect(
		dialog.getByRole('switch', { name: 'Sponsored content', exact: true })
	).toBeChecked();
	await page.keyboard.press('Escape');
});

test('required text stays inline and the native grapheme counter has no grouping or extra spaces', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto('/compose');
	const field = page.locator('.post-text-label .facet-input'),
		label = field.locator('label'),
		required = label.locator('.facet-hint');
	await expect(required).toHaveText('required');
	await expect(label).toHaveCSS('display', 'flex');
	await expect(required).toHaveCSS('font-weight', '600');
	expect(
		await label.evaluate((node) => node.getBoundingClientRect().height)
	).toBeLessThan(32);
	await expect(required).toHaveCSS('font-size', '12px');
	await expect(page.locator('.character-count')).toHaveText('0/2000');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('🧶');
	await expect(page.locator('.character-count')).toHaveText('1/2000');
	await expect(page.getByText('Link preview', { exact: true })).toHaveCount(0);
	await expect(page.getByLabel('Link to preview')).toHaveCount(0);
});

test('complete body URLs automatically load native previews through AppView, with carousel, dismissal, Undo and media suppression', async ({
	page
}) => {
	const requests: { url: string; account: string | undefined }[] = [];
	await page.route('**/api/appview/v1/link-previews', (route) => {
		const body = route.request().postDataJSON();
		requests.push({
			url: body.url,
			account: route.request().headers()['x-craftsky-account-id']
		});
		return route.fulfill({
			json: {
				url: body.url,
				title: new URL(body.url).hostname,
				description: 'Automatic pattern details'
			}
		});
	});
	await page.goto('/compose');
	const text = page.getByRole('textbox', {
		name: 'What would you like to share?',
		exact: true
	});
	await text.fill(
		'My pattern https://one.example/pattern#notes and https://two.example/pattern '
	);
	const card = page.getByRole('region', { name: 'Link preview', exact: true });
	await expect(
		card.getByRole('heading', { name: 'one.example', exact: true })
	).toBeVisible();
	await expect(
		card.getByText('Link preview 1 of 2', { exact: true })
	).toBeVisible();
	expect(requests).toEqual([
		{ url: 'https://one.example/pattern', account: alice.id },
		{ url: 'https://two.example/pattern', account: alice.id }
	]);
	await card
		.getByRole('button', { name: 'Next link preview', exact: true })
		.click();
	await expect(
		card.getByRole('heading', { name: 'two.example', exact: true })
	).toBeVisible();
	await card
		.getByRole('button', { name: 'Remove preview', exact: true })
		.click();
	await expect(card).toHaveCount(0);
	await text.fill(
		'My pattern https://one.example/pattern#notes and https://two.example/pattern more notes'
	);
	await expect(card).toHaveCount(0);
	await page.getByRole('button', { name: 'Undo', exact: true }).click();
	await expect(card).toBeVisible();
	expect(requests).toHaveLength(2);
	await page.locator('input[type=file][multiple]').setInputFiles({
		name: 'private.png',
		mimeType: 'image/png',
		buffer: await imageFixture(page)
	});
	await expect(card).toHaveCount(0);
	await page
		.locator('.media-card')
		.getByRole('button', { name: 'Remove image 1', exact: true })
		.click();
	await page.getByRole('button', { name: 'Back to post', exact: true }).click();
	await expect(card).toBeVisible();
	expect(requests).toHaveLength(2);
	expect(
		(await calls()).filter(
			(call) => call.method === 'POST' && /\/v1\/posts(?:\/|$)/.test(call.path)
		)
	).toEqual([]);
});

test('a delayed preview for replaced text cannot overwrite the current URL or its saved draft', async ({
	page
}) => {
	let held: import('@playwright/test').Route | undefined;
	await page.route('**/api/appview/v1/link-previews', (route) => {
		const body = route.request().postDataJSON();
		if (body.url === 'https://old.example/pattern') {
			held = route;
			return;
		}
		return route.fulfill({
			json: {
				url: body.url,
				title: 'Current pattern',
				description: 'Current details'
			}
		});
	});
	await page.goto('/compose');
	const text = page.getByRole('textbox', {
		name: 'What would you like to share?',
		exact: true
	});
	await text.fill('https://old.example/pattern ');
	await expect.poll(() => !!held).toBe(true);
	await text.fill('https://current.example/pattern ');
	const card = page.getByRole('region', { name: 'Link preview', exact: true });
	await expect(
		card.getByRole('heading', { name: 'Current pattern', exact: true })
	).toBeVisible();
	await held!
		.fulfill({
			json: {
				url: 'https://old.example/pattern',
				title: 'Stale pattern',
				description: 'Stale details'
			}
		})
		.catch(() => {});
	await expect(
		card.getByRole('heading', { name: 'Stale pattern', exact: true })
	).toHaveCount(0);
	await expect(
		card.getByRole('heading', { name: 'Current pattern', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Draft saved on this device', { exact: true })
	).toBeVisible();
	const uri = await page.evaluate(
		() =>
			new Promise<string>((resolve) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readonly'),
						get = tx.objectStore('drafts').getAll();
					get.onsuccess = () =>
						resolve(
							get.result.find((value) => value.owner === 'did:plc:alice')
								.external.uri
						);
					tx.oncomplete = () => db.close();
				};
			})
	);
	expect(uri).toBe('https://current.example/pattern');
});

test('photo selection opens details, retains descriptions on return, and supports reorder and removal', async ({
	page
}) => {
	const errors: string[] = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto('/compose/project');
	const buffer = await imageFixture(page);
	await page
		.locator('input[type=file][multiple]')
		.first()
		.setInputFiles([
			{ name: 'first.png', mimeType: 'image/png', buffer },
			{ name: 'second.png', mimeType: 'image/png', buffer }
		]);
	const details = page.getByRole('region', {
		name: 'Photo details',
		exact: true
	});
	await expect(details).toBeVisible();
	await details
		.getByRole('textbox', { name: 'Image description', exact: true })
		.first()
		.fill('Blue cotton fabric');
	await details
		.getByRole('button', { name: 'Move image 1 later', exact: true })
		.click();
	await expect(
		details
			.getByRole('textbox', { name: 'Image description', exact: true })
			.nth(1)
	).toHaveValue('Blue cotton fabric');
	await details.screenshot({
		path: test.info().outputPath('photo-details.png')
	});
	const audit = await new AxeBuilder({ page })
		.include('.photo-details-page')
		.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
		.analyze();
	expect(audit.violations).toEqual([]);
	await details
		.getByRole('button', { name: 'Back to post', exact: true })
		.click();
	await expect(details).not.toBeVisible();
	const summary = page.getByRole('button', { name: /Photos.*Reorder photos/ });
	await expect(summary).toContainText('1 / 2 described');
	await summary.click();
	await expect(
		details
			.getByRole('textbox', { name: 'Image description', exact: true })
			.nth(1)
	).toHaveValue('Blue cotton fabric');
	await details
		.getByRole('button', { name: 'Remove image 1', exact: true })
		.click();
	await expect(
		details.getByRole('textbox', { name: 'Image description', exact: true })
	).toHaveValue('Blue cotton fabric');
	await details
		.getByRole('button', { name: 'Back to post', exact: true })
		.click();
	await page.getByRole('button', { name: 'Clear photos', exact: true }).click();
	await expect(
		page.locator('input[type=file][multiple]').first()
	).toBeAttached();
	expect(errors).toEqual([]);
});

test('browser Back closes photo details without discarding the project', async ({
	page
}) => {
	await page.goto('/feed');
	await page
		.getByRole('button', { name: 'Create post', exact: true })
		.filter({ visible: true })
		.click();
	await page
		.getByRole('dialog', { name: 'What would you like to share?' })
		.getByRole('link', { name: /Project post/ })
		.click();
	const buffer = await imageFixture(page);
	await page
		.locator('input[type=file][multiple]')
		.first()
		.setInputFiles({ name: 'photo.png', mimeType: 'image/png', buffer });
	const details = page.getByRole('region', {
		name: 'Photo details',
		exact: true
	});
	await expect(details).toBeVisible();
	await details
		.getByRole('textbox', { name: 'Image description', exact: true })
		.fill('Retained photo description');
	await page.goBack();
	await expect(details).toBeHidden();
	await expect(page).toHaveURL(/\/compose\/project$/);
	await expect(page.getByRole('dialog', { name: 'Save draft?' })).toBeHidden();
	await page.getByRole('button', { name: /Photos.*Reorder photos/ }).click();
	await expect(
		details.getByRole('textbox', { name: 'Image description', exact: true })
	).toHaveValue('Retained photo description');
	await page.goBack();
	await page.goForward();
	await expect(details).toBeVisible();
	await expect(
		details.getByRole('textbox', { name: 'Image description', exact: true })
	).toHaveValue('Retained photo description');
	await page.getByRole('link', { name: 'Drafts', exact: true }).click();
	await expect(
		page.getByRole('dialog', { name: 'Save your draft?' })
	).toBeVisible();
});

for (const width of [402, 1280]) {
	test(`photo alt text retains focus, caret and preview while typing at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/compose');
		await page.locator('input[type=file][multiple]').setInputFiles({
			name: 'focus.png',
			mimeType: 'image/png',
			buffer: await imageFixture(page)
		});
		const details = page.getByRole('region', {
			name: 'Photo details',
			exact: true
		});
		const input = details.getByRole('textbox', {
			name: 'Image description',
			exact: true
		});
		const preview = details.locator('.photo-preview img');
		const source = await preview.getAttribute('src');
		await input.pressSequentially('Blue cotton', { delay: 25 });
		await expect(input).toBeFocused();
		await expect(input).toHaveValue('Blue cotton');
		await expect(preview).toHaveAttribute('src', source!);
		const thumbnail = page.locator('.photo-summary .thumbnails img');
		const thumbSource = await thumbnail.getAttribute('src');
		await input.evaluate((node) =>
			(node as HTMLTextAreaElement).setSelectionRange(5, 5)
		);
		await input.pressSequentially('soft ', { delay: 25 });
		await expect(input).toBeFocused();
		await expect(input).toHaveValue('Blue soft cotton');
		expect(
			await input.evaluate(
				(node) => (node as HTMLTextAreaElement).selectionStart
			)
		).toBe(10);
		await expect(thumbnail).toHaveAttribute('src', thumbSource!);
		await details
			.getByRole('button', { name: 'Back to post', exact: true })
			.click();
		await page.getByRole('button', { name: /Photos.*Reorder photos/ }).click();
		await expect(input).toHaveValue('Blue soft cotton');
	});
}

for (const width of [402, 1280]) {
	test(`photo handle lifts the complete card and keeps descriptions with reordered photos at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 1900 });
		await page.goto('/compose');
		const buffer = await imageFixture(page);
		await page.locator('input[type=file][multiple]').setInputFiles([
			{ name: 'first.png', mimeType: 'image/png', buffer },
			{ name: 'second.png', mimeType: 'image/png', buffer }
		]);
		const details = page.getByRole('region', {
			name: 'Photo details',
			exact: true
		});
		const rows = details.locator('.photo-row:not(.reorder-ghost)');
		const descriptions = rows.getByRole('textbox', {
			name: 'Image description',
			exact: true
		});
		await descriptions.nth(0).fill('First photo description');
		await descriptions.nth(1).fill('Second photo description');
		const firstSource = await rows
			.first()
			.locator('.photo-preview img')
			.getAttribute('src');
		const ghost = details.locator('.reorder-ghost');
		async function liftFirst() {
			const handle = rows
				.first()
				.getByRole('button', { name: 'Drag to reorder image 1', exact: true });
			await expect(handle).toHaveCSS('touch-action', 'none');
			await handle.scrollIntoViewIfNeeded();
			const grip = (await handle.boundingBox())!;
			const first = (await rows.first().boundingBox())!;
			const second = (await rows.nth(1).boundingBox())!;
			await page.mouse.move(grip.x + grip.width / 2, grip.y + grip.height / 2);
			await page.mouse.down();
			await page.mouse.move(
				grip.x + grip.width / 2,
				grip.y + grip.height / 2 + second.y - first.y + 20,
				{ steps: 12 }
			);
			await expect(ghost.locator('article.media-card')).toBeVisible();
			await expect(ghost.locator('.photo-preview img')).toHaveAttribute(
				'src',
				firstSource!
			);
			await expect(ghost.locator('textarea')).toHaveValue(
				'First photo description'
			);
			await expect(rows.first()).toHaveCSS('opacity', '0.15');
			expect((await ghost.boundingBox())!.height).toBeGreaterThanOrEqual(
				first.height
			);
		}
		await liftFirst();
		await page.screenshot({
			path: test.info().outputPath('lifted-photo-card.png')
		});
		await page.keyboard.press('Escape');
		await page.mouse.up();
		await expect(ghost).toHaveCount(0);
		await expect(details).toBeVisible();
		await expect(descriptions.nth(0)).toHaveValue('First photo description');
		await expect(rows.first()).toHaveCSS('opacity', '1');
		await liftFirst();
		await page.mouse.up();
		await expect(ghost).toHaveCount(0);
		await expect(descriptions.nth(0)).toHaveValue('Second photo description');
		await expect(descriptions.nth(1)).toHaveValue('First photo description');
		await expect(rows.nth(1).locator('.photo-preview img')).toHaveAttribute(
			'src',
			firstSource!
		);
		await rows
			.nth(1)
			.getByRole('button', { name: 'Drag to reorder image 2', exact: true })
			.press('Alt+ArrowUp');
		await expect(descriptions.nth(0)).toHaveValue('First photo description');
	});
}
