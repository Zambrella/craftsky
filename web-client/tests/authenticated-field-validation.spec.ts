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
	await page.getByRole('button', { name: 'Save', exact: true }).click();
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
	const dialog = page.locator('.event-composer-form');
	await dialog
		.getByRole('button', { name: 'Create event', exact: true })
		.click();
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

test('event server time validation is shown at Start instead of a generic failure', async ({
	page
}) => {
	await page.route('**/api/appview/v1/events', (route) =>
		route.fulfill({
			status: 422,
			json: {
				error: 'validation_failed',
				message: 'invalid business event',
				fields: {
					startsAt: 'startsAt and endsAt must form a valid event time range'
				}
			}
		})
	);
	await page.goto('/compose/event');
	await page.getByLabel('Event name', { exact: true }).fill('Maker fair');
	await page.getByLabel('Start', { exact: true }).fill('2026-12-01T10:00');
	await page.getByLabel('End', { exact: true }).fill('2026-12-01T12:00');
	await page.getByLabel('Your role', { exact: true }).click();
	await page.getByRole('checkbox', { name: 'Organizer', exact: true }).click();
	await page.getByRole('button', { name: 'Create event', exact: true }).click();
	await expect(page.getByLabel('Start', { exact: true })).toHaveAttribute(
		'aria-invalid',
		'true'
	);
	await expect(page.getByLabel('Start', { exact: true })).toBeFocused();
	await expect(
		page.getByText('startsAt and endsAt must form a valid event time range', {
			exact: true
		})
	).toBeVisible();
	await expect(
		page.getByText("That didn't work. Please try again.", { exact: true })
	).toHaveCount(0);
});

test('publication field errors stay inline while uncertain writes retain recovery', async ({
	page
}) => {
	let uncertain = false;
	await page.route('**/api/appview/v1/posts', (route) =>
		route.fulfill({
			status: uncertain ? 503 : 422,
			json: uncertain
				? { error: 'unavailable', message: 'Temporarily unavailable' }
				: {
						error: 'validation_error',
						message: 'Invalid post',
						fields: { text: 'Update the post text before publishing.' }
					}
		})
	);
	await page.goto('/compose');
	const input = page.getByRole('textbox', {
		name: 'What would you like to share?',
		exact: true
	});
	await input.fill('A post ready to publish');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(input).toHaveAttribute('aria-invalid', 'true');
	await expect(input).toBeFocused();
	await expect(page.locator('.field-error')).toHaveText(
		'Update the post text before publishing.'
	);
	await expect(page.locator('.snackbar')).toHaveCount(0);
	await expect(input).toBeEnabled();

	uncertain = true;
	await input.fill('A corrected post ready to publish');
	await expect(input).not.toHaveAttribute('aria-invalid', 'true');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Check the same action', exact: true })
	).toBeVisible();
	await expect(input).toBeDisabled();
	await expect(
		page
			.getByRole('alert')
			.filter({ hasText: 'This action is still being confirmed.' })
	).toBeVisible();
});

test('an invalid project craft picker has only the input error border', async ({
	page
}) => {
	await page.route('**/api/appview/v1/profiles/me', async (route) => {
		const response = await route.fetch();
		await route.fulfill({
			response,
			json: { ...(await response.json()), crafts: [] }
		});
	});
	await page.goto('/compose/project');
	const craft = page.getByRole('combobox', { name: 'Primary craft required' });
	await page.getByLabel('Project description').fill('A project description');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(craft).toHaveAttribute('aria-invalid', 'true');
	await expect(craft).toBeFocused();
	const error = page.getByText('Choose a primary craft.', { exact: true });
	await expect(error).toBeVisible();
	await expect(craft).toHaveCSS(
		'outline-color',
		await error.evaluate((node) => getComputedStyle(node).color)
	);
	await expect(craft).toHaveCSS('outline-offset', '-2px');
	await expect(page.locator('.single-field').filter({ has: craft })).toHaveCSS(
		'outline-style',
		'none'
	);
});

test('custom project subtype input preserves complete UTF-8 characters within the byte limit', async ({
	page
}) => {
	await page.goto('/compose/project');
	await page.getByRole('combobox', { name: 'Primary craft required' }).click();
	await page.getByRole('option', { name: 'Sewing', exact: true }).click();
	await page
		.getByRole('button', { name: 'Sewing details', exact: true })
		.click();
	await page
		.getByRole('combobox', { name: 'Project type', exact: true })
		.fill('Other');
	await page.getByRole('option', { name: 'Other', exact: true }).click();
	const input = page.getByRole('textbox', {
		name: 'What did you make?',
		exact: true
	});
	await input.fill('🧶'.repeat(30));
	await expect(input).toHaveValue('🧶'.repeat(25));
});

test('multiple profile validation errors focus the first invalid input', async ({
	page
}) => {
	await page.goto('/profile/edit');
	const name = page.getByRole('textbox', { name: 'Display name', exact: true });
	const pronouns = page.getByRole('textbox', { name: 'Pronouns', exact: true });
	const bio = page.getByRole('textbox', { name: 'Bio', exact: true });
	await name.fill('a'.repeat(65));
	await pronouns.fill('b'.repeat(21));
	await bio.fill('c'.repeat(257));
	await page.getByRole('button', { name: 'Save profile', exact: true }).click();
	for (const input of [name, pronouns, bio])
		await expect(input).toHaveAttribute('aria-invalid', 'true');
	await expect(page.locator('.field-error')).toHaveCount(3);
	await expect(name).toBeFocused();
});
