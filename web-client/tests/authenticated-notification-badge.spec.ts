import { test, expect } from '@playwright/test';
import { resolve } from 'node:path';
const fixture = 'http://127.0.0.1:19847';
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const otherAccount = {
	id: 'c6dc7d6a-4bff-45f0-87ee-d42634646546',
	did: 'did:plc:bob',
	handle: 'bob.test',
	token: Buffer.alloc(32, 2).toString('base64url')
};
const subscription = '00000000-0000-4000-8000-000000000040';
const chatReference = '00000000-0000-4000-8000-000000000041';
const nextChatReference = '00000000-0000-4000-8000-000000000042';
test.beforeEach(async ({ page, context }) => {
	await fetch(fixture + '/__test/reset');
	await fetch(fixture + '/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ visual: true })
	});
	await page.route('**/api/service-status', (route) =>
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

for (const width of [390, 1280]) {
	test(`notification profile links and row destinations stay separate at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 900 });
		await page.goto('/notifications');
		const row = page.locator('.notification').first();
		const avatar = row.locator('.notification-avatar');
		const name = row.locator('.notification-title a');
		const profile = await avatar.getAttribute('href');
		const destination = await row
			.locator('.notification-destination')
			.getAttribute('href');
		expect(profile).toMatch(/^\/profiles\//);
		expect(destination).toMatch(/^\/posts\//);
		await expect(name).toHaveAttribute('href', profile!);

		await avatar.click();
		await expect(page).toHaveURL(new RegExp(profile! + '$'));
		await page.goto('/notifications');
		await name.click();
		await expect(page).toHaveURL(new RegExp(profile! + '$'));
		await page.goto('/notifications');
		await name.focus();
		await page.keyboard.press('Enter');
		await expect(page).toHaveURL(new RegExp(profile! + '$'));

		for (const target of ['.notification-preview', 'time']) {
			await page.goto('/notifications');
			// Click the visible content's coordinates: the row link covers this area.
			const bounds = await row.locator(target).boundingBox();
			expect(bounds).not.toBeNull();
			await page.mouse.click(
				bounds!.x + bounds!.width / 2,
				bounds!.y + bounds!.height / 2
			);
			await expect(page).toHaveURL(new RegExp(destination! + '$'));
		}
		await page.goto('/notifications');
		await row.click({ position: { x: 4, y: 4 } });
		await expect(page).toHaveURL(new RegExp(destination! + '$'));

		await page.goto('/notifications');
		const followRow = page
			.locator('.notification')
			.filter({ has: page.locator('.follow-notification') });
		await followRow
			.getByRole('button', { name: 'Follow', exact: true })
			.click();
		await expect(
			followRow.getByRole('button', { name: 'Unfollow', exact: true })
		).toBeVisible();
		await expect(page).toHaveURL(/\/notifications$/);
	});

	test(`notifications clear their badge immediately and ignore stale polls at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 900 });
		let count = 3;
		let releaseList!: () => void;
		let releaseSeen!: () => void;
		const listHeld = new Promise<void>((resolve) => (releaseList = resolve));
		const seenHeld = new Promise<void>((resolve) => (releaseSeen = resolve));
		await page.route('**/api/appview/v1/notifications/new-count', (route) =>
			route.fulfill({ json: { newCount: count } })
		);
		await page.route(
			/\/api\/appview\/v1\/notifications(?:\?.*)?$/,
			async (route) => {
				await listHeld;
				await route.continue();
			}
		);
		await page.route('**/api/appview/v1/notifications/seen', async (route) => {
			await seenHeld;
			count = 0;
			await route.continue();
		});
		await page.clock.install();
		await page.goto('/feed');
		const badge = page.locator(
			width < 900
				? '#bottom-activity-count'
				: '.app-sidebar a[href="/notifications"] .count-badge'
		);
		await expect(badge).toHaveText('3');
		const nav = page.getByRole('navigation', {
			name: width < 900 ? 'Main tabs' : 'App navigation',
			exact: true
		});
		await nav
			.getByRole('link', { name: width < 900 ? 'Activity' : /^Notifications/ })
			.click();
		await expect(page).toHaveURL(/notifications$/);
		await expect(badge).toHaveCount(0);
		const seenRequest = page.waitForRequest(
			'**/api/appview/v1/notifications/seen'
		);
		releaseList();
		await seenRequest;
		const poll = page.waitForResponse(
			'**/api/appview/v1/notifications/new-count'
		);
		await page.clock.fastForward(30001);
		await poll;
		await expect(badge).toHaveCount(0);
		await nav.getByRole('link', { name: 'Home', exact: true }).click();
		await expect(badge).toHaveCount(0);
		const reconciled = page.waitForResponse(
			'**/api/appview/v1/notifications/new-count'
		);
		releaseSeen();
		await reconciled;
		await expect(badge).toHaveCount(0);
		count = 1;
		await page.clock.fastForward(30001);
		await expect(badge).toHaveText('1');
	});
}

