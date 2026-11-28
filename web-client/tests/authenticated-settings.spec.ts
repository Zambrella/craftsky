import AxeBuilder from '@axe-core/playwright';
import { test, expect, type BrowserContext, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import bzipFixture from '../src/lib/features/settings/bzip-fixture.json' with { type: 'json' };
import { nativeIconPaths } from '../src/lib/features/shell/native-icons.ts';
test('business images are reviewed, cropped and stripped/recompressed before explicit upload', async ({
	page
}) => {
	await page.goto('/settings/products');
	await page.getByRole('button', { name: 'Add product', exact: true }).click();
	const productDialog = page.locator('.product-composer-form');
	await productDialog.getByLabel('Title', { exact: true }).fill('Rotated wool');
	await productDialog
		.getByLabel('Destination', { exact: true })
		.fill('https://shop.example/rotated');
	const png = await page.evaluate(() => {
		const canvas = document.createElement('canvas');
		canvas.width = 32;
		canvas.height = 48;
		canvas.getContext('2d')!.fillRect(0, 0, 32, 48);
		return canvas.toDataURL('image/png').split(',')[1];
	});
	await productDialog.getByLabel('Add images', { exact: true }).setInputFiles({
		name: 'large-source.png',
		mimeType: 'image/png',
		buffer: Buffer.concat([Buffer.from(png, 'base64'), Buffer.alloc(2_100_000)])
	});
	const details = page.locator('.photo-details-page');
	await expect(details).toBeVisible();
	expect(
		(await calls()).filter((row) => row.path === '/v1/blobs/images')
	).toHaveLength(0);
	await details
		.getByRole('button', { name: 'Edit photo', exact: true })
		.click();
	const editor = page.getByRole('dialog', { name: 'Edit photo', exact: true });
	await editor
		.getByRole('button', { name: 'Rotate photo', exact: true })
		.click();
	await editor.getByRole('button', { name: 'Apply edit', exact: true }).click();
	await expect(editor).not.toBeVisible();
	await details
		.getByLabel('Image description', { exact: true })
		.fill('Portrait image rotated to landscape');
	await details
		.getByRole('button', { name: 'Back to product', exact: true })
		.click();
	await expect(details).not.toBeVisible();
	await productDialog
		.getByRole('button', { name: 'Save product', exact: true })
		.click();
	await expect(
		page.getByRole('button', { name: 'Edit Rotated wool', exact: true })
	).toBeVisible();
	await expect(page).toHaveURL(/\/settings\/products$/);
	const requests = await calls();
	const upload = requests.find(
		(row) => row.path === '/v1/blobs/images'
	) as unknown as { bytes: number; contentType: string };
	expect(upload.contentType).toBe('image/jpeg');
	expect(upload.bytes).toBeLessThan(2_000_000);
	const write = requests.find(
		(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
	);
	expect(write?.body.products).toEqual(
		expect.arrayContaining([
			expect.objectContaining({
				title: 'Rotated wool',
				images: [
					expect.objectContaining({
						alt: 'Portrait image rotated to landscape',
						aspectRatio: { width: 48, height: 32 }
					})
				]
			})
		])
	);
});
test('bundled service worker activates without Firebase credentials or remote requests', async ({
	page
}) => {
	await page.goto('/notifications/settings');
	const scope = await page.evaluate(async () => {
		const registration = await navigator.serviceWorker.register(
			'/service-worker.js',
			{ type: 'module', scope: '/' }
		);
		try {
			const worker =
				registration.active || registration.waiting || registration.installing;
			if (!worker) throw new Error('No service worker created');
			if (worker.state !== 'activated')
				await new Promise<void>((resolve, reject) => {
					const timer = setTimeout(
						() => reject(new Error(`Worker state ${worker.state}`)),
						10000
					);
					worker.addEventListener('statechange', () => {
						if (worker.state === 'activated') {
							clearTimeout(timer);
							resolve();
						}
						if (worker.state === 'redundant') {
							clearTimeout(timer);
							reject(new Error('Worker failed to evaluate'));
						}
					});
				});
			return registration.scope;
		} finally {
			await registration.unregister();
		}
	});
	expect(scope).toBe('http://127.0.0.1:19846/');
});
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
async function login(context: BrowserContext) {
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
}
async function calls() {
	return (await (
		await fetch('http://127.0.0.1:19847/__test/requests')
	).json()) as {
		method: string;
		path: string;
		body: Record<string, unknown>;
		key?: string;
		ifMatch?: string;
		owner: string;
	}[];
}
async function verifyInstagram(page: Page) {
	const exportHelp = page.getByRole('link', {
		name: 'How to export your Instagram following',
		exact: true
	});
	await expect(exportHelp).toHaveCount(0);
	await page
		.getByRole('button', { name: 'Create verification challenge', exact: true })
		.click();
	await expect(
		page.getByText('craftsky-test-challenge', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Copy challenge' })
	).toBeEnabled();
	const dmUrl = 'https://www.instagram.com/direct/t/fixture';
	const openDm = page.getByRole('link', { name: 'Open Instagram DM' });
	await expect(openDm).toHaveAttribute('href', dmUrl);
	await expect(
		page.getByRole('button', { name: 'Cancel verification' })
	).toBeEnabled();
	await page.context().route(dmUrl, (route) => route.fulfill({ body: 'DM' }));
	const [dm] = await Promise.all([
		page.waitForEvent('popup', { timeout: 5000 }),
		openDm.click()
	]);
	await expect(dm).toHaveURL(dmUrl);
	await dm.close();
	await expect(page.getByText('@alice_makes', { exact: true })).toBeVisible();
	await expect(exportHelp).toHaveCount(0);
	await page
		.getByRole('button', {
			name: 'Confirm this is my Instagram account',
			exact: true
		})
		.click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Instagram account verified.' })
	).toBeVisible();
	await expect(exportHelp).toHaveCount(0);
	await page.getByRole('link', { name: 'Import accounts you follow' }).click();
	await expect(page).toHaveURL(/\/settings\/instagram\/following$/);
	await expect(exportHelp).toBeVisible();
}
test.beforeEach(async ({ context, page }) => {
	await page.route('**/api/service-status', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await fetch('http://127.0.0.1:19847/__test/reset');
	await login(context);
});
test('appearance changes immediately and survives reload', async ({ page }) => {
	await page.goto('/settings/appearance');
	await page.getByLabel('Dark', { exact: true }).check();
	await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
	await page.reload();
	await expect(page.getByLabel('Dark', { exact: true })).toBeChecked();
	await page.getByLabel('Light', { exact: true }).check();
	await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
});
test('language preferences use supported choices and replace the complete preference object', async ({
	page
}) => {
	await page.goto('/settings/languages');
	await page
		.getByRole('combobox', { name: 'Primary language', exact: true })
		.fill('Welsh');
	await page.getByRole('option', { name: 'Welsh', exact: true }).click();
	await expect(
		page.getByRole('combobox', { name: 'Primary language', exact: true })
	).toBeEnabled();
	await page
		.getByRole('searchbox', { name: 'Content languages', exact: true })
		.click();
	await page
		.getByRole('searchbox', { name: 'Content languages', exact: true })
		.fill('French');
	await page.getByLabel('French', { exact: true }).click();
	await expect(
		page.getByRole('button', {
			name: 'Remove French from Content languages',
			exact: true
		})
	).toBeVisible();
	await page
		.getByRole('searchbox', { name: 'Content languages', exact: true })
		.fill('English');
	await page.getByLabel('English', { exact: true }).uncheck();
	await expect(
		page.getByRole('status').filter({ hasText: 'Saved.' })
	).toBeVisible();
	const requests = (await calls()).filter(
		(row) => row.method === 'PUT' && row.path === '/v1/languages/preferences'
	);
	expect(requests.map((row) => row.body)).toEqual([
		{ primaryLanguage: 'cy', contentLanguages: ['en'] },
		{ primaryLanguage: 'cy', contentLanguages: ['en', 'fr'] },
		{ primaryLanguage: 'cy', contentLanguages: ['fr'] }
	]);
	expect(requests.every((row) => row.owner === alice.did)).toBe(true);
	await page.reload();
	await expect(
		page.getByRole('combobox', { name: 'Primary language', exact: true })
	).toHaveValue('Welsh');
	await expect(
		page.getByRole('button', {
			name: 'Remove French from Content languages',
			exact: true
		})
	).toBeVisible();
	await expect(
		page.getByRole('button', {
			name: 'Remove English from Content languages',
			exact: true
		})
	).toHaveCount(0);
});
test('notification changes patch only the chosen category and field', async ({
	page
}) => {
	await page.goto('/notifications/settings');
	const row = page
		.locator('.notification-card')
		.filter({ has: page.getByRole('heading', { name: 'Likes', exact: true }) });
	await row.getByRole('combobox').click();
	await row
		.getByRole('option', { name: 'People I follow', exact: true })
		.click();
	await expect(row.getByRole('combobox')).toBeEnabled();
	await row
		.getByRole('switch', { name: 'Likes push notifications', exact: true })
		.uncheck();
	await expect(
		row.getByRole('switch', { name: 'Likes push notifications', exact: true })
	).toBeEnabled();
	const requests = (await calls()).filter(
		(row) =>
			row.method === 'PATCH' && row.path === '/v1/notifications/preferences'
	);
	expect(requests.map((row) => row.body)).toEqual([
		{ preferences: { like: { scope: 'peopleIFollow' } } },
		{ preferences: { like: { pushEnabled: false } } }
	]);
	await expect(
		page.getByRole('heading', { name: 'Future Category', exact: true })
	).toHaveCount(0);
	const follows = page.locator('.notification-card').filter({
		has: page.getByRole('heading', { name: 'Follows', exact: true })
	});
	await expect(
		follows.getByRole('switch', {
			name: 'Follows push notifications',
			exact: true
		})
	).toBeChecked();
	const moderation = page.locator('.notification-card').filter({
		has: page.getByRole('heading', { name: 'Moderation', exact: true })
	});
	await expect(moderation.getByRole('combobox')).toHaveCount(0);
	expect(requests.every((row) => row.owner === alice.did)).toBe(true);
});
test('business profile saves preserve products and unknown catalog values with a CID precondition', async ({
	page
}) => {
	await page.goto('/settings/business');
	await page
		.getByLabel('Tagline', { exact: true })
		.fill('Small batch colour for your next project');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Business profile saved.' })
	).toBeVisible();
	const request = (await calls()).find(
		(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
	);
	expect(request?.ifMatch).toBe('business-cid-0');
	expect(request?.key).toMatch(/^[0-9a-f-]{36}$/);
	expect(request?.body).toMatchObject({
		businessTypes: ['dyer'],
		offerings: ['yarn'],
		preserveProducts: true,
		preserveUnknownCatalogValues: true,
		tagline: 'Small batch colour for your next project'
	});
	expect(request?.body).not.toHaveProperty('products');
	await page.reload();
	await expect(page.getByLabel('Tagline', { exact: true })).toHaveValue(
		'Small batch colour for your next project'
	);
});
test('subscriptions belong to the active account without assignment or duplicate checkout', async ({
	page
}) => {
	await page.goto('/settings/subscriptions');
	await expect(
		page.getByRole('heading', { name: 'Current access: Business', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Manage', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Downgrade', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Assign license', exact: true })
	).toHaveCount(0);
	await expect(page.getByLabel('Account DID', { exact: true })).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: /^Choose (Plus|Business)$/ })
	).toHaveCount(0);
	await page.route('https://apps.apple.com/account/subscriptions', (route) =>
		route.fulfill({
			contentType: 'text/html',
			body: '<h1>App Store subscription management</h1>'
		})
	);
	await page.getByRole('button', { name: 'Manage', exact: true }).click();
	await expect(page).toHaveURL('https://apps.apple.com/account/subscriptions');
	await page.goto('/settings/subscriptions');
	await page.reload();
	await expect(
		page.getByRole('button', { name: 'Manage', exact: true })
	).toBeVisible();
	expect(
		(await calls()).some(
			(row) =>
				row.path.includes('/assignment') || row.path.endsWith('/web-checkout')
		)
	).toBe(false);
});
test('BZip2 ZIP imports submit usernames on selection after an empty export is rejected', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 900 });
	await page.goto('/settings/instagram');
	await verifyInstagram(page);
	await expect(page.getByText('No Instagram imports yet.')).toBeVisible();
	await expect(page.getByText('0 accounts imported')).toBeVisible();
	const picker = page.getByRole('button', {
		name: 'Select Instagram export',
		exact: true
	});
	const fileInput = page.locator('input[type="file"]');
	await fileInput.setInputFiles({
		name: 'following.json',
		mimeType: 'application/json',
		buffer: Buffer.from(JSON.stringify({ relationships_following: [] }))
	});
	await expect(
		page.getByText('No Instagram usernames were found in this export.', {
			exact: true
		})
	).toBeVisible();
	expect(
		(await calls()).filter(
			(row) => row.method === 'POST' && row.path.endsWith('/instagram/imports')
		)
	).toHaveLength(0);

	let finishImport!: () => void;
	const pendingImport = new Promise<void>((resolve) => {
		finishImport = resolve;
	});
	await page.route(
		'**/api/appview/v1/migrations/instagram/imports',
		async (route) => {
			if (route.request().method() === 'POST') await pendingImport;
			await route.continue();
		}
	);
	try {
		await fileInput.setInputFiles({
			name: 'instagram-bzip2.zip',
			mimeType: 'application/zip',
			buffer: Buffer.from(bzipFixture.zip, 'base64')
		});
		await expect(picker).toHaveAttribute('aria-busy', 'true');
		await expect(picker).toBeDisabled();
		await expect(
			page.getByRole('heading', { name: /^Review import:/ })
		).toHaveCount(0);
	} finally {
		finishImport();
	}
	await expect(
		page.getByRole('status').filter({ hasText: 'Instagram import created' })
	).toBeVisible();
	await expect(picker).toBeEnabled();
	await expect(
		page.getByRole('heading', { name: 'Instagram export', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('1 account imported', { exact: true })
	).toBeVisible();
	expect(
		(await calls())
			.filter(
				(row) =>
					row.method === 'POST' && row.path.endsWith('/instagram/imports')
			)
			.map((row) => row.body)
	).toEqual([
		{
			sourceType: 'instagramJson',
			entries: [{ username: 'maker.one' }]
		}
	]);
});
test('moderation history shows the affected content reference and its reviewed post destination', async ({
	page
}) => {
	await page.goto('/settings/moderation');
	await expect(
		page.getByText(
			'at://did:plc:alice/social.craftsky.feed.post/moderated-post',
			{ exact: true }
		)
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'View post', exact: true })
	).toHaveAttribute('href', '/posts/did%3Aplc%3Aalice/moderated-post');
	await expect(
		page.getByRole('link', { name: 'Appeal by email', exact: true })
	).toHaveAttribute('href', /^mailto:moderation@craftsky.social\?/);
});
test('Instagram verification, automatic export import and following a match complete through real controls', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 900 });
	await page.route(
		'**/api/appview/v1/migrations/instagram/imports',
		async (route) => {
			if (route.request().method() !== 'GET') return route.continue();
			const response = await route.fetch();
			const history = await response.json();
			await route.fulfill({
				response,
				json: {
					...history,
					items: [
						...history.items,
						{
							importId: 'legacy-manual-import',
							sourceType: 'manual',
							state: 'active',
							followingCount: 3,
							createdAt: '2026-11-01T12:00:00Z'
						}
					]
				}
			});
		}
	);
	await page.goto('/settings/instagram');
	await verifyInstagram(page);
	await expect(
		page.getByRole('button', { name: 'Enter handles', exact: true })
	).toHaveCount(0);
	const exportHelp = page.getByRole('link', {
		name: 'How to export your Instagram following',
		exact: true
	});
	const selectExport = page.getByRole('button', {
		name: 'Select Instagram export',
		exact: true
	});
	await expect(selectExport).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Manual handles', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('3 accounts imported', { exact: true })
	).toBeVisible();
	const importCard = page.getByRole('region', {
		name: 'Instagram following export'
	});
	await expect(importCard.getByRole('link')).toHaveCount(0);
	await expect(exportHelp).toHaveText('How to export');
	await expect(exportHelp).toHaveCSS('background-color', 'rgb(201, 184, 232)');
	const guideIcons = exportHelp.locator('svg path');
	await expect(guideIcons).toHaveCount(2);
	await expect(guideIcons.first()).toHaveAttribute(
		'd',
		nativeIconPaths.question.regular
	);
	await expect(guideIcons.last()).toHaveAttribute(
		'd',
		nativeIconPaths.arrowSquareOut.regular
	);
	const helpBounds = (await exportHelp.boundingBox())!;
	const cardBounds = (await importCard.boundingBox())!;
	expect(helpBounds.y + helpBounds.height).toBeLessThan(cardBounds.y);
	expect(helpBounds.width).toBeLessThan(cardBounds.width);
	expect(helpBounds.x).toBeCloseTo(cardBounds.x);
	const [chooser] = await Promise.all([
		page.waitForEvent('filechooser'),
		selectExport.click()
	]);
	expect(await chooser.element().getAttribute('aria-label')).toBe(
		'Select Instagram export'
	);
	const followingExport = {
		relationships_following: [
			{ string_list_data: [{ value: '@maker_one' }] },
			{
				title: 'Maker_Two',
				string_list_data: [{ href: 'https://www.instagram.com/_u/Maker_Two' }]
			},
			{ string_list_data: [{ value: 'maker_one' }] },
			{ string_list_data: [{ value: 'invalid name' }] }
		]
	};
	await chooser.setFiles({
		name: 'following.json',
		mimeType: 'application/json',
		buffer: Buffer.from(JSON.stringify(followingExport))
	});
	await expect(
		page.getByRole('status').filter({ hasText: 'Instagram import created' })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Your imports' })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Instagram export', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('2 accounts imported', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: /^Review import:/ })
	).toHaveCount(0);
	await page.getByRole('link', { name: 'Back to Instagram' }).click();
	await expect(page).toHaveURL(/\/settings\/instagram$/);
	const suggestions = page.getByRole('link', { name: /Suggested accounts/ });
	await expect(suggestions).toContainText('2');
	await suggestions.click();
	await expect(page).toHaveURL(/\/settings\/instagram\/suggestions$/);
	await expect(
		page.getByRole('heading', { name: 'Suggested accounts', exact: true })
	).toBeVisible();
	const matches = page.getByRole('region', { name: 'Suggested accounts' });
	const bob = matches.getByRole('link', { name: 'Bob Maker @bob.test' });
	const carol = matches.getByRole('link', { name: 'Carol Maker @carol.test' });
	const follow = matches.getByRole('button', { name: 'Follow', exact: true });
	const dismiss = matches.getByRole('button', { name: 'Dismiss', exact: true });
	await expect(bob).toHaveAttribute('href', '/profiles/did%3Aplc%3Abob');
	await bob.click();
	await expect(page).toHaveURL(/\/profiles\/did%3Aplc%3Abob$/);
	await page.goBack();
	await expect(bob).toBeVisible();
	await expect(carol).toBeVisible();
	await expect(dismiss.first()).toHaveCSS('border-top-width', '0px');
	const profileBounds = (await bob.boundingBox())!;
	const followBounds = (await follow.first().boundingBox())!;
	const dismissBounds = (await dismiss.first().boundingBox())!;
	expect(profileBounds.y + profileBounds.height).toBeLessThanOrEqual(
		followBounds.y
	);
	expect(followBounds.y).toBeCloseTo(dismissBounds.y);
	expect(followBounds.width).toBeGreaterThan(dismissBounds.width);

	let finishFollow!: () => void;
	const pendingFollow = new Promise<void>((resolve) => {
		finishFollow = resolve;
	});
	await page.route(
		'**/migrations/instagram/suggestions/suggestion-1/accept',
		async (route) => {
			await pendingFollow;
			await route.continue();
		}
	);
	try {
		await follow.first().click();
		await expect(follow.first()).toHaveAttribute('aria-busy', 'true');
	} finally {
		finishFollow();
	}
	await expect(
		page.getByRole('status').filter({ hasText: 'Following this maker.' })
	).toBeVisible();
	await expect(bob).toHaveCount(0);
	await expect(carol).toBeVisible();
	await dismiss.click();
	await expect(carol).toHaveCount(0);
	await expect(
		matches.getByText('No possible CraftSky accounts yet.', { exact: true })
	).toBeVisible();
	await expect(
		matches.getByText(
			'Import the accounts you follow on Instagram to look for matches.',
			{ exact: true }
		)
	).toBeVisible();
	await expect(matches.locator('svg path')).toHaveAttribute(
		'd',
		nativeIconPaths.userFocus.regular
	);
	await page.getByRole('link', { name: 'Back to Instagram' }).click();
	await page.getByRole('link', { name: 'Import accounts you follow' }).click();
	const importedRow = page.locator('.import-row').filter({
		has: page.getByRole('heading', { name: 'Instagram export', exact: true })
	});
	const deleteImport = importedRow.getByRole('button', {
		name: 'Delete import',
		exact: true
	});
	await expect(deleteImport).toHaveText('');
	await expect(deleteImport.locator('svg path')).toHaveAttribute(
		'd',
		nativeIconPaths.trash.bold!
	);
	await deleteImport.click();
	await expect(importedRow).toHaveCount(0);
	await expect(
		page.getByRole('dialog', { name: 'Delete import?' })
	).toHaveCount(0);
	const requests = await calls();
	expect(
		requests.find((row) => row.path.endsWith('/verify-1/confirm'))?.body
	).toEqual({ discoverable: true });
	expect(
		requests.find(
			(row) => row.method === 'POST' && row.path.endsWith('/instagram/imports')
		)?.body
	).toEqual({
		sourceType: 'instagramJson',
		entries: [{ username: 'maker_one' }, { username: 'maker_two' }]
	});
	expect(
		requests.find((row) => row.path.endsWith('/suggestion-1/accept'))?.key
	).toMatch(/^[0-9a-f-]{36}$/);
	expect(
		requests.find((row) => row.path.endsWith('/suggestion-2'))?.method
	).toBe('DELETE');
	expect(
		requests.find((row) => row.path.endsWith('/instagram/imports/import-1'))
			?.method
	).toBe('DELETE');
});
for (const width of [402, 1280]) {
	test(`event shared photo picker limits to one, keeps edits local and uploads once across retry (${width})`, async ({
		page
	}, testInfo) => {
		await page.setViewportSize({ width, height: 900 });
		let attempts = 0;
		await page.route('**/api/appview/v1/events', async (route) => {
			if (route.request().method() !== 'POST') return route.continue();
			if (++attempts === 1)
				return route.fulfill({
					status: 202,
					headers: { 'Retry-After': '1' },
					json: { status: 'ambiguous' }
				});
			return route.continue();
		});
		await page.goto('/compose/event');
		const input = page.getByLabel('Add images', { exact: true });
		await expect(input).not.toHaveAttribute('multiple');
		const png = await page.evaluate(() => {
			const canvas = document.createElement('canvas');
			canvas.width = 32;
			canvas.height = 48;
			canvas.getContext('2d')!.fillRect(0, 0, 32, 48);
			return canvas.toDataURL('image/png').split(',')[1];
		});
		const buffer = Buffer.from(png, 'base64');
		await page.locator('.photo-picker').evaluate((node, encoded) => {
			const data = new DataTransfer();
			const bytes = Uint8Array.from(atob(encoded), (c) => c.charCodeAt(0));
			data.items.add(new File([bytes], 'first.png', { type: 'image/png' }));
			data.items.add(new File([bytes], 'second.png', { type: 'image/png' }));
			node.dispatchEvent(
				new DragEvent('drop', {
					bubbles: true,
					cancelable: true,
					dataTransfer: data
				})
			);
		}, png);
		await expect(
			page.getByText('Choose one photo for an event.', { exact: true })
		).toBeVisible();
		await input.setInputFiles({
			name: 'event.png',
			mimeType: 'image/png',
			buffer
		});
		const details = page.getByRole('region', {
			name: 'Photo details',
			exact: true
		});
		await expect(details).toBeVisible();
		await expect(details).toContainText('1/1');
		await expect(details.getByLabel('Add images', { exact: true })).toHaveCount(
			0
		);
		await expect(
			details.getByRole('button', { name: /Move image|Drag to reorder/ })
		).toHaveCount(0);
		await details
			.getByRole('textbox', { name: 'Image description', exact: true })
			.fill('A portrait event poster');
		await details
			.getByRole('button', { name: 'Edit photo', exact: true })
			.click();
		const crop = page.getByRole('dialog', { name: 'Edit photo', exact: true });
		await crop
			.getByRole('button', { name: 'Rotate photo', exact: true })
			.click();
		await crop.getByRole('button', { name: 'Apply edit', exact: true }).click();
		await expect(crop).not.toBeVisible();
		await details.screenshot({
			path: testInfo.outputPath(`event-photo-details-${width}.png`)
		});
		const audit = await new AxeBuilder({ page })
			.include('.photo-details-page')
			.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
			.analyze();
		expect(audit.violations).toEqual([]);
		await page.goBack();
		await expect(details).toBeHidden();
		await expect(page).toHaveURL(/\/compose\/event$/);
		await expect(
			page.getByRole('dialog', { name: 'Discard changes?', exact: true })
		).toBeHidden();
		await page
			.getByRole('button', { name: /Photo.*Edit photo, remove it/ })
			.click();
		await expect(
			details.getByRole('textbox', { name: 'Image description', exact: true })
		).toHaveValue('A portrait event poster');
		await details
			.getByRole('button', { name: 'Back to event', exact: true })
			.click();
		expect(
			(await calls()).filter((row) => row.path === '/v1/blobs/images')
		).toHaveLength(0);
		await page
			.getByLabel('Event name', { exact: true })
			.fill('Yarn fair with photo');
		await page.getByLabel('Start', { exact: true }).fill('2026-12-01T10:00');
		await page.getByLabel('End', { exact: true }).fill('2026-12-01T12:00');
		await page.getByLabel('Your role', { exact: true }).click();
		await page
			.getByRole('checkbox', { name: 'Organizer', exact: true })
			.click();
		await page
			.getByRole('button', { name: 'Create event', exact: true })
			.click();
		await expect(
			page.getByRole('button', { name: 'Check and retry action', exact: true })
		).toBeVisible();
		await page
			.getByRole('button', { name: 'Check and retry action', exact: true })
			.click();
		await expect(page).toHaveURL(/\/settings\/events$/);
		const requests = await calls();
		expect(
			requests.filter((row) => row.path === '/v1/blobs/images')
		).toHaveLength(1);
		const saved = requests.find(
			(row) => row.method === 'POST' && row.path === '/v1/events'
		);
		expect(saved?.body.image).toMatchObject({
			alt: 'A portrait event poster',
			aspectRatio: { width: 48, height: 32 }
		});
	});
}

