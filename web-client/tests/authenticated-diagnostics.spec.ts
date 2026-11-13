import { test, expect, type Page } from '@playwright/test';

// Build with the documented loopback DSN; these tests never contact hosted Sentry.
test.skip(
	process.env.WEB_CLIENT_DIAGNOSTICS !== '1',
	'Requires the loopback diagnostic fixture build'
);
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const canary = 'PRIVATE_DIAGNOSTIC_CANARY';
async function captureEnvelopes(page: Page) {
	const envelopes: string[] = [];
	await page.addInitScript(() =>
		localStorage.setItem('craftsky.diagnostics-consent', 'true')
	);
	await page.route(
		'http://127.0.0.1:19846/api/1/envelope/**',
		async (route) => {
			envelopes.push(route.request().postData() || '');
			await route.fulfill({ status: 200, body: '{}' });
		}
	);
	return envelopes;
}
test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
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

test('browser reports require opt-in, survive reload and stop immediately after revocation', async ({
	page
}) => {
	const envelopes: string[] = [];
	await page.route(
		'http://127.0.0.1:19846/api/1/envelope/**',
		async (route) => {
			envelopes.push(route.request().postData() || '');
			await route.fulfill({ status: 200, body: '{}' });
		}
	);
	const fail = () =>
		page.evaluate((message) => {
			setTimeout(() => {
				throw new TypeError(message);
			}, 0);
		}, canary);
	await page.goto('/settings/privacy');
	const consent = page.getByRole('switch', {
		name: 'Error reporting',
		exact: true
	});
	await expect(consent).not.toBeChecked();
	await fail();
	await page.waitForTimeout(150);
	expect(envelopes).toHaveLength(0);
	await consent.check();
	await page.reload();
	await expect(consent).toBeChecked();
	await fail();
	await expect.poll(() => envelopes.length).toBe(1);
	expect(envelopes[0]).toContain('TypeError');
	expect(envelopes[0]).toContain('craftsky-web@');
	expect(envelopes[0]).not.toContain(canary);
	expect(envelopes[0]).not.toContain(account.did);
	await consent.uncheck();
	await fail();
	await page.waitForTimeout(150);
	expect(envelopes).toHaveLength(1);
});

test('a consumed AppView failure includes safe correlation while hiding private content', async ({
	page
}) => {
	const envelopes: string[] = [];
	await page.addInitScript(() =>
		localStorage.setItem('craftsky.diagnostics-consent', 'true')
	);
	await page.route(
		'http://127.0.0.1:19846/api/1/envelope/**',
		async (route) => {
			envelopes.push(route.request().postData() || '');
			await route.fulfill({ status: 200, body: '{}' });
		}
	);
	const id = '12345678-1234-4123-8123-123456789012';
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			status: 503,
			headers: { 'X-Request-ID': id },
			json: { error: 'temporarily_unavailable', message: canary }
		})
	);
	await page.goto('/feed');
	await expect.poll(() => envelopes.length).toBe(1);
	const wire = envelopes[0];
	expect(wire).toContain('"httpStatus":503');
	expect(wire).toContain('"appViewRequestId":"' + id + '"');
	expect(wire).toContain('"operation":"social.Feed.load"');
	expect(wire).toContain('"debug_id"');
	expect(wire).not.toContain(canary);
	expect(wire).not.toContain('feed/timeline');
	expect(wire).not.toContain(account.did);
	await expect(
		page.getByRole('button', { name: 'Retry', exact: true })
	).toBeVisible();
});

test('expected API failures and reporting outages preserve the normal recovery UI', async ({
	page
}) => {
	let reports = 0;
	await page.addInitScript(() =>
		localStorage.setItem('craftsky.diagnostics-consent', 'true')
	);
	await page.route(
		'http://127.0.0.1:19846/api/1/envelope/**',
		async (route) => {
			reports++;
			await route.abort();
		}
	);
	let status = 403;
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			status,
			json: {
				error: status === 403 ? 'not_allowed' : 'internal_error',
				message: 'Fixture failure'
			}
		})
	);
	await page.goto('/feed');
	await expect(
		page.getByText("You don't have permission to do that.", { exact: true })
	).toBeVisible();
	expect(reports).toBe(0);
	status = 500;
	await page.reload();
	await expect.poll(() => reports).toBe(1);
	await expect(
		page.getByText('CraftSky is having trouble right now. Please try again.', {
			exact: true
		})
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Retry', exact: true })
	).toBeEnabled();
});

