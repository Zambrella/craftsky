import { test, expect } from '@playwright/test';
import { reportConfirmation } from '../src/lib/features/moderation/report-confirmation';

for (const subject of ['post', 'profile']) {
	test(`guest submits a ${subject} report through the shared form`, async ({
		page
	}) => {
		const writes: { path: string; body: unknown }[] = [];
		page.on('request', (request) => {
			const path = new URL(request.url()).pathname;
			if (path.startsWith('/api/') && request.method() !== 'GET')
				writes.push({ path, body: request.postDataJSON() });
		});
		await page.goto(subject === 'post' ? '/feed' : '/profiles/alice.test');
		if (subject === 'post') {
			await page.getByRole('button', { name: 'Explore CraftSky' }).click();
			const card = page.locator('.member-post').first();
			await card.getByLabel('More post actions', { exact: true }).click();
			await card
				.getByRole('button', { name: 'Report post', exact: true })
				.click();
		} else {
			await page.getByLabel('More profile actions', { exact: true }).click();
			await page
				.getByRole('button', { name: 'Report profile', exact: true })
				.click();
		}
		const dialog = page.getByRole('dialog', {
			name: `Report ${subject}`,
			exact: true
		});
		await expect(dialog).toBeVisible();
		await expect(
			page.getByRole('dialog', { name: 'Join CraftSky' })
		).toHaveCount(0);
		await dialog
			.getByLabel('Spam, misleading or off-topic content', { exact: true })
			.click();
		await dialog.getByLabel('Spam', { exact: true }).check();
		await dialog
			.getByLabel('Details', { exact: true })
			.fill('Private guest concern');
		if (subject === 'post') {
			await dialog.getByLabel('Contact email (optional)').fill('not-an-email');
			await dialog.getByRole('button', { name: 'Submit report' }).click();
			expect(writes).toEqual([]);
			await dialog
				.getByLabel('Contact email (optional)')
				.fill('visitor@example.test');
		}
		await dialog.getByRole('button', { name: 'Submit report' }).click();
		await expect(dialog).toHaveCount(0);
		await expect(
			page.locator('.snackbar').getByText(reportConfirmation, { exact: true })
		).toBeVisible();
		await expect(
			page
				.locator('.snackbar')
				.getByRole('link', { name: 'Find support', exact: true })
		).toHaveAttribute('href', '/safety');
		expect(writes).toHaveLength(1);
		expect(writes[0].path).toMatch(
			subject === 'post'
				? /^\/api\/public\/posts\/[^/]+\/first\/reports$/
				: /^\/api\/public\/profiles\/[^/]+\/reports$/
		);
		expect(writes[0].body).toEqual({
			reasonType: 'spam',
			details: 'Private guest concern',
			...(subject === 'post' ? { contactEmail: 'visitor@example.test' } : {})
		});
	});
}

test('guest report can be cancelled and a failed submission retried in place', async ({
	page
}) => {
	await page.goto('/profiles/alice.test');
	await page.getByLabel('More profile actions', { exact: true }).click();
	await page
		.getByRole('button', { name: 'Report profile', exact: true })
		.click();
	await page.getByRole('button', { name: 'Cancel report' }).click();
	await expect(
		page.getByRole('dialog', { name: 'Report profile', exact: true })
	).toHaveCount(0);
	await page.getByLabel('More profile actions', { exact: true }).click();
	await page
		.getByRole('button', { name: 'Report profile', exact: true })
		.click();
	const dialog = page.getByRole('dialog', {
		name: 'Report profile',
		exact: true
	});
	await dialog
		.getByLabel('Spam, misleading or off-topic content', { exact: true })
		.click();
	await dialog.getByLabel('Spam', { exact: true }).check();
	await dialog
		.getByLabel('Details', { exact: true })
		.fill('Retain this report');
	const submissions: unknown[] = [];
	await page.route('**/api/public/profiles/*/reports', async (route) => {
		submissions.push(route.request().postDataJSON());
		if (submissions.length === 1)
			await route.fulfill({
				status: 429,
				json: { error: 'rate_limited', message: 'Please try again shortly.' },
				headers: { 'retry-after': '1' }
			});
		else await route.continue();
	});
	await dialog.getByRole('button', { name: 'Submit report' }).click();
	await expect(dialog.getByLabel('Details', { exact: true })).toHaveValue(
		'Retain this report'
	);
	await page
		.getByRole('alert')
		.getByRole('button', { name: 'Retry', exact: true })
		.click();
	await expect(dialog).toHaveCount(0);
	expect(submissions).toEqual([
		{ reasonType: 'spam', details: 'Retain this report' },
		{ reasonType: 'spam', details: 'Retain this report' }
	]);
});

test('report option labels stay in the list-page dialog', async ({ page }) => {
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	const listUrl = page.url();
	const card = page.locator('.member-post').first();
	await card.getByLabel('More post actions', { exact: true }).click();
	await card.getByRole('button', { name: 'Report post', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Report post', exact: true });
	await dialog.getByText('Child safety', { exact: true }).click();
	await expect(
		dialog.getByRole('radio', { name: 'Grooming', exact: true })
	).toBeVisible();
	await expect(page).toHaveURL(listUrl);
	await dialog.getByText('Grooming', { exact: true }).click();
	await expect(
		dialog.getByRole('radio', { name: 'Grooming', exact: true })
	).toBeChecked();
	await dialog.getByLabel('Details', { exact: true }).click();
	await expect(page).toHaveURL(listUrl);
	await dialog.getByRole('button', { name: 'Change category' }).click();
	await dialog
		.getByText('Spam, misleading or off-topic content', { exact: true })
		.click();
	await expect(
		dialog.getByRole('radio', { name: 'Spam', exact: true })
	).toBeVisible();
	await expect(page).toHaveURL(listUrl);
	await dialog.getByRole('button', { name: 'Cancel report' }).click();
	await expect(dialog).toHaveCount(0);
	await expect(page).toHaveURL(listUrl);
	await card.getByText('Guest knitting project', { exact: true }).click();
	await expect(page).toHaveURL(/\/posts\//);
});
