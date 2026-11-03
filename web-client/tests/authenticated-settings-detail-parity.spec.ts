import { test, expect, type Route } from '@playwright/test';

const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const appview = '/api/appview/v1';
const status = (enabled: boolean) => ({
	generation: 1,
	enabled,
	pushEnabled: false,
	state: enabled ? 'ready' : 'disabled',
	capabilities: {
		canRead: enabled,
		canSend: enabled,
		canReport: enabled,
		canSetPolicy: enabled
	},
	syncState: 'ready'
});
type Captured = {
	method: string;
	body: Record<string, unknown>;
	headers: Record<string, string>;
};
function capture(route: Route): Captured {
	const request = route.request();
	return {
		method: request.method(),
		body: request.postDataJSON(),
		headers: request.headers()
	};
}
async function readJSON(route: Route, value: unknown) {
	await route.fulfill({ status: 200, json: value });
}

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
});

test('settings enables messaging in place and failed shared preferences keep their saved values', async ({
	page
}) => {
	let enabled = false;
	const preferenceReads: string[] = [],
		writes: Captured[] = [];
	await page.route(`**${appview}/chat/status`, (route) =>
		readJSON(route, status(enabled))
	);
	await page.route(`**${appview}/chat/connection`, async (route) => {
		writes.push(capture(route));
		enabled = route.request().postDataJSON().enabled;
		await readJSON(route, status(enabled));
	});
	for (const endpoint of ['push-preferences', 'incoming-policy']) {
		await page.route(`**${appview}/chat/${endpoint}`, async (route) => {
			if (route.request().method() === 'GET') {
				preferenceReads.push(endpoint);
				await readJSON(
					route,
					endpoint === 'push-preferences'
						? { pushEnabled: false }
						: { allowIncoming: 'following', persisted: true }
				);
			} else {
				writes.push(capture(route));
				await route.fulfill({
					status: 400,
					json: {
						error: 'invalid_request',
						message: 'Preference was not saved.'
					}
				});
			}
		});
	}
	await page.goto('/app/settings/messaging');
	const connection = page.getByRole('switch', {
		name: /^Enable messaging(?:\s|$)/
	});
	await expect(connection).not.toBeChecked();
	expect(preferenceReads).toEqual([]);
	await connection.check();
	const push = page.getByRole('switch', {
		name: /^Push notifications(?:\s|$)/
	});
	await expect(push).toBeVisible();
	await expect(push).toBeEnabled();
	await expect(page).toHaveURL(/\/app\/settings\/messaging$/);
	expect(preferenceReads.sort()).toEqual([
		'incoming-policy',
		'push-preferences'
	]);
	await push.click();
	await expect(page.getByRole('alert')).toContainText(
		'Preference was not saved.'
	);
	await expect(push).not.toBeChecked();
	const incoming = page.getByRole('combobox', { name: 'Who can message me' });
	await expect(incoming).toHaveValue('following');
	await incoming.selectOption('nobody');
	await expect(page.getByRole('alert')).toContainText(
		'Preference was not saved.'
	);
	await expect(incoming).toHaveValue('following');
	expect(writes.map((value) => value.body)).toEqual([
		{ enabled: true },
		{ pushEnabled: true },
		{ allowIncoming: 'nobody' }
	]);
	expect(
		writes.every((value) => value.headers['x-craftsky-account-id'] === alice.id)
	).toBe(true);
});

test('an unconfirmed messaging connection does not leave the switch enabled', async ({
	page
}) => {
	await page.route(`**${appview}/chat/status`, (route) =>
		readJSON(route, status(false))
	);
	await page.route(`**${appview}/chat/connection`, (route) =>
		readJSON(route, status(false))
	);
	await page.goto('/app/settings/messaging');
	const connection = page.getByRole('switch', {
		name: /^Enable messaging(?:\s|$)/
	});
	await connection.click();
	await expect(page.getByRole('alert')).toContainText(
		'Messaging connection change was not confirmed.'
	);
	await expect(connection).not.toBeChecked();
	await expect(
		page.getByRole('switch', { name: /^Push notifications(?:\s|$)/ })
	).toHaveCount(0);
});