test('a failed seen acknowledgement restores and reconciles the badge', async ({
	page
}) => {
	let release!: () => void;
	const held = new Promise<void>((resolve) => (release = resolve));
	await page.route('**/api/appview/v1/notifications/new-count', (route) =>
		route.fulfill({ json: { newCount: 3 } })
	);
	await page.route('**/api/appview/v1/notifications/seen', async (route) => {
		await held;
		await route.fulfill({ status: 503, json: { error: 'unavailable' } });
	});
	await page.goto('/feed');
	const badge = page.locator(
		'.app-sidebar a[href="/notifications"] .count-badge'
	);
	await expect(badge).toHaveText('3');
	await page
		.getByRole('navigation', { name: 'App navigation', exact: true })
		.getByRole('link', { name: /^Notifications/ })
		.click();
	await expect(badge).toHaveCount(0);
	release();
	await expect(badge).toHaveText('3');
	await expect(
		page.getByRole('heading', { name: 'Notifications', exact: true })
	).toBeVisible();
});

test('retrying a failed first page acknowledges the recovered notifications', async ({
	page
}) => {
	await page.route(
		/\/api\/appview\/v1\/notifications(?:\?.*)?$/,
		(route) => route.fulfill({ status: 503, json: { error: 'unavailable' } }),
		{ times: 1 }
	);
	const acknowledgements: string[] = [];
	page.on('request', (request) => {
		if (request.url().endsWith('/v1/notifications/seen'))
			acknowledgements.push(request.method());
	});
	await page.goto('/notifications');
	const retry = page.getByRole('button', { name: 'Retry', exact: true });
	await expect(retry).toBeVisible();
	expect(acknowledgements).toEqual([]);
	await retry.click();
	await expect(page.locator('.notification').first()).toBeVisible();
	await expect.poll(() => acknowledgements).toEqual(['POST']);
});

test('returning from a notification restores loaded pages and scroll position', async ({
	page
}) => {
	await page.route(
		/\/api\/appview\/v1\/notifications(?:\?.*)?$/,
		async (route) => {
			const url = new URL(route.request().url());
			const later = url.searchParams.has('cursor');
			url.searchParams.delete('cursor');
			const response = await route.fetch({ url: url.href });
			const { items } = await response.json();
			const pageItems = Array.from({ length: 15 }, (_, index) => ({
				...items[0],
				id: `${later ? 'later' : 'first'}-${index}`
			}));
			await route.fulfill({
				json: { items: pageItems, ...(later ? {} : { cursor: 'another-page' }) }
			});
		}
	);
	await page.goto('/notifications');
	await expect(page.locator('.notification')).toHaveCount(15);
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(page.locator('.notification')).toHaveCount(30);
	const profile = page.locator('.notification-avatar').nth(20);
	await profile.scrollIntoViewIfNeeded();
	const scroll = await page.evaluate(() => window.scrollY);
	const actor = await profile.getAttribute('href');
	expect(scroll).toBeGreaterThan(0);
	await profile.click();
	await expect(page).toHaveURL(new RegExp(actor! + '$'));
	await page.goBack();
	await expect(page.locator('.notification')).toHaveCount(30);
	await expect
		.poll(() => page.evaluate(() => window.scrollY))
		.toBeCloseTo(scroll, 0);
});

