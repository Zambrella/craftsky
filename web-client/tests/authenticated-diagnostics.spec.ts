import { test, expect } from '@playwright/test';

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
	await page.goto('/app/settings/privacy');
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
	await page.goto('/app/home');
	await expect.poll(() => envelopes.length).toBe(1);
	const wire = envelopes[0];
	expect(wire).toContain('"httpStatus":503');
	expect(wire).toContain('"appViewRequestId":"' + id + '"');
	expect(wire).toContain('"operation":"workspace"');
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
	await page.goto('/app/home');
	await expect(
		page.getByText('Fixture failure', { exact: true })
	).toBeVisible();
	expect(reports).toBe(0);
	status = 500;
	await page.reload();
	await expect.poll(() => reports).toBe(1);
	await expect(
		page.getByText('Fixture failure', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Retry', exact: true })
	).toBeEnabled();
});