for (const remove of [false, true]) {
	test(`event shared photo details ${remove ? 'removes' : 'describes'} an existing image without uploading`, async ({
		page
	}) => {
		const event = {
			did: alice.did,
			rkey: 'existing-photo',
			cid: 'event-photo-cid',
			uri: `at://${alice.did}/social.craftsky.business.event/existing-photo`,
			name: 'Maker fair',
			startsAt: '2026-12-01T10:00:00Z',
			endsAt: '2026-12-01T12:00:00Z',
			timeZone: 'UTC',
			roles: [{ value: 'organizer', known: true }],
			mode: { value: 'in-person', known: true },
			status: { value: 'scheduled', known: true },
			image: {
				cid: 'existing-image-cid',
				mime: 'image/jpeg',
				size: 100,
				alt: 'Original poster',
				thumb: 'https://fixture.craftsky.test/event.png'
			}
		};
		let body: Record<string, unknown> | undefined;
		await page.route('https://fixture.craftsky.test/event.png', (route) =>
			route.fulfill({ path: 'static/app-icon.png', contentType: 'image/png' })
		);
		await page.route(
			`**/api/appview/v1/events/${encodeURIComponent(alice.did)}/existing-photo`,
			async (route) => {
				if (route.request().method() === 'GET')
					return route.fulfill({ json: event });
				body = route.request().postDataJSON();
				return route.fulfill({
					json: {
						cid: 'next-event-cid',
						uri: event.uri,
						rkey: event.rkey,
						indexing: 'pending'
					}
				});
			}
		);
		await page.goto('/compose/event/edit?event=existing-photo');
		await page
			.getByRole('button', { name: /Photo.*Edit photo, remove it/ })
			.click();
		const details = page.getByRole('region', {
			name: 'Photo details',
			exact: true
		});
		if (remove) {
			await details
				.getByRole('button', { name: 'Remove image 1', exact: true })
				.click();
			await expect(
				details.getByLabel('Add images', { exact: true })
			).toBeAttached();
		} else
			await details
				.getByRole('textbox', { name: 'Image description', exact: true })
				.fill('Updated poster description');
		await details
			.getByRole('button', { name: 'Back to event', exact: true })
			.click();
		await page.getByRole('button', { name: 'Save event', exact: true }).click();
		await expect(page).toHaveURL(/\/settings\/events$/);
		expect(
			(await calls()).filter((row) => row.path === '/v1/blobs/images')
		).toHaveLength(0);
		if (remove) expect(body).not.toHaveProperty('image');
		else
			expect(body?.image).toMatchObject({
				alt: 'Updated poster description',
				image: { ref: { $link: 'existing-image-cid' } }
			});
	});
}

