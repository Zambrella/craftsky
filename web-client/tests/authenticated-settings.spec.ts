import { test, expect, type BrowserContext, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import bzipFixture from '../src/lib/features/settings/bzip-fixture.json' with { type: 'json' };
test('business images are reviewed, cropped and stripped/recompressed before explicit upload', async ({
	page
}) => {
	await page.goto('/app/settings/products');
	await page
		.getByRole('button', { name: 'Edit Blue wool', exact: true })
		.click();
	const productDialog = page.locator('.product-dialog');
	await expect(productDialog).toBeVisible();
	const png = await page.evaluate(() => {
		const canvas = document.createElement('canvas');
		canvas.width = 32;
		canvas.height = 48;
		canvas.getContext('2d')!.fillRect(0, 0, 32, 48);
		return canvas.toDataURL('image/png').split(',')[1];
	});
	const large = Buffer.concat([
		Buffer.from(png, 'base64'),
		Buffer.alloc(2_100_000)
	]);
	await productDialog
		.getByLabel('Replace image', { exact: true })
		.setInputFiles({
			name: 'large-source.png',
			mimeType: 'image/png',
			buffer: large
		});
	const dialog = page.locator('.business-image-dialog');
	await expect(dialog).toBeVisible();
	expect(
		(await calls()).filter((row) => row.path === '/v1/blobs/images')
	).toHaveLength(0);
	await dialog
		.getByRole('button', { name: 'Crop / rotate', exact: true })
		.click();
	await expect(
		dialog.getByRole('button', { name: 'Use image', exact: true })
	).toBeDisabled();
	await dialog
		.getByRole('button', { name: 'Rotate 90° ↻', exact: true })
		.click();
	await dialog.getByRole('button', { name: 'Apply edit', exact: true }).click();
	await expect(
		dialog.getByRole('button', { name: 'Use image', exact: true })
	).toBeEnabled();
	await dialog
		.getByLabel('Image description', { exact: true })
		.fill('Portrait image rotated to landscape');
	await dialog.getByRole('button', { name: 'Use image', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await expect(productDialog).toBeVisible();
	await productDialog
		.getByRole('button', { name: 'Save product', exact: true })
		.click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Products saved.' })
	).toBeVisible();
	const requests = await calls();
	const upload = requests.find(
		(row) => row.path === '/v1/blobs/images'
	) as unknown as { bytes: number; contentType: string };
	expect(upload.contentType).toBe('image/jpeg');
	expect(upload.bytes).toBeLessThan(2_000_000);
	const write = requests.find(
		(row) => row.method === 'PUT' && row.path === '/v1/profiles/me/business'
	);
	expect(write?.owner).toBe(alice.did);
	expect(write?.ifMatch).toBe('business-cid-0');
	expect(write?.key).toMatch(/^[0-9a-f-]{36}$/);
	expect(write?.body.products).toEqual([
		expect.objectContaining({
			image: expect.objectContaining({
				alt: 'Portrait image rotated to landscape',
				aspectRatio: { width: 48, height: 32 }
			})
		})
	]);
});
test('bundled service worker activates without Firebase credentials or remote requests', async ({
	page
}) => {
	await page.goto('/app/settings/notifications');
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
	await page
		.getByRole('button', { name: 'Create verification challenge', exact: true })
		.click();
	await expect(
		page.getByText('craftsky-test-challenge', { exact: true })
	).toBeVisible();
	await page
		.getByRole('button', { name: 'Check verification status', exact: true })
		.click();
	await expect(page.getByText('@alice_makes', { exact: true })).toBeVisible();
	await page
		.getByRole('button', {
			name: 'Confirm this is my Instagram account',
			exact: true
		})
		.click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Instagram account verified.' })
	).toBeVisible();
}
test.beforeEach(async ({ context, page }) => {
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await fetch('http://127.0.0.1:19847/__test/reset');
	await login(context);
});
test('appearance changes immediately and survives reload', async ({ page }) => {
	await page.goto('/app/settings/appearance');
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
	await page.goto('/app/settings/languages');
	await page
		.getByRole('combobox', { name: 'Primary language', exact: true })
		.selectOption('cy');
	await expect(
		page.getByRole('combobox', { name: 'Primary language', exact: true })
	).toBeEnabled();
	await page
		.getByRole('searchbox', { name: 'Content languages', exact: true })
		.click();
	await page
		.getByRole('searchbox', { name: 'Content languages', exact: true })
		.fill('French');
	await page.getByLabel('French', { exact: true }).check();
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
	).toHaveValue('cy');
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
	await page.goto('/app/settings/notifications');
	const row = page
		.locator('.notification-card')
		.filter({ has: page.getByRole('heading', { name: 'Likes', exact: true }) });
	await row.getByRole('combobox').selectOption('peopleIFollow');
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
	await page.goto('/app/settings/business');
	await page
		.getByLabel('Tagline', { exact: true })
		.fill('Small batch colour for your next project');
	await page.getByRole('button', { name: 'Save business profile' }).click();
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
test('license assignment targets the entered DID and is reflected after reloading billing', async ({
	page
}) => {
	await page.goto('/app/settings/subscriptions');
	await expect(
		page.getByRole('link', { name: 'Manage in App Store', exact: true })
	).toHaveAttribute('href', 'https://apps.apple.com/account/subscriptions');
	await page.getByText('Assign another account', { exact: true }).click();
	await page.getByLabel('Account DID', { exact: true }).fill('did:plc:bob');
	await page
		.getByRole('button', { name: 'Assign license', exact: true })
		.click();
	await expect(
		page.getByRole('status').filter({ hasText: 'License assignment updated.' })
	).toBeVisible();
	const request = (await calls()).find(
		(row) => row.method === 'PUT' && row.path.endsWith('/license-1/assignment')
	);
	expect(request?.body).toEqual({ targetDid: 'did:plc:bob' });
	expect(request?.owner).toBe(alice.did);
	await page.reload();
	await expect(
		page.getByRole('combobox', { name: 'Account', exact: true })
	).toHaveValue('did:plc:bob');
	await expect(
		page.locator('.license p strong').filter({ hasText: 'did:plc:bob' })
	).toBeVisible();
});
test('BZip2 ZIP imports use the bundled worker and require review before sending usernames', async ({
	page
}) => {
	await page.goto('/app/settings/instagram');
	await verifyInstagram(page);
	await expect(
		page.getByRole('button', { name: 'Instagram export', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	await page
		.getByLabel('Choose your Instagram ZIP export or following.json', {
			exact: true
		})
		.setInputFiles({
			name: 'instagram-bzip2.zip',
			mimeType: 'application/zip',
			buffer: Buffer.from(bzipFixture.zip, 'base64')
		});
	await expect(
		page.getByRole('heading', { name: 'Review import: instagram-bzip2.zip' })
	).toBeVisible();
	await expect(page.getByText('@maker.one', { exact: true })).toBeVisible();
	expect(
		(await calls()).filter(
			(row) => row.method === 'POST' && row.path.endsWith('/instagram/imports')
		)
	).toHaveLength(0);
	await page
		.getByRole('button', { name: 'Import 1 usernames', exact: true })
		.click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Following list imported.' })
	).toBeVisible();
	expect(
		(await calls()).find(
			(row) => row.method === 'POST' && row.path.endsWith('/instagram/imports')
		)?.body
	).toEqual({
		sourceType: 'instagramJson',
		entries: [{ username: 'maker.one' }]
	});
});
test('moderation history shows the affected content reference and its reviewed post destination', async ({
	page
}) => {
	await page.goto('/app/settings/moderation');
	await expect(
		page.getByText(
			'at://did:plc:alice/social.craftsky.feed.post/moderated-post',
			{ exact: true }
		)
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'View post', exact: true })
	).toHaveAttribute('href', '/app/posts/did%3Aplc%3Aalice/moderated-post');
	await expect(
		page.getByRole('link', { name: 'Appeal by email', exact: true })
	).toHaveAttribute('href', /^mailto:moderation@craftsky.social\?/);
});
test('Instagram verification, reviewed manual import and following a match complete through real controls', async ({
	page
}) => {
	await page.goto('/app/settings/instagram');
	await verifyInstagram(page);
	await page
		.getByRole('button', { name: 'Enter handles', exact: true })
		.click();
	await expect(
		page.getByRole('button', { name: 'Enter handles', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	await page
		.getByLabel('Instagram usernames', { exact: true })
		.fill('@maker_one\nMaker_Two\nmaker_one\ninvalid name');
	await page.getByRole('button', { name: 'Review usernames' }).click();
	await expect(
		page.getByText(
			'2 valid usernames · 1 duplicates removed · 1 invalid entries ignored',
			{ exact: true }
		)
	).toBeVisible();
	await page.getByRole('button', { name: 'Import 2 usernames' }).click();
	await expect(page.getByText('Bob Maker', { exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Follow', exact: true }).click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Following this maker.' })
	).toBeVisible();
	const requests = await calls();
	expect(
		requests.find((row) => row.path.endsWith('/verify-1/confirm'))?.body
	).toEqual({ discoverable: true });
	expect(
		requests.find(
			(row) => row.method === 'POST' && row.path.endsWith('/instagram/imports')
		)?.body
	).toEqual({
		sourceType: 'manual',
		entries: [{ username: 'maker_one' }, { username: 'maker_two' }]
	});
	expect(
		requests.find((row) => row.path.endsWith('/suggestion-1/accept'))?.key
	).toMatch(/^[0-9a-f-]{36}$/);
});
test('business events create, edit using the accepted CID, and delete', async ({
	page
}) => {
	await page.goto('/app/settings/events');
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
	await expect(page.locator('.event-dialog')).toBeVisible();
	await page
		.getByLabel('Event name', { exact: true })
		.fill('Indigo dye workshop');
	await page
		.getByRole('combobox', { name: 'Time zone', exact: true })
		.selectOption('UTC');
	await page.getByLabel('Starts', { exact: true }).fill('2026-12-01T10:00');
	await page.getByLabel('Ends', { exact: true }).fill('2026-12-01T12:00');
	await page.getByRole('button', { name: 'Save event' }).click();
	await expect(
		page.getByRole('heading', { name: 'Indigo dye workshop', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'Edit', exact: true }).click();
	await page
		.getByLabel('Event name', { exact: true })
		.fill('Natural indigo workshop');
	await page.getByRole('button', { name: 'Save event' }).click();
	await expect(
		page.getByRole('heading', { name: 'Natural indigo workshop', exact: true })
	).toBeVisible();
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: 'Delete', exact: true }).click();
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
		startsAt: '2026-12-01T10:00:00.000Z',
		endsAt: '2026-12-01T12:00:00.000Z',
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

test('customisation changes preview choices and saves the exact native colour and background values', async ({
	page
}) => {
	await page.goto('/app/settings/customisation');
	const save = page.getByRole('button', { name: 'Save', exact: true });
	await expect(save).toBeDisabled();
	await page.getByText('Green', { exact: true }).click();
	await page.getByText('Cross stitch', { exact: true }).click();
	await expect(
		page.getByRole('radio', { name: 'Green', exact: true })
	).toBeChecked();
	await expect(
		page.getByRole('radio', { name: 'Cross stitch', exact: true })
	).toBeChecked();
	await expect(
		page.getByLabel('Profile customisation preview')
	).toHaveAttribute('style', /--chosen-colour:\s*#23770f/i);
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
		profileBackground: 'dotcrossdark'
	});
	await page.reload();
	await expect(
		page.getByRole('radio', { name: 'Green', exact: true })
	).toBeChecked();
	await expect(
		page.getByRole('radio', { name: 'Cross stitch', exact: true })
	).toBeChecked();
	await expect(
		page.getByRole('button', { name: 'Save', exact: true })
	).toBeDisabled();
});

test('product modal cancellation restores existing details and removes an unsaved new product without writing', async ({
	page
}) => {
	await page.goto('/app/settings/products');
	await page
		.getByRole('button', { name: 'Edit Blue wool', exact: true })
		.click();
	let dialog = page.locator('.product-dialog');
	await expect(dialog).toBeVisible();
	await dialog.getByLabel('Title', { exact: true }).fill('Unsaved wool');
	await dialog.getByLabel('Price · optional', { exact: true }).fill('99');
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await page
		.getByRole('button', { name: 'Edit Blue wool', exact: true })
		.click();
	dialog = page.locator('.product-dialog');
	await expect(dialog.getByLabel('Title', { exact: true })).toHaveValue(
		'Blue wool'
	);
	await expect(
		dialog.getByLabel('Price · optional', { exact: true })
	).toHaveValue('12.5');
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await page.getByRole('button', { name: 'Add product', exact: true }).click();
	await expect(dialog).toBeVisible();
	await dialog.getByLabel('Title', { exact: true }).fill('Unsaved new product');
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
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

test('About displays the embedded build version alongside native legal rows', async ({
	page
}) => {
	await page.goto('/app/settings/about');
	const version = readFileSync(
		new URL('../../appview/VERSION', import.meta.url),
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
