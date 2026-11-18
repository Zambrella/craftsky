import { test, expect } from '@playwright/test';

test('guest profile menus offer link sharing without a share card', async ({
	page
}) => {
	await page.goto('/profiles/alice.test');
	await page.getByLabel('More profile actions', { exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Share card', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Share', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Copy link', exact: true })
	).toBeVisible();
});