test('failed native notification choices restore the saved category without changing another category', async ({
	page
}) => {
	const writes: Captured[] = [];
	await page.route(`**${appview}/notifications/preferences`, async (route) => {
		if (route.request().method() === 'GET')
			return readJSON(route, {
				preferences: {
					like: { scope: 'everyone', pushEnabled: false },
					follow: { scope: 'peopleIFollow', pushEnabled: true }
				}
			});
		writes.push(capture(route));
		await route.fulfill({
			status: 400,
			json: {
				error: 'invalid_request',
				message: 'Notification preferences were not saved.'
			}
		});
	});
	await page.goto('/app/settings/notifications');
	const likes = page
		.locator('.notification-card')
		.filter({ has: page.getByRole('heading', { name: 'Likes', exact: true }) });
	await likes.getByRole('combobox').selectOption('peopleIFollow');
	await expect(page.getByRole('alert')).toContainText(
		'Notification preferences were not saved.'
	);
	await expect(likes.getByRole('combobox')).toHaveValue('everyone');
	await likes
		.getByRole('switch', { name: 'Likes push notifications', exact: true })
		.click();
	await expect(
		likes.getByRole('switch', { name: 'Likes push notifications', exact: true })
	).not.toBeChecked();
	const follows = page.locator('.notification-card').filter({
		has: page.getByRole('heading', { name: 'Follows', exact: true })
	});
	await expect(follows.getByRole('combobox')).toHaveValue('peopleIFollow');
	await expect(follows.getByRole('switch')).toBeChecked();
	expect(writes.map((value) => value.body)).toEqual([
		{ preferences: { like: { scope: 'peopleIFollow' } } },
		{ preferences: { like: { pushEnabled: true } } }
	]);
});

test('future customisation values render native defaults without writing them back', async ({
	page
}) => {
	const writes: Captured[] = [],
		assetRequests: string[] = [];
	await page.route(`**${appview}/profiles/me`, async (route) => {
		const response = await route.fetch();
		const profile = await response.json();
		await route.fulfill({
			response,
			json: {
				...profile,
				customisation: {
					colour: 'future-colour',
					profileBackground: '../../unsupported-pattern'
				}
			}
		});
	});
	await page.route(`**${appview}/profiles/me/customisation`, async (route) => {
		writes.push(capture(route));
		await readJSON(route, {});
	});
	page.on('request', (request) => {
		if (request.url().includes('/profile_backgrounds/'))
			assetRequests.push(request.url());
	});
	await page.goto('/app/settings/customisation');
	await expect(
		page.getByRole('radio', { name: 'Cobalt', exact: true })
	).toBeChecked();
	await expect(
		page.getByRole('radio', { name: 'None', exact: true })
	).toBeChecked();
	await expect(
		page.getByRole('button', { name: 'Save', exact: true })
	).toBeDisabled();
	expect(writes).toEqual([]);
	expect(assetRequests).toEqual([]);
	await page.getByRole('radio', { name: 'Rose', exact: true }).check();
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Profile customisation saved.' })
	).toBeVisible();
	expect(writes[0].body).toEqual({ colour: 'rose', profileBackground: 'none' });
	expect(writes[0].headers['x-craftsky-account-id']).toBe(alice.id);
});

