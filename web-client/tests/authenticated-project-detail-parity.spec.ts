import { test, expect, type Page } from '@playwright/test';
import type { Draft } from '../src/lib/features/drafts/store.ts';
import type { JsonObject } from '../src/lib/api/types.ts';

const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const draftId = '3f9c35e5-270e-420d-94e4-7380780b6c56';
test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await context.addCookies([
		{
			name: `craftsky_account_${alice.id}`,
			value: Buffer.from(JSON.stringify(alice)).toString('base64url'),
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		},
		{
			name: 'craftsky_active',
			value: alice.id,
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	// These tests exercise device-private drafts only; publishing is never permitted.
	await page.route('**/api/appview/v1/posts', async (route) => {
		if (route.request().method() === 'POST')
			throw new Error('Project detail tests must not publish');
		await route.continue();
	});
});
async function seed(page: Page, common: JsonObject) {
	await page.goto('/app/drafts');
	await expect(
		page.getByRole('heading', { level: 1, name: 'Drafts', exact: true })
	).toBeVisible();
	const draft: Draft = {
		id: draftId,
		owner: alice.did,
		kind: 'project',
		text: 'Project details reference',
		langs: ['en'],
		sponsored: false,
		images: [],
		updatedAt: '2026-11-03T09:00:00Z',
		project: {
			common: {
				craftType: 'social.craftsky.feed.defs#sewing',
				title: 'Native project fields',
				...common
			}
		}
	};
	await page.evaluate(
		(draft: unknown) =>
			new Promise<void>((resolve, reject) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onupgradeneeded = () => {
					const store = request.result.createObjectStore('drafts', {
						keyPath: ['owner', 'id']
					});
					store.createIndex('owner', 'owner');
				};
				request.onerror = () => reject(request.error);
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readwrite');
					tx.objectStore('drafts').put(draft);
					tx.oncomplete = () => {
						db.close();
						resolve();
					};
					tx.onerror = () => {
						db.close();
						reject(tx.error);
					};
				};
			}),
		draft as unknown
	);
	await page.goto(`/app/compose/project?draft=${draftId}`);
	await expect(
		page.getByRole('textbox', { name: 'Project title', exact: true })
	).toHaveValue('Native project fields');
}
async function storedProject(page: Page) {
	return page.evaluate(
		({ owner, id }) =>
			new Promise<JsonObject>((resolve, reject) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onerror = () => reject(request.error);
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readonly'),
						read = tx.objectStore('drafts').get([owner, id]);
					read.onsuccess = () => resolve(read.result.project);
					read.onerror = () => reject(read.error);
					tx.oncomplete = () => db.close();
				};
			}),
		{ owner: alice.did, id: draftId }
	);
}
async function save(page: Page) {
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(page.getByText('Draft saved', { exact: true })).toBeVisible();
}
async function materials(page: Page) {
	await page
		.getByRole('button', { name: 'Materials and style', exact: true })
		.click();
	const dialog = page.getByRole('dialog', {
		name: 'Materials and style',
		exact: true
	});
	await expect(dialog).toBeVisible();
	return dialog;
}

test('searchable project selections retain unknown values and existing craft metadata while excluding the new primary craft', async ({
	page
}) => {
	const knitting = 'social.craftsky.feed.defs#knitting',
		crochet = 'social.craftsky.feed.defs#crochet';
	const retained = {
		craftType: knitting,
		existingIndex: 3,
		details: { futureMetadata: 'kept' }
	};
	await seed(page, {
		additionalCrafts: [
			retained,
			{ craftType: 'social.craftsky.feed.defs#future-craft', existingIndex: 8 }
		],
		colors: ['future-colour'],
		designTags: ['social.craftsky.project.defs#future-design']
	});
	const additional = page.getByRole('searchbox', {
		name: 'Additional crafts',
		exact: true
	});
	await additional.fill('Crochet');
	await page.getByRole('checkbox', { name: 'Crochet', exact: true }).check();
	await additional.press('Escape');
	const dialog = await materials(page);
	const colours = dialog.getByRole('searchbox', {
		name: 'Colours',
		exact: true
	});
	await colours.fill('Red');
	await dialog.getByRole('checkbox', { name: 'Red', exact: true }).check();
	await colours.press('Escape');
	const design = dialog.getByRole('searchbox', {
		name: 'Design',
		exact: true
	});
	await design.fill('Stripes');
	await dialog.getByRole('checkbox', { name: 'Stripes', exact: true }).check();
	await design.press('Escape');
	await dialog
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	await save(page);
	await expect
		.poll(() => storedProject(page))
		.toMatchObject({
			common: {
				additionalCrafts: [
					retained,
					{
						craftType: 'social.craftsky.feed.defs#future-craft',
						existingIndex: 8
					},
					{ craftType: crochet }
				],
				colors: ['future-colour', 'red'],
				designTags: [
					'social.craftsky.project.defs#future-design',
					'social.craftsky.project.defs#stripes'
				]
			}
		});
	await page.goto(`/app/compose/project?draft=${draftId}`);
	await expect(
		page.getByRole('textbox', { name: 'Project title', exact: true })
	).toHaveValue('Native project fields');
	await page
		.getByRole('combobox', { name: 'Primary craft', exact: false })
		.selectOption(knitting);
	await save(page);
	await expect
		.poll(() => storedProject(page))
		.toMatchObject({
			common: {
				craftType: knitting,
				additionalCrafts: [
					{
						craftType: 'social.craftsky.feed.defs#future-craft',
						existingIndex: 8
					},
					{ craftType: crochet }
				]
			}
		});
});

