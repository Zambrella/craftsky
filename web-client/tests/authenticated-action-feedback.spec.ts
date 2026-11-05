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

test('published confirmation survives navigation and clears when dismissed', async ({
	page
}) => {
	await page.goto('/compose');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('A new project for today');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).not.toHaveURL(/compose/);
	const toast = page.locator('.snackbar');
	await expect(toast).toContainText('Your post has been published.');
	await expect(toast).toHaveAttribute('role', 'status');
	await page
		.getByRole('navigation', { name: 'App navigation', exact: true })
		.getByRole('link', { name: 'Home', exact: true })
		.click();
	await expect(page).toHaveURL(/feed$/);
	await expect(toast).toContainText('Your post has been published.');
	await toast.getByRole('button', { name: 'Dismiss notification' }).click();
	await expect(toast).toHaveCount(0);
});

test('successful actions queue one toast at a time and pause while hovered', async ({
	page
}) => {
	await page.goto('/settings/business');
	await page.getByLabel('Tagline', { exact: true }).fill('Hand dyed wool');
	await page.clock.install();
	await page.getByRole('button', { name: 'Save business profile' }).click();
	const toast = page.locator('.snackbar');
	await expect(toast).toContainText('Business profile saved.');
	await page.evaluate(() =>
		window.dispatchEvent(
			new CustomEvent('craftsky-toast', {
				detail: { message: 'Another action completed.' }
			})
		)
	);
	await expect(toast).toHaveCount(1);
	await toast.hover();
	await page.clock.fastForward(10000);
	await expect(toast).toContainText('Business profile saved.');
	await page.mouse.move(0, 0);
	await page.clock.fastForward(4100);
	await expect(toast).toContainText('Another action completed.');
	await page.clock.fastForward(4100);
	await expect(toast).toHaveCount(0);
});

test('errors persist and dismissing a toast retains uncertain-write recovery', async ({
	page
}) => {
	let failing = true;
	const writes: { body: string | null; key: string | undefined }[] = [];
	await page.route('**/api/appview/v1/profiles/me/business', (route) => {
		if (route.request().method() !== 'PUT') return route.continue();
		writes.push({
			body: route.request().postData(),
			key: route.request().headers()['idempotency-key']
		});
		return failing
			? route.fulfill({
					status: 503,
					json: { error: 'unavailable', message: 'Could not confirm the save.' }
				})
			: route.continue();
	});
	await page.goto('/settings/business');
	await page.getByLabel('Tagline', { exact: true }).fill('Hand dyed wool');
	await page.clock.install();
	await page.getByRole('button', { name: 'Save business profile' }).click();
	const toast = page.locator('.snackbar');
	await expect(toast).toHaveAttribute('role', 'alert');
	await page.clock.fastForward(10000);
	await expect(toast).toBeVisible();
	await toast.getByRole('button', { name: 'Dismiss notification' }).click();
	await expect(toast).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Save business profile' })
	).toBeDisabled();
	failing = false;
	await page
		.getByRole('button', { name: 'Check and retry action', exact: true })
		.click();
	await expect(toast).toContainText('Business profile saved.');
	expect(writes).toHaveLength(2);
	expect(writes[1]).toEqual(writes[0]);
});

test('action errors in a native modal are visible, interactive and not duplicated', async ({
	page
}) => {
	await page.route('**/api/appview/v1/profiles/me/business', (route) =>
		route.request().method() === 'PUT'
			? route.fulfill({
					status: 503,
					json: {
						error: 'unavailable',
						message: 'Could not confirm the product save.'
					}
				})
			: route.continue()
	);
	await page.goto('/settings/products');
	await page
		.getByRole('button', { name: 'Edit Blue wool', exact: true })
		.click();
	const dialog = page.locator('.product-dialog');
	await dialog.getByLabel('Title', { exact: true }).fill('Blue hand dyed wool');
	await dialog
		.getByRole('button', { name: 'Save product', exact: true })
		.click();
	await expect(page.locator('.snackbar')).toHaveCount(1);
	await expect(dialog.locator('.snackbar')).toHaveAttribute('role', 'alert');
	await dialog
		.locator('.snackbar')
		.getByRole('button', { name: 'Dismiss notification' })
		.click();
	await expect(page.locator('.snackbar')).toHaveCount(0);
	await expect(
		dialog.getByRole('button', { name: 'Check and retry action' })
	).toBeEnabled();
});

