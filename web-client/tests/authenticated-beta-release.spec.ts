import { test, expect } from '@playwright/test';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};

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

test('beta release hides subscriptions and keeps unavailable actions local', async ({
	page
}) => {
	test.skip(
		process.env.WEB_CLIENT_SUBSCRIPTIONS_ENABLED !== 'false',
		'Runs against the native beta configuration.'
	);
	await page.goto('/settings');
	await expect(
		page.getByRole('heading', { name: 'Settings', exact: true })
	).toBeVisible();
	await expect(page.locator('.subscription-callout')).toHaveCount(0);
	await page
		.getByRole('link', { name: 'Growth is coming soon', exact: true })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Growth is coming soon', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'OK', exact: true }).click();
	await expect(page).toHaveURL(/\/settings$/);
	await page.goto('/settings/customisation');
	await expect(
		page.getByRole('heading', { name: 'Customisation is coming soon' })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Save', exact: true })
	).toHaveCount(0);
	await page.goto('/compose');
	await page
		.getByRole('button', { name: 'Scheduled posts is coming soon' })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Scheduled posts is coming soon' })
	).toBeVisible();
	await page.getByRole('button', { name: 'OK', exact: true }).click();
	await page.goto('/saved');
	await expect(page.locator('.saved-scope')).toHaveCount(0);
	await expect(
		page.getByRole('heading', { name: 'Unfiled', exact: true })
	).toHaveCount(0);
	await page
		.getByRole('button', { name: 'Folders is coming soon', exact: true })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Folders is coming soon' })
	).toBeVisible();
});
