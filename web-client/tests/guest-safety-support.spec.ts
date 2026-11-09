import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
test('public guide needs no sign-in, preloads no support sites and has accessible controls', async ({
	page
}) => {
	const outgoing: string[] = [];
	page.on('request', (r) => {
		if (!new URL(r.url()).hostname.match(/^(127\.0\.0\.1|localhost)$/))
			outgoing.push(r.url());
	});
	await page.goto('/safety');
	await expect(
		page.getByRole('heading', { name: 'For parents and carers' })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Bluesky messages' })
	).toBeVisible();
	const links = page.locator('.safety-guide a[target="_blank"]');
	expect(await links.count()).toBe(8);
	for (const a of await links.all()) {
		expect(await a.getAttribute('rel')).toContain('noreferrer');
		expect(await a.getAttribute('referrerpolicy')).toBe('no-referrer');
		expect(new URL((await a.getAttribute('href'))!).search).toBe('');
	}
	expect(outgoing).toEqual([]);
	expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([]);
	await page.setViewportSize({ width: 320, height: 640 });
	await page.locator('html').evaluate((e) => (e.style.fontSize = '200%'));
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
});
test('relevant submitted searches keep results and support can be dismissed or revisited', async ({
	page
}) => {
	await page.goto('/search?q=self-harm');
	await expect(
		page.getByRole('complementary', { name: 'Support' })
	).toBeVisible();
	await expect(
		page.getByRole('searchbox', { name: 'Search CraftSky' })
	).toHaveValue('self-harm');
	await page.getByRole('button', { name: 'Dismiss', exact: true }).click();
	await expect(
		page.getByRole('complementary', { name: 'Support' })
	).toHaveCount(0);
	await page.goto('/search?q=knitting');
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('complementary', { name: 'Support' })
	).toHaveCount(0);
	await page.goto('/search?q=suicide');
	await page.getByRole('link', { name: 'Find support', exact: true }).click();
	await expect(page).toHaveURL('/safety');
});