test('event composer hints and roles validate only after submission', async ({
	page
}, testInfo) => {
	await page.goto('/settings/events');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	const dialog = page.locator('.event-composer-form');
	for (const [label, hint] of [
		['Event name', 'Autumn yarn fair'],
		[
			'Summary',
			'Meet local makers, browse handmade goods and join a workshop.'
		],
		['Venue', 'Bristol Craft Hall'],
		['Event link', 'https://your-business.com/events/autumn-fair'],
		['Registration link', 'https://your-business.com/events/autumn-fair/book'],
		['Start', 'Select date and time'],
		['End', 'Select date and time']
	])
		await expect(dialog.getByLabel(label, { exact: true })).toHaveAttribute(
			'placeholder',
			hint
		);
	await expect(dialog.locator('[aria-invalid="true"]')).toHaveCount(0);
	const audit = await new AxeBuilder({ page })
		.include('.event-composer-form')
		.analyze();
	expect(audit.violations).toEqual([]);
	await dialog.screenshot({
		path: testInfo.outputPath('event-composer-desktop.png')
	});
	await dialog
		.getByLabel('Event name', { exact: true })
		.fill('Autumn yarn fair');
	await dialog.getByLabel('Start', { exact: true }).fill('2026-12-01T10:00');
	await dialog.getByLabel('End', { exact: true }).fill('2026-12-01T12:00');
	await dialog
		.getByRole('button', { name: 'Create event', exact: true })
		.click();
	await expect(dialog.getByLabel('Your role', { exact: true })).toHaveAttribute(
		'aria-invalid',
		'true'
	);
	await expect(dialog.locator('.field-error')).toContainText(
		'This field is required.'
	);
	expect(
		(await calls()).filter(
			(row) => row.method === 'POST' && row.path === '/v1/events'
		)
	).toHaveLength(0);
	await dialog.getByLabel('Your role', { exact: true }).click();
	await dialog
		.getByRole('checkbox', { name: 'Organizer', exact: true })
		.click();
	await expect(
		dialog.getByLabel('Your role', { exact: true })
	).not.toHaveAttribute('aria-invalid', 'true');
	await expect(dialog.locator('.field-error')).toHaveCount(0);
	await dialog
		.getByRole('switch', { name: 'All-day event', exact: true })
		.check();
	await expect(dialog.getByLabel('Start', { exact: true })).toHaveAttribute(
		'type',
		'date'
	);
	await expect(dialog.getByLabel('Start', { exact: true })).toHaveValue(
		'2026-12-01'
	);
	await expect(dialog.getByLabel('End', { exact: true })).toHaveValue(
		'2026-12-02'
	);
	await expect(
		dialog.getByText(
			'End is the first date after the event, in the selected timezone.'
		)
	).toBeVisible();
});