test('failed settings sign-out retains HTTP diagnostics and a safe recovery message', async ({
	page
}) => {
	const envelopes = await captureEnvelopes(page);
	const id = '12345678-1234-4123-8123-123456789012';
	await page.route('**/api/auth/logout', (route) =>
		route.fulfill({
			status: 503,
			json: { error: 'upstream_unavailable', message: canary, requestId: id }
		})
	);
	await page.goto('/settings');
	await page.getByRole('button', { name: 'Sign out', exact: true }).click();
	await page
		.getByRole('dialog', { name: 'Sign out?', exact: true })
		.getByRole('button', { name: 'Sign out', exact: true })
		.click();
	await expect(
		page.getByText('CraftSky is having trouble right now. Please try again.', {
			exact: true
		})
	).toBeVisible();
	await expect.poll(() => envelopes.length).toBe(1);
	expect(envelopes[0]).toContain(
		'"operation":"settings.SettingsWorkspace.signOut"'
	);
	expect(envelopes[0]).toContain('"httpStatus":503');
	expect(envelopes[0]).toContain('"httpMethod":"POST"');
	expect(envelopes[0]).toContain('"appViewRequestId":"' + id + '"');
	expect(envelopes[0]).not.toContain(canary);
	expect(envelopes[0]).not.toContain(account.id);
	expect(envelopes[0]).not.toContain(account.did);
	await expect(page).toHaveURL('/settings');
});

test('failed account switches report their HTTP cause while keeping the current account usable', async ({
	page,
	context
}) => {
	const envelopes = await captureEnvelopes(page);
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
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	await page.route('**/api/auth/switch', (route) =>
		route.fulfill({
			status: 503,
			json: { error: 'upstream_unavailable', message: canary }
		})
	);
	await page.goto('/feed');
	await page.getByRole('button', { name: /^Switch accounts,/ }).click();
	await page
		.locator('.account-menu .person-row')
		.filter({ hasText: '@bob.test' })
		.click();
	await expect(
		page.getByText('The account could not be switched. Try again.', {
			exact: true
		})
	).toBeVisible();
	await expect.poll(() => envelopes.length).toBe(1);
	expect(envelopes[0]).toContain('"operation":"account.switch"');
	expect(envelopes[0]).toContain('"httpStatus":503');
	expect(envelopes[0]).not.toContain(canary);
	expect(envelopes[0]).not.toContain(bob.id);
	expect(envelopes[0]).not.toContain(account.did);
	await expect(
		page.getByRole('button', {
			name: 'Switch accounts, @alice.test',
			exact: true
		})
	).toBeVisible();
});

test('optional startup identity failures retain their cause without blocking the feed', async ({
	page
}) => {
	const envelopes = await captureEnvelopes(page);
	let profileRequests = 0;
	await page.route('**/api/appview/v1/profiles/me', (route) => {
		// Startup's optional identity read precedes the feed's independent read.
		// Fail only the former so this verifies one physical failure and owner.
		if (++profileRequests > 1) return route.continue();
		return route.fulfill({
			status: 503,
			json: { error: 'upstream_unavailable', message: canary }
		});
	});
	await page.goto('/feed');
	await expect(
		page.getByText("Couldn't load your crafts.", { exact: true })
	).toBeVisible();
	await expect(page.locator('.post-card').first()).toBeVisible();
	expect(profileRequests).toBe(2);
	await expect.poll(() => envelopes.length).toBe(1);
	expect(envelopes[0]).toContain('"operation":"startup.identity"');
	expect(envelopes[0]).toContain('"httpStatus":503');
	for (const wire of envelopes) {
		expect(wire).not.toContain(canary);
		expect(wire).not.toContain(account.id);
		expect(wire).not.toContain(account.did);
	}
});

test('repeated badge failures report once until a successful refresh', async ({
	page
}) => {
	const envelopes = await captureEnvelopes(page);
	await page.clock.install();
	let calls = 0,
		failing = true;
	await page.route('**/api/appview/v1/notifications/new-count', (route) => {
		calls++;
		return route.fulfill({
			status: failing ? 503 : 200,
			json: failing
				? { error: 'upstream_unavailable', message: canary + calls }
				: { newCount: 0 }
		});
	});
	await page.goto('/feed');
	await expect.poll(() => envelopes.length).toBe(1);
	await page.clock.fastForward(30_000);
	await expect.poll(() => calls).toBe(2);
	failing = false;
	await page.clock.fastForward(30_000);
	await expect.poll(() => calls).toBe(3);
	failing = true;
	await page.clock.fastForward(30_000);
	await expect.poll(() => envelopes.length).toBe(2);
	for (const wire of envelopes) {
		expect(wire).toContain('"operation":"startup.count.activity"');
		expect(wire).not.toContain(canary);
	}
	await expect(page.locator('.post-card').first()).toBeVisible();
});