test('native content-language chips preserve authoritative preferences after failure and replace only the chosen selection', async ({
	page
}) => {
	let preferences = { primaryLanguage: 'cy', contentLanguages: ['en', 'cy'] };
	let refuse = true;
	const writes: Captured[] = [];
	await page.route(`**${appview}/languages/preferences`, async (route) => {
		if (route.request().method() === 'GET') return readJSON(route, preferences);
		writes.push(capture(route));
		if (refuse) {
			refuse = false;
			return route.fulfill({
				status: 400,
				json: {
					error: 'invalid_request',
					message: 'Language preferences were not saved.'
				}
			});
		}
		preferences = route.request().postDataJSON();
		await readJSON(route, preferences);
	});
	await page.goto('/app/settings/languages');
	await page
		.getByRole('button', {
			name: 'Remove English from Content languages',
			exact: true
		})
		.click();
	await expect(page.getByRole('alert')).toContainText(
		'Language preferences were not saved.'
	);
	await expect(
		page.getByRole('button', {
			name: 'Remove English from Content languages',
			exact: true
		})
	).toBeVisible();
	await expect(
		page.getByRole('button', {
			name: 'Remove Welsh from Content languages',
			exact: true
		})
	).toBeVisible();
	const picker = page.getByRole('searchbox', {
		name: 'Content languages',
		exact: true
	});
	await picker.fill('French');
	await page.getByRole('checkbox', { name: 'French', exact: true }).click();
	await expect(
		page.getByRole('button', {
			name: 'Remove French from Content languages',
			exact: true
		})
	).toBeVisible();
	await page
		.getByRole('button', {
			name: 'Remove English from Content languages',
			exact: true
		})
		.click();
	await expect(
		page.getByRole('button', {
			name: 'Remove English from Content languages',
			exact: true
		})
	).toHaveCount(0);
	await expect(
		page.getByRole('combobox', { name: 'Primary language', exact: true })
	).toHaveValue('cy');
	expect(writes.map((value) => value.body)).toEqual([
		{ primaryLanguage: 'cy', contentLanguages: ['cy'] },
		{ primaryLanguage: 'cy', contentLanguages: ['en', 'cy', 'fr'] },
		{ primaryLanguage: 'cy', contentLanguages: ['cy', 'fr'] }
	]);
	expect(
		writes.every((value) => value.headers['x-craftsky-account-id'] === alice.id)
	).toBe(true);
});

test('following is the native count-bearing profile list with inline crafts and no extra unfollow action', async ({
	page
}) => {
	await page.route(`**${appview}/profiles/me/following`, (route) =>
		readJSON(route, {
			items: [
				{
					did: 'did:plc:bob',
					handle: 'bob.test',
					displayName: 'Bob Maker',
					crafts: [
						'social.craftsky.defs#knitting',
						'social.craftsky.defs#future-craft'
					]
				}
			],
			totalCount: 42,
			cursor: null
		})
	);
	await page.goto('/app/settings/following');
	await expect(
		page.getByRole('heading', { level: 1, name: 'Following (42)' })
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: /Bob Maker.*@bob.test/ })
	).toHaveAttribute('href', '/app/profile/did%3Aplc%3Abob');
	await expect(
		page.getByRole('img', { name: /Crafts: Knitting, Future craft/i })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Unfollow', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByText('social.craftsky.defs#knitting', { exact: true })
	).toHaveCount(0);
});

test('product currency uses the supported ISO catalogue and saves a native-length Unicode title with ownership and CID intact', async ({
	page
}) => {
	const writes: Captured[] = [];
	await page.route(`**${appview}/profiles/me/business`, async (route) => {
		if (route.request().method() !== 'PUT') return route.continue();
		writes.push(capture(route));
		await readJSON(route, {
			uri: `at://${alice.did}/social.craftsky.business.profile/self`,
			cid: 'business-cid-next',
			rkey: 'self',
			indexing: 'pending'
		});
	});
	await page.goto('/app/settings/products');
	await page
		.getByRole('button', { name: 'Edit Blue wool', exact: true })
		.click();
	const dialog = page.locator('.product-dialog');
	const currency = dialog.getByRole('combobox', {
		name: 'Currency',
		exact: true
	});
	await expect(currency).toHaveValue('GBP');
	await expect(
		currency.getByRole('option', { name: 'JPY - Japanese Yen', exact: true })
	).toHaveCount(1);
	await currency.selectOption('JPY');
	await dialog.getByLabel('Price · optional', { exact: true }).fill('2500');
	await dialog.getByLabel('Title', { exact: true }).fill('🧶'.repeat(150));
	await dialog
		.getByRole('button', { name: 'Save product', exact: true })
		.click();
	await expect(
		page.getByRole('status').filter({ hasText: 'Products saved.' })
	).toBeVisible();
	expect(writes).toHaveLength(1);
	expect(writes[0].headers['x-craftsky-account-id']).toBe(alice.id);
	expect(writes[0].headers['if-match']).toBe('business-cid-0');
	expect(writes[0].headers['idempotency-key']).toMatch(/^[0-9a-f-]{36}$/);
	expect(writes[0].body.products).toEqual([
		expect.objectContaining({
			title: '🧶'.repeat(150),
			price: { amount: '2500', currency: 'JPY' }
		})
	]);
});

