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

for (const [path, label, action] of [
	[
		'/compose/reply?did=did%3Aplc%3Aalice&rkey=first',
		'Write your comment',
		'Comment'
	],
	['/compose', 'What would you like to share?', 'Post']
]) {
	test(`empty composer text is a field error: ${label}`, async ({ page }) => {
		let writes = 0;
		await page.route('**/api/appview/v1/posts', (route) => {
			if (route.request().method() === 'POST') writes++;
			return route.fulfill({
				status: 500,
				json: { error: 'unexpected_write' }
			});
		});
		await page.goto(path);
		const input = page.getByRole('textbox', { name: label, exact: true });
		await expect(input).toBeVisible();
		await page.getByRole('button', { name: action, exact: true }).click();
		await expect(input).toHaveAttribute('aria-invalid', 'true');
		await expect(input).toBeFocused();
		const invalidColor = await page
			.locator('.field-error')
			.evaluate((node) => getComputedStyle(node).color);

		await expect(input).toHaveCSS('outline-color', invalidColor);
		await expect(page.locator('.field-error')).toContainText('Add some text');
		await expect(page.locator('.composer-error')).toHaveCount(0);
		const described = await input.getAttribute('aria-describedby');
		expect(described).toContain('-hint');
		expect(described).toContain('field-error-');
		await input.fill(' ');
		await expect(input).toHaveAttribute('aria-invalid', 'true');
		await input.fill('A corrected comment');
		await expect(input).not.toHaveAttribute('aria-invalid', 'true');
		await expect(page.locator('.field-error')).toHaveCount(0);
		expect(writes).toBe(0);
	});
}
test('required and custom URL errors stay with the profile link fields', async ({
	page
}) => {
	await page.goto('/profile/edit');
	await page.getByRole('button', { name: 'Add link', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Add link', exact: true });
	const input = dialog.getByRole('textbox', { name: 'URL', exact: true });
	await dialog.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(input).toHaveAttribute('aria-invalid', 'true');
	await expect(dialog.locator('.field-error')).toHaveCount(1);
	await input.fill('http://example.com');
	await dialog.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(input).toHaveAttribute('aria-invalid', 'true');
	await expect(dialog.locator('.field-error')).toContainText('HTTPS');
	await input.fill('https://example.com');
	await dialog.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog).not.toBeVisible();
});
test('business country validation uses its field rather than the page error', async ({
	page
}) => {
	await page.goto('/settings/business');
	const country = page.getByLabel('Country code', { exact: true });
	await country.fill('X');
	await page
		.getByRole('button', { name: 'Save business profile', exact: true })
		.click();
	await expect(country).toHaveAttribute('aria-invalid', 'true');
	await expect(page.locator('.field-error')).toContainText('two-letter');
	await country.fill('GB');
	await expect(country).not.toHaveAttribute('aria-invalid', 'true');
});
test('event native required fields all show inline errors before any write', async ({
	page
}) => {
	let writes = 0;
	await page.route('**/api/appview/v1/events', (route) => {
		if (route.request().method() === 'POST') writes++;
		return route.continue();
	});
	await page.goto('/settings/events');
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	const dialog = page.locator('.event-dialog');
	await dialog.getByRole('button', { name: 'Save event', exact: true }).click();
	await expect(
		dialog.getByLabel('Event name', { exact: true })
	).toHaveAttribute('aria-invalid', 'true');
	await expect(dialog.locator('.field-error').first()).toBeVisible();
	expect(writes).toBe(0);
});
test('server field errors are attached to profile inputs and service failures remain general errors', async ({
	page
}) => {
	let failure = 'field';
	await page.route('**/api/appview/v1/profiles/me', async (route) => {
		if (route.request().method() !== 'PUT') return route.continue();
		await route.fulfill({
			status: failure === 'field' ? 400 : 503,
			json:
				failure === 'field'
					? {
							error: 'validation_error',
							message: 'Invalid profile',
							fields: { displayName: 'Choose another display name.' }
						}
					: { error: 'unavailable', message: 'Temporarily unavailable' }
		});
	});
	await page.goto('/profile/edit');
	const name = page.getByRole('textbox', { name: 'Display name', exact: true });
	await name.fill('Changed name');
	await page.getByRole('button', { name: 'Save profile', exact: true }).click();
	await expect(name).toHaveAttribute('aria-invalid', 'true');
	await expect(name).toBeFocused();
	await expect(page.locator('.field-error')).toContainText(
		'Choose another display name.'
	);
	failure = 'service';
	await name.fill('Corrected name');
	await page.getByRole('button', { name: 'Save profile', exact: true }).click();
	await expect(name).not.toHaveAttribute('aria-invalid', 'true');
	await expect(
		page
			.getByRole('alert')
			.filter({ hasText: 'The action is still being confirmed' })
			.first()
	).toBeVisible();
});
test('empty sign-in marks the handle without starting authentication', async ({
	page,
	context
}) => {
	await context.clearCookies();
	let requests = 0;
	await page.route('**/api/auth/start', (route) => {
		requests++;
		return route.fulfill({ status: 500, json: {} });
	});
	await page.goto('/sign-in');
	const input = page.getByRole('combobox', {
		name: 'Your Atmosphere Handle',
		exact: true
	});
	await expect(input).toBeEnabled();
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(input).toHaveAttribute('aria-invalid', 'true');
	await expect(input).toBeFocused();
	expect(requests).toBe(0);
});
