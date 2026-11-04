import { test, expect } from '@playwright/test';
import { resolve } from 'node:path';
const fixture = 'http://127.0.0.1:19847';
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
test.beforeEach(async ({ page, context }) => {
	await fetch(fixture + '/__test/reset');
	await fetch(fixture + '/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ visual: true })
	});
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
		await page.goto('/app/home');
		const badge = page.locator(
			width < 900
				? '#bottom-activity-count'
				: '.app-sidebar a[href="/app/activity"] .count-badge'
		);
		await expect(badge).toHaveText('3');
		const nav = page.getByRole('navigation', {
			name: width < 900 ? 'Main tabs' : 'App navigation',
			exact: true
		});
		await nav
			.getByRole('link', { name: width < 900 ? 'Activity' : /^Notifications/ })
			.click();
		await expect(page).toHaveURL(/app\/activity$/);
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
	await page.goto('/app/home');
	const badge = page.locator(
		'.app-sidebar a[href="/app/activity"] .count-badge'
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