test('the activity badge updates while the chat count request is pending', async ({
	page
}) => {
	let release!: () => void;
	const held = new Promise<void>((resolve) => (release = resolve));
	await page.route('**/api/appview/v1/chat/unread-counts', async (route) => {
		await held;
		await route.fulfill({ json: { accepted: 0, requests: 0 } });
	});
	await page.route('**/api/appview/v1/notifications/new-count', (route) =>
		route.fulfill({ json: { newCount: 3 } })
	);
	await page.goto('/feed');
	try {
		await expect(
			page.locator('.app-sidebar a[href="/notifications"] .count-badge')
		).toHaveText('3');
	} finally {
		release();
	}
});

test('a push for another account switches before listing or acknowledging activity', async ({
	page,
	context
}) => {
	await context.addCookies([
		{
			name: 'craftsky_account_' + otherAccount.id,
			value: Buffer.from(JSON.stringify(otherAccount)).toString('base64url'),
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	await page.addInitScript(
		({ accountId, subscriptionId }) =>
			localStorage.setItem(
				'craftsky.push.bindings',
				JSON.stringify([{ accountId, subscriptionId, enabled: true }])
			),
		{ accountId: otherAccount.id, subscriptionId: subscription }
	);
	const loaded: string[] = [];
	const seen: string[] = [];
	page.on('request', (request) => {
		const path = new URL(request.url()).pathname;
		const id = request.headers()['x-craftsky-account-id'];
		if (path === '/api/appview/v1/notifications') loaded.push(id);
		if (path === '/api/appview/v1/notifications/seen') seen.push(id);
	});
	let release!: () => void;
	const held = new Promise<void>((resolve) => (release = resolve));
	await page.route('**/api/auth/switch', async (route) => {
		await held;
		await route.continue();
	});
	await page.goto('/notifications?pushSubscription=' + subscription);
	try {
		await expect(page.getByText('Opening notification…')).toBeVisible();
		expect(loaded).toEqual([]);
		expect(seen).toEqual([]);
	} finally {
		release();
	}
	await expect(page).toHaveURL(/\/notifications$/);
	await expect(page.locator('.notification').first()).toBeVisible();
	await expect.poll(() => seen).toEqual([otherAccount.id]);
	expect(loaded).toEqual([otherAccount.id]);
});

test('a bound chat push opens the resolved conversation for the active account', async ({
	page
}) => {
	await page.addInitScript(
		({ accountId, subscriptionId }) =>
			localStorage.setItem(
				'craftsky.push.bindings',
				JSON.stringify([{ accountId, subscriptionId, enabled: true }])
			),
		{ accountId: account.id, subscriptionId: subscription }
	);
	await page.route(
		'**/api/appview/v1/chat/notification-opens/' + chatReference,
		(route) => {
			expect(route.request().headers()['x-craftsky-account-id']).toBe(
				account.id
			);
			return route.fulfill({
				json: { type: 'conversation', conversationId: 'push-chat' }
			});
		}
	);
	await page.goto(
		`/messages?pushSubscription=${subscription}&pushReference=${chatReference}`
	);
	await expect(page).toHaveURL(/\/messages\/conversations\/push-chat$/);
});

test('an unbound push does not resolve private chat under the active account', async ({
	page
}) => {
	const opens: string[] = [];
	page.on('request', (request) => {
		if (request.url().includes('/chat/notification-opens/'))
			opens.push(request.url());
	});
	await page.goto(
		`/messages?pushSubscription=${subscription}&pushReference=${chatReference}`
	);
	await expect(page).toHaveURL(/\/feed$/);
	await expect(
		page.getByText('Sign in to the account for this notification to open it.')
	).toBeVisible();
	expect(opens).toEqual([]);
});

test('a newer chat push replaces a pending lookup in the same window', async ({
	page
}) => {
	await page.addInitScript(
		({ accountId, subscriptionId }) =>
			localStorage.setItem(
				'craftsky.push.bindings',
				JSON.stringify([{ accountId, subscriptionId, enabled: true }])
			),
		{ accountId: account.id, subscriptionId: subscription }
	);
	let release!: () => void;
	const held = new Promise<void>((resolve) => (release = resolve));
	await page.route(
		'**/api/appview/v1/chat/notification-opens/' + chatReference,
		async (route) => {
			await held;
			// The first fetch can be canceled while the route fixture is holding it.
			await route
				.fulfill({
					json: { type: 'conversation', conversationId: 'older-chat' }
				})
				.catch(() => {});
		}
	);
	await page.route(
		'**/api/appview/v1/chat/notification-opens/' + nextChatReference,
		(route) =>
			route.fulfill({
				json: { type: 'conversation', conversationId: 'newer-chat' }
			})
	);
	const firstLookup = page.waitForRequest(
		'**/api/appview/v1/chat/notification-opens/' + chatReference
	);
	await page.goto(
		`/messages?pushSubscription=${subscription}&pushReference=${chatReference}`
	);
	try {
		await firstLookup;
		await page.evaluate((url) => {
			const anchor = document.createElement('a');
			anchor.href = url;
			document.body.append(anchor);
			anchor.click();
			anchor.remove();
		}, `/messages?pushSubscription=${subscription}&pushReference=${nextChatReference}`);
		await expect(page).toHaveURL(/\/messages\/conversations\/newer-chat$/);
	} finally {
		release();
	}
	await expect(page).toHaveURL(/\/messages\/conversations\/newer-chat$/);
});

test('a malformed notification link never loads or acknowledges Activity', async ({
	page
}) => {
	const activity: string[] = [];
	page.on('request', (request) => {
		const path = new URL(request.url()).pathname;
		if (/\/v1\/notifications(?:\/seen)?$/.test(path)) activity.push(path);
	});
	await page.goto(
		`/notifications?pushSubscription=${subscription}&pushReference=invalid`
	);
	await expect(page).toHaveURL(/\/feed$/);
	await expect(
		page.getByText('This notification link is invalid.')
	).toBeVisible();
	expect(activity).toEqual([]);
});

test('resolving a chat push reselects its account after a different tab changes it', async ({
	page,
	context
}) => {
	await context.addCookies([
		{
			name: 'craftsky_account_' + otherAccount.id,
			value: Buffer.from(JSON.stringify(otherAccount)).toString('base64url'),
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	await page.addInitScript(
		({ accountId, subscriptionId }) =>
			localStorage.setItem(
				'craftsky.push.bindings',
				JSON.stringify([{ accountId, subscriptionId, enabled: true }])
			),
		{ accountId: account.id, subscriptionId: subscription }
	);
	await page.route(
		'**/api/appview/v1/chat/notification-opens/' + chatReference,
		async (route) => {
			expect(route.request().headers()['x-craftsky-account-id']).toBe(
				account.id
			);
			await context.addCookies([
				{
					name: 'craftsky_active',
					value: otherAccount.id,
					url: 'http://127.0.0.1:19846',
					httpOnly: true,
					sameSite: 'Lax'
				}
			]);
			await route.fulfill({
				json: { type: 'conversation', conversationId: 'push-chat' }
			});
		}
	);
	await page.goto(
		`/messages?pushSubscription=${subscription}&pushReference=${chatReference}`
	);
	await expect(page).toHaveURL(/\/messages\/conversations\/push-chat$/);
	const active = (await context.cookies()).find(
		(cookie) => cookie.name === 'craftsky_active'
	);
	expect(active?.value).toBe(account.id);
});
