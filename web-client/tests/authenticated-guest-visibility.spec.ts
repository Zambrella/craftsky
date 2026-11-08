import { test, expect } from '@playwright/test';
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const label = 'Ask apps and sites not to show my account to logged-out users';
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
test('AT-008 shared preference saves through the authenticated proxy and survives reopening', async ({
	page
}) => {
	await page.goto('/settings/privacy');
	const toggle = page.getByRole('switch', { name: label });
	await expect(toggle).not.toBeChecked();
	await toggle.click();
	await expect(
		page
			.getByRole('status')
			.filter({ hasText: 'Saved. This preference also applies on Bluesky.' })
	).toBeVisible();
	await page.reload();
	await expect(toggle).toBeChecked();
	await expect(
		page.getByText(/It doesn't make your account private/)
	).toBeVisible();
});
test('AT-008 read failure stays unknown until an authoritative retry', async ({
	page
}) => {
	let reads = 0;
	await page.route(
		'**/api/appview/v1/preferences/logged-out-visibility',
		(route) =>
			route.fulfill(
				++reads === 1
					? {
							status: 503,
							json: { error: 'unavailable', message: 'Unavailable' }
						}
					: { json: { hideFromLoggedOut: true, expectedCid: 'current' } }
			)
	);
	await page.goto('/settings/privacy');
	await expect(
		page.getByRole('button', { name: 'Reload preference' })
	).toBeVisible();
	await expect(page.getByRole('switch', { name: label })).toHaveCount(0);
	await page.getByRole('button', { name: 'Reload preference' }).click();
	await expect(page.getByRole('switch', { name: label })).toBeChecked();
});
test('AT-008 pending saves display the confirmed value and retry the original command', async ({
	page
}) => {
	const writes: { body: unknown; key: string | undefined }[] = [];
	await page.route(
		'**/api/appview/v1/preferences/logged-out-visibility',
		async (route) => {
			if (route.request().method() === 'GET')
				return route.fulfill({
					json: { hideFromLoggedOut: false, expectedCid: 'current' }
				});
			writes.push({
				body: route.request().postDataJSON(),
				key: route.request().headers()['idempotency-key']
			});
			return route.fulfill(
				writes.length === 1
					? {
							status: 202,
							headers: { 'Retry-After': '1' },
							json: { state: 'pending' }
						}
					: { json: { hideFromLoggedOut: true, expectedCid: 'saved' } }
			);
		}
	);
	await page.goto('/settings/privacy');
	const toggle = page.getByRole('switch', { name: label });
	await expect(toggle).toBeEnabled();
	await toggle.click();
	await expect(page.getByText('Save pending confirmation.')).toBeVisible();
	await expect(toggle).not.toBeChecked();
	await expect(toggle).toBeDisabled();
	await page.getByRole('button', { name: 'Retry save', exact: true }).click();
	await expect(toggle).toBeChecked();
	await expect(toggle).toBeEnabled();
	expect(writes).toHaveLength(2);
	expect(writes[0]).toEqual(writes[1]);
});
