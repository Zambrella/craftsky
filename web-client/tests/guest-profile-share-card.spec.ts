import { test, expect } from '@playwright/test';

test('guests can open and save a public profile card without signing in', async ({
	page
}) => {
	await page.addInitScript(() =>
		Object.defineProperty(navigator, 'canShare', {
			value: () => false,
			configurable: true
		})
	);
	await page.goto('/profiles/alice.test');
	await page.getByLabel('More profile actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Share card', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Your share card' });
	await expect(dialog.getByRole('img')).toBeVisible();
	const image = dialog.getByRole('img');
	expect(
		await image.evaluate((element) => [
			(element as HTMLImageElement).naturalWidth,
			(element as HTMLImageElement).naturalHeight
		])
	).toEqual([1200, 1500]);
	const pending = page.waitForEvent('download');
	await dialog.getByRole('button', { name: 'Share', exact: true }).click();
	expect((await pending).suggestedFilename()).toBe('CraftSky-profile.png');
	await expect(page.getByRole('dialog')).toHaveCount(1);
	await dialog.getByRole('button', { name: 'Close share card' }).click();
	await expect(dialog).not.toBeVisible();
	await expect(
		page.getByLabel('More profile actions', { exact: true })
	).toBeFocused();
});