test('business events create, edit using the accepted CID, and delete', async ({
	page
}) => {
	await page.goto('/settings/events');
	await page.getByRole('button', { name: 'History', exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'History', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	await expect(
		page.getByText('No event history yet.', { exact: false })
	).toBeVisible();
	await page.getByRole('button', { name: 'Upcoming', exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Upcoming', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await expect(page.locator('.event-composer-form')).toBeVisible();
	await page
		.getByLabel('Event name', { exact: true })
		.fill('Indigo dye workshop');
	await page
		.getByRole('combobox', { name: 'Timezone', exact: true })
		.fill('UTC');
	await page.getByRole('option', { name: 'UTC', exact: true }).click();
	await page.getByLabel('Start', { exact: true }).fill('2026-12-01T10:00');
	await page.getByLabel('End', { exact: true }).fill('2026-12-01T12:00');
	await page.getByLabel('Your role', { exact: true }).click();
	await page.getByRole('checkbox', { name: 'Organizer', exact: true }).click();
	await page
		.locator('.event-composer-form')
		.getByRole('button', { name: 'Create event', exact: true })
		.click();
	await expect(
		page.getByRole('heading', { name: 'Indigo dye workshop', exact: true })
	).toBeVisible();
	await page
		.getByLabel('Actions for Indigo dye workshop', { exact: true })
		.click();
	await page.getByRole('button', { name: 'Edit', exact: true }).click();
	await expect(page).toHaveURL(/\/compose\/event\/edit\?event=event-1$/);
	await page.reload();
	await page
		.getByLabel('Event name', { exact: true })
		.fill('Natural indigo workshop');
	await page.getByRole('button', { name: 'Save event' }).click();
	await expect(
		page.getByRole('heading', { name: 'Natural indigo workshop', exact: true })
	).toBeVisible();
	await page
		.getByLabel('Actions for Natural indigo workshop', { exact: true })
		.click();
	await page.getByRole('button', { name: 'Delete', exact: true }).click();
	await page
		.getByRole('dialog', { name: 'Delete event?' })
		.getByRole('button', { name: 'Delete', exact: true })
		.click();
	await expect(
		page.getByText('No upcoming events yet.', { exact: false })
	).toBeVisible();
	const requests = await calls();
	const create = requests.find(
		(row) => row.method === 'POST' && row.path === '/v1/events'
	);
	expect(create?.owner).toBe(alice.did);
	expect(create?.key).toMatch(/^[0-9a-f-]{36}$/);
	expect(create?.body).toMatchObject({
		name: 'Indigo dye workshop',
		startsAt: '2026-12-01T10:00:00Z',
		endsAt: '2026-12-01T12:00:00Z',
		timeZone: 'UTC',
		roles: ['organizer']
	});
	expect(
		requests.find(
			(row) => row.method === 'PUT' && row.path.endsWith('/event-1')
		)?.ifMatch
	).toBe('event-cid-1');
	expect(
		requests.find(
			(row) => row.method === 'DELETE' && row.path.endsWith('/event-1')
		)?.ifMatch
	).toBe('event-cid-1-next');
});

test('compact event editor has reachable save and protects an unfinished draft', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/settings/events');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await expect(page).toHaveURL(/\/compose\/event$/);
	const dialog = page.getByRole('form', {
		name: 'Create event',
		exact: true
	});
	await expect(
		dialog.getByRole('searchbox', { name: 'Your role', exact: true })
	).toHaveValue('');
	await expect(
		dialog.getByRole('combobox', { name: 'Timezone', exact: true })
	).toHaveValue('UTC');
	await dialog
		.getByLabel('Event name', { exact: true })
		.fill('Unfinished workshop');
	const bounds = await dialog
		.getByRole('button', { name: 'Create event', exact: true })
		.evaluate((node) => ({
			top: node.getBoundingClientRect().top,
			bottom: node.getBoundingClientRect().bottom,
			height: innerHeight
		}));
	expect(bounds.top).toBeGreaterThanOrEqual(0);
	expect(bounds.bottom).toBeLessThanOrEqual(bounds.height);
	await page
		.getByRole('button', { name: 'Close event editor', exact: true })
		.click();
	await page.getByRole('button', { name: 'Keep editing', exact: true }).click();
	await expect(dialog.getByLabel('Event name', { exact: true })).toHaveValue(
		'Unfinished workshop'
	);
	await page
		.getByRole('button', { name: 'Close event editor', exact: true })
		.click();
	await page.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await expect(page).toHaveURL(/\/settings\/events$/);
	expect(
		(await calls()).filter(
			(row) => row.path === '/v1/events' && row.method === 'POST'
		)
	).toHaveLength(0);
});

test('event composer routes load directly, survive refresh and protect browser back', async ({
	page
}) => {
	await page.goto('/settings/events');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await expect(page).toHaveURL(/\/compose\/event$/);
	await page.reload();
	await expect(
		page.getByRole('heading', { name: 'Create event', exact: true })
	).toBeVisible();
	await expect(page.locator('dialog[open]')).toHaveCount(0);
	await page
		.getByLabel('Event name', { exact: true })
		.fill('Unfinished routed event');
	await page.goBack();
	await page.getByRole('button', { name: 'Keep editing', exact: true }).click();
	await expect(page).toHaveURL(/\/compose\/event$/);
	await expect(page.getByLabel('Event name', { exact: true })).toHaveValue(
		'Unfinished routed event'
	);
	await page.goBack();
	await page.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/events$/);
	await page.goto('/compose/event/edit?event=missing-event');
	await expect(
		page.getByRole('heading', { name: 'Edit event', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Try again', exact: true })
	).toBeVisible();
	await expect(page.locator('.event-composer-form')).toHaveCount(0);
});

test('customisation changes preview choices and saves the exact native colour and background values', async ({
	page
}) => {
	await page.goto('/settings/customisation');
	const save = page.getByRole('button', { name: 'Save', exact: true });
	await expect(save).toBeDisabled();
	for (const name of [
		'Dither',
		'Grid',
		'Cross stitch',
		'Scallops',
		'Diagonal weave',
		'Crosshatch'
	])
		await expect(page.getByRole('radio', { name, exact: true })).toHaveCount(0);
	for (const name of [
		'Spinning',
		'Dyeing',
		'Punch Needle',
		'Macramé',
		'Weaving',
		'Felting'
	])
		await expect(page.getByRole('radio', { name, exact: true })).toBeVisible();
	await page.getByText('Green', { exact: true }).click();
	const tileResponse = page.waitForResponse(
		'**/profile_backgrounds/craft-punch-needle.svg'
	);
	await page.getByText('Punch Needle', { exact: true }).click();
	const tile = await tileResponse;
	expect(tile.ok()).toBe(true);
	expect(tile.headers()['content-type']).toContain('image/svg+xml');
	await expect(
		page.getByRole('radio', { name: 'Green', exact: true })
	).toBeChecked();
	await expect(
		page.getByRole('radio', { name: 'Punch Needle', exact: true })
	).toBeChecked();
	await expect(
		page.getByLabel('Profile customisation preview')
	).toHaveAttribute('style', /--chosen-colour:\s*#23770f/i);
	await expect(page.locator('.preview-pattern')).toHaveCSS(
		'mask-image',
		/\/profile_backgrounds\/craft-punch-needle\.svg/
	);
	await expect(save).toBeEnabled();
	expect(
		(await calls()).filter(
			(row) => row.method === 'PUT' && row.path.endsWith('/customisation')
		)
	).toHaveLength(0);
	await save.click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Profile customisation saved.' })
	).toBeVisible();
	const request = (await calls()).find(
		(row) => row.method === 'PUT' && row.path.endsWith('/customisation')
	);
	expect(request?.owner).toBe(alice.did);
	expect(request?.body).toEqual({
		colour: 'lime',
		profileBackground: 'craft-punch-needle'
	});
	await page.reload();
	await expect(
		page.getByRole('radio', { name: 'Green', exact: true })
	).toBeChecked();
	await expect(
		page.getByRole('radio', { name: 'Punch Needle', exact: true })
	).toBeChecked();
	await expect(
		page.getByRole('button', { name: 'Save', exact: true })
	).toBeDisabled();
	await page.goto('/profiles/alice.test');
	await expect(page.locator('.profile-texture')).toHaveCSS(
		'mask-image',
		/\/profile_backgrounds\/craft-punch-needle\.svg/
	);
});

for (const viewport of [
	{ width: 1280, height: 480 },
	{ width: 375, height: 640 }
]) {
	test(`customisation preview stays pinned while choosing backgrounds at ${viewport.width}px`, async ({
		page
	}) => {
		await page.setViewportSize(viewport);
		await page.goto('/settings/customisation');
		const preview = page.getByLabel('Profile customisation preview');
		await expect(preview).toBeVisible();
		await page
			.getByRole('button', { name: 'Save', exact: true })
			.scrollIntoViewIfNeeded();
		expect(await page.evaluate(() => window.scrollY)).toBeGreaterThan(0);
		const pinned = await preview.boundingBox();
		const header = await page.locator('.workspace-heading').boundingBox();
		expect(pinned!.y).toBeCloseTo(header!.y + header!.height, 0);
		await page.getByText('Embroidery', { exact: true }).click();
		await expect(
			page.getByRole('radio', { name: 'Embroidery', exact: true })
		).toBeChecked();
		await expect(preview.locator('.preview-pattern')).toHaveAttribute(
			'style',
			/craft-embroidery/
		);
		const updated = await preview.boundingBox();
		expect(updated!.y).toBeGreaterThanOrEqual(header!.y + header!.height - 1);
		expect(updated!.y + updated!.height).toBeLessThanOrEqual(viewport.height);
	});
}

test('product route cancellation restores existing details and removes an unsaved new product without writing', async ({
	page
}) => {
	await page.goto('/settings/products');
	await page
		.getByRole('button', { name: 'Edit Blue wool', exact: true })
		.click();
	let dialog = page.locator('.product-composer-form');
	await expect(dialog).toBeVisible();
	await dialog.getByLabel('Title', { exact: true }).fill('Unsaved wool');
	await dialog.getByLabel('Amount', { exact: true }).fill('99');
	await page
		.getByRole('button', { name: 'Close product editor', exact: true })
		.click();
	await page.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await page
		.getByRole('button', { name: 'Edit Blue wool', exact: true })
		.click();
	dialog = page.locator('.product-composer-form');
	await expect(dialog.getByLabel('Title', { exact: true })).toHaveValue(
		'Blue wool'
	);
	await expect(dialog.getByLabel('Amount', { exact: true })).toHaveValue(
		'12.5'
	);
	await page
		.getByRole('button', { name: 'Close product editor', exact: true })
		.click();
	await page.getByRole('button', { name: 'Add product', exact: true }).click();
	await expect(dialog).toBeVisible();
	await dialog.getByLabel('Title', { exact: true }).fill('Unsaved new product');
	await page
		.getByRole('button', { name: 'Close product editor', exact: true })
		.click();
	await page.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(
		page.getByText('1 of 4 products', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Edit Unsaved new product', exact: true })
	).toHaveCount(0);
	expect(
		(await calls()).filter(
			(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
		)
	).toHaveLength(0);
});

test('Business details is a dedicated settings page and personal profile editing excludes it', async ({
	page
}) => {
	await page.goto('/settings');
	await page
		.getByRole('link', { name: 'Business details', exact: true })
		.click();
	await expect(
		page.getByRole('heading', { name: 'Business details', exact: true })
	).toBeVisible();
	await expect(page.getByLabel('Tagline', { exact: true })).toHaveValue(
		'Original tagline'
	);
	await expect(
		page.getByRole('searchbox', { name: 'Business types', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Save', exact: true })
	).toBeDisabled();
	await page.goto('/profile/edit');
	await expect(page.getByLabel('Tagline', { exact: true })).toHaveCount(0);
	await expect(
		page.getByRole('searchbox', { name: 'Business types', exact: true })
	).toHaveCount(0);
});

for (const width of [1280, 390])
	test(`Business details matches native form layout and text limits at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 844 });
		await page.goto('/settings/business');
		const form = page.locator('.business-details-form');
		await expect(form).not.toHaveClass(/panel/);
		await expect(form).toHaveCSS('border-top-style', 'none');
		await expect(
			page.getByText('Choose up to 5.', { exact: true })
		).toBeVisible();
		await expect(
			page.getByText('Choose up to 10.', { exact: true })
		).toBeVisible();
		const save = page.getByRole('button', { name: 'Save', exact: true });
		await expect(save).toBeDisabled();
		await expect(
			page.getByRole('button', { name: 'Save business details', exact: true })
		).toHaveCount(0);
		for (const [label, maximum] of [
			['Tagline', 100],
			['Hours', 300],
			['Service area', 200],
			['Town or locality', 100]
		] as const) {
			const field = page.getByLabel(label, { exact: true });
			await field.fill('x'.repeat(maximum + 1));
			await expect(field).toHaveValue('x'.repeat(maximum));
			await expect(field.locator('..').locator('.field-count')).toHaveText(
				`${maximum}/${maximum}`
			);
		}
		const tagline = page.getByLabel('Tagline', { exact: true });
		await tagline.fill('🧶'.repeat(101));
		await expect(tagline).toHaveValue('🧶'.repeat(100));
		await tagline.fill('e\u0301'.repeat(101));
		await expect(tagline).toHaveValue('e\u0301'.repeat(100));
		await expect(save).toBeEnabled();
		const position = await save.evaluate((node) => {
			const button = node.getBoundingClientRect();
			const header = node.closest('header')!.getBoundingClientRect();
			return {
				top: button.top,
				bottom: button.bottom,
				right: button.right,
				headerBottom: header.bottom,
				headerRight: header.right
			};
		});
		expect(position.top).toBeGreaterThanOrEqual(0);
		expect(position.bottom).toBeLessThanOrEqual(position.headerBottom);
		expect(position.headerRight - position.right).toBeLessThan(30);
		expect(
			(await calls()).filter(
				(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
			)
		).toHaveLength(0);
	});

async function choosePrimaryAction(page: Page, name: string) {
	await page
		.getByRole('combobox', { name: 'Primary action', exact: true })
		.fill(name);
	await page.getByRole('option', { name, exact: true }).click();
}

for (const width of [1280, 390])
	test(`primary action destination is conditional and cleared at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/settings/business');
		const destination = page.locator('input[name="destination"]');
		const save = page.getByRole('button', { name: 'Save', exact: true });
		await expect(destination).toHaveCount(0);
		await choosePrimaryAction(page, 'Shop');
		await expect(
			page.getByRole('button', { name: 'Clear Primary action', exact: true })
		).toHaveCount(0);
		await expect(page.getByLabel('Shop URL', { exact: true })).toHaveAttribute(
			'placeholder',
			'https://example.com/shop'
		);
		await destination.fill('invalid URL');
		await choosePrimaryAction(page, 'No primary action');
		await expect(destination).toHaveCount(0);
		await expect(save).toBeDisabled();
		await choosePrimaryAction(page, 'Book class');
		await expect(page.getByLabel('Booking URL', { exact: true })).toHaveValue(
			''
		);
		await destination.fill('https://example.com/book');
		await save.click();
		await expect(
			page.getByRole('status').filter({ hasText: 'Business profile saved.' })
		).toBeVisible();
		await page.reload();
		await expect(page.getByLabel('Booking URL', { exact: true })).toHaveValue(
			'https://example.com/book'
		);
		await choosePrimaryAction(page, 'No primary action');
		await save.click();
		await expect(save).toBeDisabled();
		const writes = (await calls()).filter(
			(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
		);
		expect(writes[0].body.primaryAction).toEqual({
			type: 'book-class',
			destination: 'https://example.com/book'
		});
		expect(writes[1].body).not.toHaveProperty('primaryAction');
		expect(writes[1].body.preserveProducts).toBe(true);
		await page.reload();
		await expect(destination).toHaveCount(0);
	});

test('email action saves and reopens a plain email address', async ({
	page
}) => {
	await page.goto('/settings/business');
	await choosePrimaryAction(page, 'Email');
	await page
		.getByLabel('Email address', { exact: true })
		.fill('hello@example.com');
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Business profile saved.' })
	).toBeVisible();
	const write = (await calls()).find(
		(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
	);
	expect(write?.body.primaryAction).toEqual({
		type: 'email',
		destination: 'mailto:hello@example.com'
	});
	await page.reload();
	await expect(page.getByLabel('Email address', { exact: true })).toHaveValue(
		'hello@example.com'
	);
});

test('product editor enforces native title limits and points incomplete prices to their fields', async ({
	page
}) => {
	await page.goto('/compose/product/edit?product=0');
	const form = page.locator('.product-composer-form');
	const title = form.getByLabel('Title', { exact: true });
	await title.fill('🧶'.repeat(151));
	await expect(title).toHaveValue('🧶'.repeat(150));
	await expect(title.locator('..').locator('.field-count')).toHaveText(
		'150/150'
	);
	await title.fill('Blue wool');
	const amount = form.getByLabel('Amount', { exact: true });
	await amount.fill('12.34567');
	await form.getByRole('button', { name: 'Save product', exact: true }).click();
	await expect(amount).toHaveAttribute('aria-invalid', 'true');
	await expect(form.locator('.field-error')).toHaveText(
		'Use a valid decimal price and three-letter currency code.'
	);
	await amount.fill('12.50');
	await form
		.getByRole('button', { name: 'Clear Currency', exact: true })
		.click();
	await form.getByRole('button', { name: 'Save product', exact: true }).click();
	await expect(
		form.getByRole('combobox', { name: 'Currency', exact: true })
	).toHaveAttribute('aria-invalid', 'true');
	await expect(form.locator('.field-error')).toHaveText(
		'Enter both the price and currency, or leave both blank.'
	);
	expect(
		(await calls()).filter(
			(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
		)
	).toHaveLength(0);
});

test('event link validation is shown beside the field without losing the draft', async ({
	page
}) => {
	await page.goto('/compose/event');
	await page.getByLabel('Event name', { exact: true }).fill('Maker fair');
	await page.getByLabel('Start', { exact: true }).fill('2026-12-01T10:00');
	await page.getByLabel('End', { exact: true }).fill('2026-12-01T12:00');
	await page.getByLabel('Your role', { exact: true }).click();
	await page.getByRole('checkbox', { name: 'Organizer', exact: true }).click();
	const link = page.getByLabel('Event link', { exact: true });
	await link.fill('http://example.com/event');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await expect(link).toHaveAttribute('aria-invalid', 'true');
	await expect(page.locator('.field-error')).toHaveText(
		'Use secure HTTPS event and registration links.'
	);
	await expect(page.getByLabel('Event name', { exact: true })).toHaveValue(
		'Maker fair'
	);
	expect(
		(await calls()).filter(
			(row) => row.method === 'POST' && row.path === '/v1/events'
		)
	).toHaveLength(0);
	await link.fill('  https://example.com/event  ');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/events$/);
	expect(
		(await calls()).find(
			(row) => row.method === 'POST' && row.path === '/v1/events'
		)?.body.eventUri
	).toBe('https://example.com/event');
});

async function addProductPhoto(page: Page) {
	const png = await page.evaluate(() => {
		const canvas = document.createElement('canvas');
		canvas.width = 32;
		canvas.height = 32;
		canvas.getContext('2d')!.fillRect(0, 0, 32, 32);
		return canvas.toDataURL('image/png').split(',')[1];
	});
	await page.getByLabel('Add images', { exact: true }).setInputFiles({
		name: 'product.png',
		mimeType: 'image/png',
		buffer: Buffer.from(png, 'base64')
	});
	await page
		.locator('.photo-details-page')
		.getByRole('button', { name: 'Back to product', exact: true })
		.click();
}

for (const width of [1280, 390])
	test(`Business product can be created without a declaration at ${width}px`, async ({
		page
	}, testInfo) => {
		await page.setViewportSize({ width, height: 900 });
		await fetch('http://127.0.0.1:19847/__test/control', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ owners: { alice: { business: null } } })
		});
		await page.goto('/settings/products');
		await page
			.getByRole('button', { name: 'Add product', exact: true })
			.click();
		const dialog = page.getByRole('form', {
			name: 'Add product',
			exact: true
		});
		await expect(page).toHaveURL(/\/compose\/product$/);
		await expect(page.locator('dialog[open]')).toHaveCount(0);
		await expect(
			page.getByRole('heading', { name: 'Add product', exact: true })
		).toBeVisible();
		await expect(
			dialog.getByRole('button', { name: 'Save product', exact: true })
		).toBeInViewport();
		const audit = await new AxeBuilder({ page })
			.include('.product-composer')
			.analyze();
		expect(audit.violations).toEqual([]);
		await page.locator('.product-composer').screenshot({
			path: testInfo.outputPath(`product-composer-${width}.png`)
		});
		await dialog.getByLabel('Title', { exact: true }).fill('New indigo wool');
		await dialog
			.getByLabel('Destination', { exact: true })
			.fill('https://shop.example/new-wool');
		await addProductPhoto(page);
		await dialog
			.getByRole('button', { name: 'Save product', exact: true })
			.click();
		await expect(dialog).not.toBeVisible();
		await page
			.getByRole('button', { name: 'Edit New indigo wool', exact: true })
			.click();
		await expect(page.getByLabel('Title', { exact: true })).toHaveValue(
			'New indigo wool'
		);
		const write = (await calls()).find(
			(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
		);
		expect(write?.ifMatch).toBe('*');
		expect(write?.body.products).toEqual([
			expect.objectContaining({
				title: 'New indigo wool',
				uri: 'https://shop.example/new-wool'
			})
		]);
		expect(
			(write?.body.products as Record<string, unknown>[])[0]
		).not.toHaveProperty('price');
		const bounds = await page
			.locator('.product-composer-form')
			.evaluate((node) => ({
				left: node.getBoundingClientRect().left,
				right: node.getBoundingClientRect().right,
				viewport: innerWidth
			}));
		expect(bounds.left).toBeGreaterThanOrEqual(0);
		expect(bounds.right).toBeLessThanOrEqual(bounds.viewport);
	});

test('product manager reorders and confirms removal outside the editor', async ({
	page
}) => {
	await page.goto('/settings/products');
	await page.getByRole('button', { name: 'Add product', exact: true }).click();
	await page.getByLabel('Title', { exact: true }).fill('Second product');
	await page
		.getByLabel('Destination', { exact: true })
		.fill('https://shop.example/second');
	await addProductPhoto(page);
	await page.getByRole('button', { name: 'Save product', exact: true }).click();
	await page
		.getByLabel('Product actions for Second product', { exact: true })
		.click();
	await page.getByRole('button', { name: 'Move up', exact: true }).click();
	await expect(page.locator('.product-summary').first()).toHaveAttribute(
		'aria-label',
		'Edit Second product'
	);
	await expect(
		page.getByRole('status').filter({ hasText: 'Products saved.' })
	).toBeVisible();
	await page
		.getByLabel('Product actions for Second product', { exact: true })
		.click();
	await page
		.getByRole('button', { name: 'Remove product', exact: true })
		.click();
	await page
		.getByRole('dialog', { name: 'Remove product?', exact: true })
		.getByRole('button', { name: 'Remove', exact: true })
		.click();
	await expect(
		page.getByRole('button', { name: 'Edit Second product', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Edit Blue wool', exact: true })
	).toBeVisible();
	const writes = (await calls()).filter(
		(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
	);
	expect(
		(writes[1].body.products as Record<string, unknown>[]).map((p) => p.title)
	).toEqual(['Second product', 'Blue wool']);
	expect(
		(writes[2].body.products as Record<string, unknown>[]).map((p) => p.title)
	).toEqual(['Blue wool']);
});

test('About displays the embedded build version alongside native legal rows', async ({
	page
}) => {
	await page.goto('/settings/about');
	const version = readFileSync(
		new URL('../VERSION', import.meta.url),
		'utf8'
	).trim();
	const about = page.getByRole('navigation', {
		name: 'About and legal',
		exact: true
	});
	await expect(
		about.getByRole('link', { name: 'Terms', exact: true })
	).toHaveAttribute('href', 'https://craftsky.social/terms');
	await expect(
		about.getByRole('link', { name: 'Privacy policy', exact: true })
	).toHaveAttribute('href', 'https://craftsky.social/privacy');
	await expect(about.getByText('Version', { exact: true })).toBeVisible();
	await expect(about.getByText(version, { exact: true })).toBeVisible();
});

test('event detail owner actions edit and confirm deletion using the current CID', async ({
	page
}) => {
	const created = await fetch('http://127.0.0.1:19847/v1/events', {
		method: 'POST',
		headers: {
			authorization: 'Bearer ' + alice.token,
			'content-type': 'application/json',
			'idempotency-key': '00000000-0000-4000-8000-000000000099'
		},
		body: JSON.stringify({
			name: 'Creator event',
			startsAt: '2026-12-01T10:00:00Z',
			endsAt: '2026-12-01T12:00:00Z',
			timeZone: 'UTC',
			roles: ['organizer', 'instructor'],
			status: 'scheduled',
			mode: 'in-person',
			isAllDay: false,
			createdAt: '2026-11-15T10:00:00Z',
			past: false,
			publicSuppressionReasons: [],
			upcomingExclusionReasons: []
		})
	});
	expect(created.status).toBe(201);
	const detail = '/events/did%3Aplc%3Aalice/event-1';
	await page.goto(detail);
	await expect(
		page.getByRole('link', { name: 'View creator profile', exact: true })
	).toHaveAttribute('href', '/profiles/did%3Aplc%3Aalice');
	await page.getByLabel('Event actions', { exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Report event', exact: true })
	).toHaveCount(0);
	await page.getByRole('link', { name: 'Edit event', exact: true }).click();
	await expect(page).toHaveURL(/compose\/event\/edit\?event=event-1$/);
	await page
		.getByLabel('Event name', { exact: true })
		.fill('Updated creator event');
	await page.getByRole('button', { name: 'Save event', exact: true }).click();
	await expect(page).toHaveURL(/settings\/events$/);
	await page.goto(detail);
	await expect(
		page.getByRole('heading', { name: 'Updated creator event', exact: true })
	).toBeVisible();
	await page.getByLabel('Event actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Delete event', exact: true }).click();
	await page
		.getByRole('dialog', { name: 'Delete this event?', exact: true })
		.getByRole('button', { name: 'Cancel', exact: true })
		.click();
	expect(
		(await calls()).filter(
			(row) => row.method === 'DELETE' && row.path.endsWith('/event-1')
		)
	).toHaveLength(0);
	await page.getByLabel('Event actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Delete event', exact: true }).click();
	await page
		.getByRole('dialog', { name: 'Delete this event?', exact: true })
		.getByRole('button', { name: 'Delete', exact: true })
		.click();
	await expect(page).toHaveURL(/profiles\/did%3Aplc%3Aalice\?tab=events$/);
	expect(
		(await calls()).find(
			(row) => row.method === 'DELETE' && row.path.endsWith('/event-1')
		)
	).toMatchObject({
		ifMatch: 'event-cid-1-next',
		key: expect.stringMatching(/^[0-9a-f-]{36}$/)
	});
});

test('event owner can postpone from management and cancel from detail without losing record fields', async ({
	page
}) => {
	const image = {
		image: {
			$type: 'blob',
			ref: { $link: 'existing-photo-cid' },
			mimeType: 'image/jpeg',
			size: 100
		},
		alt: 'Event poster',
		aspectRatio: { width: 400, height: 300 }
	};
	const body = {
		name: 'Lifecycle workshop',
		startsAt: '2027-12-01T10:00:00Z',
		endsAt: '2027-12-01T12:00:00Z',
		timeZone: 'Europe/London',
		roles: ['organizer', 'instructor'],
		status: 'scheduled',
		mode: 'hybrid',
		isAllDay: false,
		summary: 'Keep this summary',
		venueName: 'Craft Hall',
		eventUri: 'https://example.com/event',
		registrationUri: 'https://example.com/book'
	};
	const created = await fetch('http://127.0.0.1:19847/v1/events', {
		method: 'POST',
		headers: {
			authorization: 'Bearer ' + alice.token,
			'content-type': 'application/json',
			'idempotency-key': '00000000-0000-4000-8000-000000000099'
		},
		body: JSON.stringify(body)
	});
	expect(created.status).toBe(201);
	// Hydrated reads use a different image shape from replacement writes.
	await page.route('**/api/appview/v1/events/**/event-1', async (route) => {
		if (route.request().method() !== 'GET') return route.continue();
		const response = await route.fetch();
		const event = await response.json();
		await route.fulfill({
			response,
			json: {
				...event,
				image: {
					cid: 'existing-photo-cid',
					mime: 'image/jpeg',
					size: 100,
					alt: 'Event poster',
					aspectRatio: { width: 400, height: 300 }
				}
			}
		});
	});
	await page.goto('/settings/events');
	let copied = '';
	await page.exposeFunction('captureCopiedEventLink', (value: string) => {
		copied = value;
	});
	await page.evaluate(() => {
		navigator.clipboard.writeText = (value: string) =>
			(
				window as unknown as {
					captureCopiedEventLink: (value: string) => Promise<void>;
				}
			).captureCopiedEventLink(value);
	});
	await page
		.getByLabel('Actions for Lifecycle workshop', { exact: true })
		.click();
	await page.getByRole('button', { name: 'Copy link', exact: true }).click();
	await expect
		.poll(() => copied)
		.toBe('https://app.craftsky.social/events/did%3Aplc%3Aalice/event-1');
	await page
		.getByLabel('Actions for Lifecycle workshop', { exact: true })
		.click();
	await fetch('http://127.0.0.1:19847/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ visual: true })
	});
	await page.getByRole('link', { name: 'Send via chat', exact: true }).click();
	await expect(page).toHaveURL(
		'/messages/new?link=' + encodeURIComponent(copied)
	);
	await expect(
		page.getByText(
			'Choose someone to send this link to. You can review the link before sending.'
		)
	).toBeVisible();
	await fetch('http://127.0.0.1:19847/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ visual: false })
	});
	await page.goto('/settings/events');
	await page
		.getByLabel('Actions for Lifecycle workshop', { exact: true })
		.click();
	await page
		.getByRole('button', { name: 'Postpone event', exact: true })
		.click();
	await expect(
		page.getByRole('heading', { name: 'Lifecycle workshop', exact: true })
	).toHaveCount(0);
	await page.goto('/events/did%3Aplc%3Aalice/event-1');
	await expect(page.getByText('Postponed', { exact: true })).toBeVisible();
	await page.getByLabel('Event actions', { exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Postpone event', exact: true })
	).toHaveCount(0);
	await page.getByRole('button', { name: 'Cancel event', exact: true }).click();
	await expect(page.getByText('Cancelled', { exact: true })).toBeVisible();
	await page.getByLabel('Event actions', { exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Cancel event', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Postpone event', exact: true })
	).toBeVisible();
	const writes = (await calls()).filter(
		(row) => row.method === 'PUT' && row.path.endsWith('/event-1')
	);
	expect(writes).toHaveLength(2);
	expect(writes[0]).toMatchObject({
		ifMatch: 'event-cid-1',
		body: { ...body, status: 'postponed' }
	});
	expect(writes[1]).toMatchObject({
		ifMatch: 'event-cid-1-next',
		body: { ...body, status: 'cancelled', image }
	});
	expect(writes[1].body).not.toHaveProperty('createdAt');
	expect(writes[0].key).not.toBe(writes[1].key);
});

for (const width of [390, 1280]) {
	test(`four product photos preserve order and descriptions at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/settings/products');
		await page
			.getByRole('button', { name: 'Add product', exact: true })
			.click();
		const dialog = page.locator('.product-composer-form');
		await dialog.getByLabel('Title', { exact: true }).fill('Four photo wool');
		await dialog
			.getByLabel('Destination', { exact: true })
			.fill('https://shop.example/four-photos');
		const png = await page.evaluate(() => {
			const c = document.createElement('canvas');
			c.width = 40;
			c.height = 30;
			c.getContext('2d')!.fillRect(0, 0, 40, 30);
			return c.toDataURL('image/png').split(',')[1];
		});
		await dialog.getByLabel('Add images', { exact: true }).setInputFiles(
			Array.from({ length: 4 }, (_, i) => ({
				name: `photo-${i}.png`,
				mimeType: 'image/png',
				buffer: Buffer.from(png, 'base64')
			}))
		);
		const details = page.locator('.photo-details-page');
		await expect(details).toBeVisible();
		await expect(details.getByLabel('Add images', { exact: true })).toHaveCount(
			0
		);
		for (let i = 0; i < 4; i++)
			await details
				.getByLabel('Image description', { exact: true })
				.nth(i)
				.fill(`Photo ${i + 1}`);
		for (let i = 4; i > 1; i--)
			await details
				.getByRole('button', { name: `Move image ${i} earlier`, exact: true })
				.click();
		await details
			.getByRole('button', { name: 'Remove image 3', exact: true })
			.click();
		await expect(details.getByLabel('Add images', { exact: true })).toHaveCount(
			1
		);
		await details.getByLabel('Add images', { exact: true }).setInputFiles({
			name: 'photo-5.png',
			mimeType: 'image/png',
			buffer: Buffer.from(png, 'base64')
		});
		await details
			.getByLabel('Image description', { exact: true })
			.nth(3)
			.fill('Photo 5');
		await page.keyboard.press('Escape');
		await expect(details).not.toBeVisible();
		await expect(dialog).toBeVisible();
		await dialog.locator('.photo-summary').click();
		await expect(
			details.getByLabel('Image description', { exact: true }).nth(0)
		).toHaveValue('Photo 4');
		await details
			.getByRole('button', { name: 'Back to product', exact: true })
			.click();
		await dialog
			.getByRole('button', { name: 'Save product', exact: true })
			.click();
		await expect(dialog).not.toBeVisible();
		const write = (await calls())
			.filter(
				(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
			)
			.at(-1);
		const products = write?.body.products as { images: { alt: string }[] }[];
		expect(products.at(-1)?.images.map((image) => image.alt)).toEqual([
			'Photo 4',
			'Photo 1',
			'Photo 3',
			'Photo 5'
		]);
	});
}

for (const width of [390, 1280]) {
	test(`profile products use a large four-image carousel at ${width}px`, async ({
		page
	}, testInfo) => {
		await page.route('https://fixture.craftsky.test/**', (route) =>
			route.fulfill({ path: 'static/app-icon.png', contentType: 'image/png' })
		);
		await page.setViewportSize({ width, height: 900 });
		const image = {
			cid: 'bafyproduct',
			mime: 'image/png',
			size: 200,
			alt: 'Wool',
			thumb: 'https://fixture.craftsky.test/app-icon.png',
			fullsize: 'https://fixture.craftsky.test/app-icon.png',
			aspectRatio: { width: 4, height: 3 }
		};
		await fetch('http://127.0.0.1:19847/__test/control', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({
				visual: true,
				owners: {
					alice: {
						business: {
							cid: 'business-four',
							businessTypes: [{ value: 'dyer', known: true }],
							products: [
								{
									title: 'Four photo wool',
									uri: 'https://shop.example/four',
									images: Array.from({ length: 4 }, (_, i) => ({
										...image,
										alt: `Photo ${i + 1}`
									})),
									price: { amount: '12.5', currency: 'GBP' }
								}
							]
						}
					}
				}
			})
		});
		await page.goto(`/profiles/${alice.did}?tab=products`);
		const card = page.locator('.product-card');
		await expect(card).toBeVisible();
		await expect(card.locator('.image-counter')).toHaveText(/1\s*\/\s*4/);
		const bounds = await card.locator('.gallery').boundingBox();
		expect(bounds!.height).toBeGreaterThan(240);
		expect(bounds!.width).toBeGreaterThan(300);
		await page.mouse.move(
			bounds!.x + bounds!.width * 0.8,
			bounds!.y + bounds!.height / 2
		);
		await page.mouse.down();
		await page.mouse.move(
			bounds!.x + bounds!.width * 0.2,
			bounds!.y + bounds!.height / 2,
			{ steps: 12 }
		);
		await page.mouse.up();
		await expect(card.locator('.image-counter')).toHaveText(/2\s*\/\s*4/);
		await expect(
			page.getByRole('dialog', { name: 'Open link?', exact: true })
		).not.toBeVisible();
		await card
			.getByRole('button', { name: 'Previous image', exact: true })
			.click();
		await expect(card.locator('.image-counter')).toHaveText(/1\s*\/\s*4/);
		await card.getByRole('button', { name: 'Next image', exact: true }).click();
		await expect(card.locator('.image-counter')).toHaveText(/2\s*\/\s*4/);
		const openLink = page.getByRole('dialog', {
			name: 'Open link?',
			exact: true
		});
		await expect(openLink).not.toBeVisible();
		await card
			.getByRole('button', { name: 'Image description', exact: true })
			.click();
		const description = page.getByRole('dialog', {
			name: 'Image description',
			exact: true
		});
		await expect(description).toBeVisible();
		await expect(openLink).not.toBeVisible();
		await description
			.getByRole('button', { name: 'Close', exact: true })
			.click();
		await expect(card.locator('.gallery a').nth(1)).toHaveAttribute(
			'href',
			'https://shop.example/four'
		);
		await card.locator('.gallery a').nth(1).click();
		await expect(openLink).toBeVisible();
		await expect(
			openLink.getByRole('link', { name: 'Open link', exact: true })
		).toHaveAttribute('href', 'https://shop.example/four');
		await openLink.getByRole('button', { name: 'Cancel', exact: true }).click();
		// The full footer padding is part of the same link, including empty space.
		await card.locator('.product-link').click({ position: { x: 4, y: 4 } });
		await expect(openLink).toBeVisible();
		await openLink.getByRole('button', { name: 'Cancel', exact: true }).click();
		await card.locator('.product-link').focus();
		await page.keyboard.press('Enter');
		await expect(openLink).toBeVisible();
		await openLink.getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(
			page.getByRole('dialog', { name: 'Image gallery', exact: true })
		).not.toBeVisible();
		await expect(card.locator('.product-link')).toHaveAttribute(
			'href',
			'https://shop.example/four'
		);
		const audit = await new AxeBuilder({ page })
			.include('.product-grid')
			.analyze();
		expect(audit.violations).toEqual([]);
		await card.evaluate((node) => node.scrollIntoView({ block: 'start' }));
		await expect(card.locator('.gallery img').nth(1)).toHaveJSProperty(
			'complete',
			true
		);
		expect(
			await card
				.locator('.gallery img')
				.nth(1)
				.evaluate((image) => (image as HTMLImageElement).naturalWidth)
		).toBeGreaterThan(0);
		await expect(card.locator('.product-link')).toBeInViewport();
		await card.screenshot({
			path: testInfo.outputPath(`product-carousel-${width}.png`)
		});
	});
}

test('product composer routes load directly, refresh and protect browser back', async ({
	page
}) => {
	await page.goto('/settings/products');
	await page.getByRole('button', { name: 'Add product', exact: true }).click();
	await expect(page).toHaveURL(/\/compose\/product$/);
	await page.reload();
	await expect(
		page.getByRole('heading', { name: 'Add product', exact: true })
	).toBeVisible();
	await expect(page.getByLabel('Title', { exact: true })).toHaveValue('');
	await page
		.getByLabel('Title', { exact: true })
		.fill('Unfinished routed product');
	await page.goBack();
	await page.getByRole('button', { name: 'Keep editing', exact: true }).click();
	await expect(page).toHaveURL(/\/compose\/product$/);
	await expect(page.getByLabel('Title', { exact: true })).toHaveValue(
		'Unfinished routed product'
	);
	await page.goBack();
	await page.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/products$/);
	await page.goto('/compose/product/edit?product=0');
	await expect(page.getByLabel('Title', { exact: true })).toHaveValue(
		'Blue wool'
	);
	await page.reload();
	await expect(
		page.getByRole('heading', { name: 'Edit product', exact: true })
	).toBeVisible();
	await expect(page.getByLabel('Title', { exact: true })).toHaveValue(
		'Blue wool'
	);
	await page.getByLabel('Title', { exact: true }).fill('Edited routed wool');
	await page.getByRole('button', { name: 'Save product', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/products$/);
	await page
		.getByRole('button', { name: 'Edit Edited routed wool', exact: true })
		.click();
	await expect(page).toHaveURL(/\/compose\/product\/edit\?product=0$/);
	await expect(page.getByLabel('Title', { exact: true })).toHaveValue(
		'Edited routed wool'
	);
	await page
		.getByRole('button', { name: 'Close product editor', exact: true })
		.click();
	await expect(page).toHaveURL(/\/settings\/products$/);
	await page.goto('/compose/product');
	await expect(page.getByLabel('Title', { exact: true })).toBeVisible();
	await page
		.getByRole('button', { name: 'Close product editor', exact: true })
		.click();
	await expect(page).toHaveURL(/\/settings\/products$/);
});

for (const query of [
	'',
	'?product=-1',
	'?product=1',
	'?product=0.5',
	'?product=invalid'
])
	test(`invalid product edit route ${query || '(missing)'} does not open another product`, async ({
		page
	}) => {
		await page.goto('/compose/product/edit' + query);
		await expect(
			page.getByRole('heading', { name: 'Edit product', exact: true })
		).toBeVisible();
		await expect(
			page.getByRole('button', { name: 'Try again', exact: true })
		).toBeVisible();
		await expect(page.locator('.product-composer-form')).toHaveCount(0);
	});

test('direct product creation respects the four-product limit', async ({
	page
}) => {
	await fetch('http://127.0.0.1:19847/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({
			owners: {
				alice: {
					business: {
						cid: 'full-catalog',
						businessTypes: [],
						offerings: [],
						products: Array.from({ length: 4 }, (_, i) => ({
							title: `Product ${i}`,
							uri: `https://shop.example/${i}`,
							images: []
						}))
					}
				}
			}
		})
	});
	await page.goto('/compose/product');
	await expect(
		page.getByRole('button', { name: 'Try again', exact: true })
	).toBeVisible();
	await expect(page.locator('.product-composer-form')).toHaveCount(0);
	expect(
		(await calls()).filter(
			(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
		)
	).toHaveLength(0);
});