test('material add, edit, reorder and removal keep identities and untouched facets through durable draft saves', async ({
	page
}) => {
	const facets = [
		{
			index: { byteStart: 5, byteEnd: 10 },
			features: [{ $type: 'app.bsky.richtext.facet#tag', tag: 'wool' }]
		}
	];
	await seed(page, {
		materials: [
			{
				text: 'Wool #wool',
				facets,
				existingIndex: 3,
				futureField: 'keep-wool'
			},
			{ text: 'Brass buttons', existingIndex: 7, futureField: 'keep-brass' },
			{ text: 'Linen', existingIndex: 8 }
		]
	});
	const dialog = await materials(page);
	await dialog
		.getByRole('button', { name: 'Move material 1 down', exact: true })
		.click();
	await dialog
		.getByRole('button', { name: 'Edit material Brass buttons', exact: true })
		.click();
	const input = dialog.getByRole('textbox', { name: 'Materials', exact: true });
	await expect(input).toHaveValue('Brass buttons');
	await input.fill('Polished brass buttons');
	await input.press('Enter');
	await expect(input).toHaveValue('');
	await dialog
		.getByRole('button', { name: 'Remove material Linen', exact: true })
		.click();
	await input.fill('Hemp yarn');
	await dialog.getByRole('button', { name: 'Add', exact: true }).click();
	await expect(
		dialog.getByRole('button', {
			name: 'Remove material Hemp yarn',
			exact: true
		})
	).toBeVisible();
	await dialog
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	await save(page);
	await expect
		.poll(() => storedProject(page))
		.toMatchObject({
			common: {
				materials: [
					{
						text: 'Polished brass buttons',
						existingIndex: 7,
						futureField: 'keep-brass'
					},
					{
						text: 'Wool #wool',
						existingIndex: 3,
						futureField: 'keep-wool',
						facets
					},
					{ text: 'Hemp yarn' }
				]
			}
		});
});

test('native material limits allow editing at capacity and discard confirmation preserves already-added materials', async ({
	page
}) => {
	await seed(page, {
		materials: Array.from({ length: 10 }, (_, index) => ({
			text: `Material ${index + 1}`,
			existingIndex: index
		}))
	});
	const dialog = await materials(page),
		input = dialog.getByRole('textbox', { name: 'Materials', exact: true });
	await input.fill('An eleventh material');
	await input.press('Enter');
	await expect(dialog.getByRole('alert')).toContainText(
		'Choose up to ten materials.'
	);
	await input.fill('');
	await dialog
		.getByRole('button', { name: 'Edit material Material 1', exact: true })
		.click();
	await input.fill('a'.repeat(101));
	await input.press('Enter');
	await expect(dialog.getByRole('alert')).toContainText(
		'Materials must be 100 characters or fewer.'
	);
	await input.fill('🧶'.repeat(100));
	await input.press('Enter');
	await expect(input).toHaveValue('');
	await expect(dialog.locator('.material-chip')).toHaveCount(10);
	await input.fill('Pending material');
	page.once('dialog', (dialog) => dialog.dismiss());
	await dialog
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	await expect(dialog).toBeVisible();
	await expect(input).toHaveValue('Pending material');
	page.once('dialog', async (dialog) => {
		expect(dialog.message()).toContain(
			"Materials you've already added will be kept."
		);
		await dialog.accept();
	});
	await dialog
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	await expect(dialog).toBeHidden();
	const reopened = await materials(page);
	await expect(
		reopened.getByRole('textbox', { name: 'Materials', exact: true })
	).toHaveValue('');
	await expect(reopened.locator('.material-chip')).toHaveCount(10);
	await reopened
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	await save(page);
	await expect
		.poll(() => storedProject(page))
		.toMatchObject({
			common: {
				materials: [
					{ text: '🧶'.repeat(100), existingIndex: 0 },
					...Array.from({ length: 9 }, (_, index) => ({
						text: `Material ${index + 2}`,
						existingIndex: index + 1
					}))
				]
			}
		});
});