test('all-day event edits retain future catalog values and freeze the original date boundaries and CID across an ambiguous retry', async ({
	page
}) => {
	const writes: Captured[] = [];
	const event = {
		did: alice.did,
		rkey: 'future-event',
		cid: 'event-original-cid',
		uri: `at://${alice.did}/social.craftsky.business.event/future-event`,
		name: 'Future maker weekend',
		startsAt: '2027-06-01T09:00:00Z',
		endsAt: '2027-06-01T11:00:00Z',
		timeZone: 'Europe/London',
		isAllDay: false,
		roles: [{ value: 'future-role', known: false }],
		mode: { value: 'future-mode', known: false },
		status: { value: 'future-status', known: false },
		createdAt: '2026-11-01T12:00:00Z',
		past: false,
		publicSuppressionReasons: [],
		upcomingExclusionReasons: []
	};
	await page.route(`**${appview}/events?*`, (route) =>
		readJSON(route, { items: [event], cursor: null })
	);
	await page.route(
		`**${appview}/events/${encodeURIComponent(alice.did)}/future-event`,
		async (route) => {
			writes.push(capture(route));
			if (writes.length === 1)
				return route.fulfill({
					status: 202,
					headers: { 'Retry-After': '1' },
					json: { status: 'ambiguous' }
				});
			await readJSON(route, {
				uri: event.uri,
				cid: 'event-next-cid',
				rkey: event.rkey,
				indexing: 'pending'
			});
		}
	);
	await page.goto('/app/settings/events');
	await page.getByRole('button', { name: 'Edit', exact: true }).click();
	const dialog = page.locator('.event-dialog');
	await expect(
		dialog.getByRole('combobox', { name: 'Mode', exact: true })
	).toHaveValue('future-mode');
	await expect(
		dialog.getByRole('combobox', { name: 'Status', exact: true })
	).toHaveValue('future-status');
	await dialog
		.getByRole('searchbox', { name: 'Your roles', exact: true })
		.click();
	await expect(dialog.getByLabel('Future role', { exact: true })).toBeChecked();
	await dialog
		.getByRole('searchbox', { name: 'Your roles', exact: true })
		.press('Escape');
	await dialog
		.getByRole('switch', { name: 'All-day event', exact: true })
		.check();
	await expect(dialog.getByLabel('Starts', { exact: true })).toHaveAttribute(
		'type',
		'date'
	);
	await dialog.getByLabel('Starts', { exact: true }).fill('2027-06-01');
	await dialog.getByLabel('Ends', { exact: true }).fill('2027-06-02');
	await dialog.getByRole('button', { name: 'Save event', exact: true }).click();
	await expect(dialog.getByLabel('Starts', { exact: true })).toBeDisabled();
	await dialog
		.getByRole('button', { name: 'Check and retry action', exact: true })
		.click();
	await expect(dialog).not.toBeVisible();
	expect(writes).toHaveLength(2);
	expect(writes[1]).toEqual(writes[0]);
	expect(writes[0].headers['if-match']).toBe(event.cid);
	expect(writes[0].headers['x-craftsky-account-id']).toBe(alice.id);
	expect(writes[0].body).toMatchObject({
		isAllDay: true,
		timeZone: 'Europe/London',
		startsAt: '2027-05-31T23:00:00.000Z',
		endsAt: '2027-06-01T23:00:00.000Z',
		roles: ['future-role'],
		mode: 'future-mode',
		status: 'future-status'
	});
});