test('account switching clears queued feedback from the previous account', async ({
	page,
	context
}) => {
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
	await page.goto('/settings/business');
	await page.getByLabel('Tagline', { exact: true }).fill('Hand dyed wool');
	await page.getByRole('button', { name: 'Save business profile' }).click();
	await expect(page.locator('.snackbar')).toBeVisible();
	await page.getByRole('button', { name: /^Switch accounts,/ }).click();
	await page
		.locator('.account-menu .person-row')
		.filter({ hasText: '@bob.test' })
		.click();
	await expect(
		page.getByRole('button', {
			name: 'Switch accounts, @bob.test',
			exact: true
		})
	).toBeVisible();
	await expect(page.locator('.snackbar')).toHaveCount(0);
});

test('empty comments retain inline field feedback and show no action toast', async ({
	page
}) => {
	await page.goto('/compose/reply?did=did%3Aplc%3Aalice&rkey=first');
	await page.getByRole('button', { name: 'Comment', exact: true }).click();
	await expect(
		page.getByRole('textbox', { name: 'Write your comment', exact: true })
	).toHaveAttribute('aria-invalid', 'true');
	await expect(page.locator('.snackbar')).toHaveCount(0);
});

test('mobile toasts remain in the viewport above bottom navigation and pause on focus', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/settings/business');
	await page.getByLabel('Tagline', { exact: true }).fill('Hand dyed wool');
	await page.clock.install();
	await page.getByRole('button', { name: 'Save business profile' }).click();
	const toast = page.locator('.snackbar');
	await expect(toast).toBeVisible();
	const bounds = await toast.boundingBox();
	expect(bounds!.x).toBeGreaterThanOrEqual(0);
	expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390);
	expect(bounds!.y + bounds!.height).toBeLessThanOrEqual(764);
	await toast.getByRole('button', { name: 'Dismiss notification' }).focus();
	await page.mouse.move(0, 0);
	await page.clock.fastForward(10000);
	await expect(toast).toBeVisible();
	await page.getByLabel('Tagline', { exact: true }).focus();
	await page.clock.fastForward(4100);
	await expect(toast).toHaveCount(0);
});

test('dismissing a post action error preserves the original retry and operation key', async ({
	page
}) => {
	let failing = true;
	const keys: (string | undefined)[] = [];
	await page.route('**/api/appview/v1/posts/*/*/likes', (route) => {
		if (route.request().method() !== 'POST') return route.continue();
		keys.push(route.request().headers()['idempotency-key']);
		return failing
			? route.fulfill({
					status: 503,
					json: {
						error: 'unavailable',
						message: 'The like could not be confirmed.'
					}
				})
			: route.continue();
	});
	await page.goto('/feed');
	const like = page.getByRole('button', { name: /Like$/ }).first();
	await like.click();
	const toast = page.locator('.snackbar');
	await expect(toast).toHaveAttribute('role', 'alert');
	await toast.getByRole('button', { name: 'Dismiss notification' }).click();
	await expect(toast).toHaveCount(0);
	await expect(like).toBeDisabled();
	failing = false;
	await page
		.getByRole('button', { name: 'Retry the same action', exact: true })
		.click();
	await expect(like).toHaveAttribute('aria-pressed', 'true');
	expect(keys).toHaveLength(2);
	expect(keys[0]).toBeTruthy();
	expect(keys[1]).toBe(keys[0]);
});