for (const width of [390, 1280, 1767])
	test(`product photo details stay inside the main column at ${width}px`, async ({
		page
	}, testInfo) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/compose/product');
		await expect(page.getByLabel('Title', { exact: true })).toBeVisible();
		const main = page.locator('main.app-main');
		const original = await main.boundingBox();
		const png = await page.evaluate(() => {
			const canvas = document.createElement('canvas');
			canvas.width = 40;
			canvas.height = 30;
			return canvas.toDataURL('image/png').split(',')[1];
		});
		await page.getByLabel('Add images', { exact: true }).setInputFiles({
			name: 'product.png',
			mimeType: 'image/png',
			buffer: Buffer.from(png, 'base64')
		});
		const details = page.getByRole('region', {
			name: 'Photo details',
			exact: true
		});
		await expect(details).toBeVisible();
		const bounds = await details.boundingBox();
		expect(bounds!.x).toBeCloseTo(original!.x, 0);
		expect(bounds!.width).toBeCloseTo(original!.width, 0);
		await details.screenshot({
			path: testInfo.outputPath(`product-photo-details-${width}.png`)
		});
		await page.setViewportSize({ width: width + 180, height: 900 });
		await expect
			.poll(async () => {
				const mainBox = await main.boundingBox(),
					photoBox = await details.boundingBox();
				return (
					Math.abs(mainBox!.x - photoBox!.x) +
					Math.abs(mainBox!.width - photoBox!.width)
				);
			})
			.toBeLessThan(1);
		await details
			.getByRole('button', { name: 'Back to product', exact: true })
			.click();
		await expect(details).toBeHidden();
		await expect(page.getByLabel('Title', { exact: true })).toBeVisible();
	});

test('product photo details restore composer scroll on return', async ({
	page
}) => {
	await page.setViewportSize({ width: 1280, height: 520 });
	await page.goto('/compose/product');
	await page
		.getByLabel('Title', { exact: true })
		.fill('Unfinished photo draft');
	await addProductPhoto(page);
	await expect(
		page.getByRole('region', { name: 'Photo details', exact: true })
	).toBeHidden();
	const scroller = page.locator('.business-composer-scroll');
	const position = await scroller.evaluate((node) => {
		node.scrollTop = node.scrollHeight;
		return node.scrollTop;
	});
	expect(position).toBeGreaterThan(0);
	await page.locator('.photo-summary').click();
	const details = page.getByRole('region', {
		name: 'Photo details',
		exact: true
	});
	await expect(details).toBeVisible();
	await details
		.getByRole('button', { name: 'Back to product', exact: true })
		.click();
	await expect(details).toBeHidden();
	await expect
		.poll(() => scroller.evaluate((node) => node.scrollTop))
		.toBeCloseTo(position, 0);
	await expect(page.getByLabel('Title', { exact: true })).toHaveValue(
		'Unfinished photo draft'
	);
});
