import { test, expect, type BrowserContext } from '@playwright/test';
import bzipFixture from '../src/lib/features/settings/bzip-fixture.json' with {type:'json'};
test('business images are reviewed, cropped and stripped/recompressed before explicit upload', async ({
	page
}) => {
	await page.goto('/app/settings/products');
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
	await page.getByLabel('Replace image', { exact: true }).setInputFiles({
		name: 'large-source.png',
		mimeType: 'image/png',
		buffer: large
	});
	const dialog = page.getByRole('dialog');
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
	await page
		.getByRole('button', { name: 'Save products', exact: true })
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
	await page.getByLabel('Find a language').fill('French');
	await page.getByLabel('French', { exact: true }).check();
	await page.getByLabel('Find a language').fill('English');
	await page.getByLabel('English', { exact: true }).uncheck();
	await page.getByRole('button', { name: 'Save languages' }).click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Saved.' })
	).toBeVisible();
	const request = (await calls()).find(
		(row) => row.method === 'PUT' && row.path === '/v1/languages/preferences'
	);
	expect(request?.body).toEqual({
		primaryLanguage: 'cy',
		contentLanguages: ['fr']
	});
});
test('notification changes patch only the chosen category and field', async ({
	page
}) => {
	await page.goto('/app/settings/notifications');
	const row = page
		.locator('.row')
		.filter({ has: page.getByRole('heading', { name: 'Likes', exact: true }) });
	await row.getByRole('combobox').selectOption('peopleIFollow');
	await expect(row.getByRole('combobox')).toBeEnabled();
	await row.getByLabel('Push', { exact: true }).uncheck();
	await expect(row.getByLabel('Push', { exact: true })).toBeEnabled();
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
	).toBeVisible();
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
	await expect(
		page.locator('code').filter({ hasText: 'did:plc:bob' })
	).toBeVisible();
});
test('BZip2 ZIP imports use the bundled worker and require review before sending usernames', async ({
	page
}) => {
	await page.goto('/app/settings/instagram');
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
	await page
		.getByRole('button', { name: 'Verify Instagram username', exact: true })
		.click();
	await expect(
		page.getByText('craftsky-test-challenge', { exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'Check verification status' }).click();
	await expect(page.getByText('@alice_makes', { exact: true })).toBeVisible();
	await page
		.getByRole('button', { name: 'Confirm this is my Instagram account' })
		.click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Instagram account verified.' })
	).toBeVisible();
	await page.getByText('Enter usernames manually', { exact: true }).click();
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
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
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
		page.getByText('No upcoming events.', { exact: true })
	).toBeVisible();
	const requests = await calls();
	const create = requests.find(
		(row) => row.method === 'POST' && row.path === '/v1/events'
	);
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