test('future subscription states keep independent tier cards without exposing unsafe assignment controls', async ({
	page
}) => {
	await page.route(`**${appview}/billing/account`, (route) =>
		readJSON(route, {
			billingAccountId: 'billing-1',
			revenueCatAppUserId: 'billing-owner',
			requestedGeneration: 1,
			reconciledGeneration: 1,
			reconciliationStale: false,
			subscriptions: [
				{
					id: 'subscription-1',
					productId: 'future-product',
					store: 'future_store',
					status: 'future_status',
					givesAccess: true,
					pendingPayment: false,
					anomaly: 'none'
				}
			],
			licenses: [
				{
					id: 'license-1',
					subscriptionId: 'subscription-1',
					tier: 'business',
					assignable: true,
					anomaly: 'none'
				}
			]
		})
	);
	await page.goto('/app/settings/subscriptions');
	await expect(
		page.getByRole('region', { name: 'CraftSky Plus', exact: true })
	).toBeVisible();
	const business = page.getByRole('region', {
		name: 'CraftSky Business',
		exact: true
	});
	await expect(business).toContainText('Status unavailable');
	await expect(
		page.getByRole('button', { name: 'Assign license', exact: true })
	).toHaveCount(0);
	await expect(page.getByRole('link', { name: /^Manage in/ })).toHaveCount(0);
	await expect(
		page.getByText(
			'Purchase or restore subscriptions in the CraftSky mobile app.',
			{ exact: false }
		)
	).toBeVisible();
});

test('Instagram keeps the native private confirmation choice and describes all revoke effects before deletion', async ({
	page
}) => {
	const writes: Captured[] = [];
	let account: Record<string, unknown> | null = null;
	let verification: Record<string, unknown> | null = {
		verificationId: 'private-verification',
		state: 'pendingConfirmation',
		candidateUsername: 'alice_makes',
		expiresAt: '2027-01-01T12:00:00Z'
	};
	await page.route(
		`**${appview}/migrations/instagram/verifications/current`,
		(route) => readJSON(route, { verification })
	);
	await page.route(
		`**${appview}/migrations/instagram/account`,
		async (route) => {
			if (route.request().method() === 'DELETE') {
				writes.push({
					method: 'DELETE',
					body: {},
					headers: route.request().headers()
				});
				account = null;
				return route.fulfill({ status: 204 });
			}
			await readJSON(route, { integrationAvailable: true, account });
		}
	);
	await page.route(
		`**${appview}/migrations/instagram/verifications/private-verification/confirm`,
		async (route) => {
			writes.push(capture(route));
			account = {
				username: 'alice_makes',
				state: 'active',
				discoverable: route.request().postDataJSON().discoverable,
				verifiedAt: '2026-11-01T12:00:00Z'
			};
			verification = null;
			await readJSON(route, { account });
		}
	);
	await page.route(
		`**${appview}/migrations/instagram/settings`,
		async (route) => {
			writes.push(capture(route));
			await route.fulfill({
				status: 400,
				json: {
					error: 'invalid_request',
					message: 'Discovery preference was not saved.'
				}
			});
		}
	);
	await page.goto('/app/settings/instagram');
	await page.getByRole('button', { name: 'Keep private', exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Keep private', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	await expect(
		page.getByText(
			'Your Instagram account remains verified, but CraftSky will not match it with people who imported your username.',
			{ exact: true }
		)
	).toBeVisible();
	await page
		.getByRole('button', {
			name: 'Confirm this is my Instagram account',
			exact: true
		})
		.click();
	const discoverable = page.getByRole('switch', {
		name: 'Let others find me by my Instagram username',
		exact: true
	});
	await expect(discoverable).not.toBeChecked();
	expect(writes[0].body).toEqual({ discoverable: false });
	await discoverable.click();
	await expect(page.getByRole('alert')).toContainText(
		'Discovery preference was not saved.'
	);
	await expect(discoverable).not.toBeChecked();

	let confirmation = '';
	page.once('dialog', async (dialog) => {
		confirmation = dialog.message();
		await dialog.accept();
	});
	await page
		.getByRole('button', { name: 'Revoke Instagram verification', exact: true })
		.click();
	await expect(
		page
			.getByRole('status')
			.filter({ hasText: 'Instagram verification revoked.' })
	).toBeVisible();
	expect(confirmation).toContain('deletes your imported handles');
	expect(confirmation).toContain(
		'Existing CraftSky follows will not be affected'
	);
	expect(writes.at(-1)?.method).toBe('DELETE');
	expect(
		writes.every((value) => value.headers['x-craftsky-account-id'] === alice.id)
	).toBe(true);
});
