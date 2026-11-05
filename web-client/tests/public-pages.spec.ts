import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
test('serves post text, project details and share metadata in initial HTML', async ({
	page,
	request
}) => {
	const response = await request.get(
		'/posts/did%3Aplc%3Aalice/first?focus=private&code=PRIVATE_CANARY'
	);
	expect(response.status()).toBe(200);
	const html = await response.text();
	expect(html).toContain('My linen dress');
	expect(html).not.toContain('PRIVATE_CANARY');
	expect(html).not.toContain('<script>alert');
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		'My linen dress'
	);
	await expect(page.locator('meta[property="og:title"]')).toHaveAttribute(
		'content',
		'My linen dress'
	);
	await expect(page.locator('meta[property="og:image"]')).toHaveAttribute(
		'content',
		'https://cdn.example/dress.jpg'
	);
	await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
		'href',
		'https://app.craftsky.social/posts/did%3Aplc%3Aalice/first'
	);
	await expect(page.getByText('Blue linen', { exact: true })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Summer dress' })).toHaveCount(0);
	await expect(
		page.getByText('Sponsored / commercial relationship', { exact: true })
	).toBeVisible();
	expect(response.headers()['cache-control']).toBe('no-store');
	expect(response.headers()['content-security-policy']).toContain(
		"frame-ancestors 'none'"
	);
});
test('profile, tab, pagination and author navigation work without JavaScript', async ({
	page
}) => {
	await page.goto('/profiles/alice.test');
	await expect(page.getByRole('heading', { level: 1 })).toContainText('Alice');
	await expect(
		page.getByText('Business account', { exact: true })
	).toBeVisible();
	await expect(page.getByRole('link', { name: 'Bad link' })).toHaveCount(0);
	await page.getByRole('link', { name: 'Projects', exact: true }).click();
	await expect(
		page.getByRole('link', { name: 'Projects', exact: true })
	).toHaveAttribute('aria-current', 'page');
	await page.getByRole('link', { name: 'Older projects' }).click();
	await expect(
		page.getByText('An older project', { exact: true })
	).toBeVisible();
	await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
		'content',
		'noindex,follow'
	);
	await page
		.getByRole('heading', { name: 'My linen dress' })
		.getByRole('link')
		.click();
	await expect(page).toHaveURL(/\/posts\//);
	await page.getByRole('link', { name: /More from/ }).click();
	await expect(page).toHaveURL(/\/profiles\//);
});
test('missing, upstream failure, invalid tab and unknown routes use real error statuses', async ({
	request
}) => {
	for (const [path, status] of [
		['/profiles/missing.test', 404],
		['/profiles/failed.test', 503],
		['/profiles/alice.test?tab=private', 400],
		['/unknown', 404]
	] as const) {
		const response = await request.get(path);
		expect(response.status()).toBe(status);
		expect(await response.text()).not.toContain('PRIVATE_CANARY');
		expect(response.headers()['x-robots-tag']).toContain('noindex');
		expect(response.headers()['cache-control']).toBe('no-store');
	}
});
test('OAuth fallbacks and association files retain their exact safe content', async ({
	request
}) => {
	for (const path of ['/auth/complete', '/account-deletion/reauth-complete']) {
		const response = await request.get(
			path + '?code=PRIVATE_CANARY&proof=PRIVATE_CANARY'
		);
		expect(response.status()).toBe(200);
		expect(await response.text()).toBe(
			readFileSync('../verified-links' + path + '.html', 'utf8')
		);
		expect(response.headers()['referrer-policy']).toBe('no-referrer');
		expect(response.headers()['cache-control']).toBe('no-store');
		expect(response.headers()['content-security-policy']).toContain(
			"default-src 'none'"
		);
	}
	for (const file of ['assetlinks.json', 'apple-app-site-association']) {
		const response = await request.get('/.well-known/' + file);
		expect(response.status()).toBe(200);
		expect(response.headers()['content-type']).toContain('application/json');
		expect(await response.text()).toBe(
			readFileSync('../verified-links/.well-known/' + file, 'utf8')
		);
	}
});
test('mobile layout fits the viewport and supports keyboard focus', async ({
	page
}) => {
	await page.setViewportSize({ width: 375, height: 812 });
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= window.innerWidth
		)
	).toBe(true);
	await page.keyboard.press('Tab');
	await expect(
		page.getByRole('link', { name: 'Skip to content' })
	).toBeFocused();
});

for (const colorScheme of ['light', 'dark'] as const) {
	test(`public pages pass accessibility checks in ${colorScheme} mode`, async ({
		browser
	}, testInfo) => {
		const { default: AxeBuilder } = await import('@axe-core/playwright');
		const context = await browser.newContext({
			javaScriptEnabled: true,
			colorScheme
		});
		const page = await context.newPage();
		try {
			for (const [name, path] of [
				['profile', '/profiles/alice.test'],
				['project', '/posts/did%3Aplc%3Aalice/first']
			]) {
				await page.goto('http://127.0.0.1:19840' + path);
				const results = await new AxeBuilder({ page })
					.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
					.analyze();
				expect(results.violations).toEqual([]);
				await page.screenshot({
					path: testInfo.outputPath(`${name}-${colorScheme}.png`),
					fullPage: true
				});
			}
		} finally {
			await context.close();
		}
	});
}

test('Flutter theme tokens, bundled fonts and narrow layouts are applied', async ({
	page
}) => {
	await page.emulateMedia({ colorScheme: 'light', reducedMotion: 'reduce' });
	await page.goto('/profiles/alice.test');
	await expect(page.locator('body')).toHaveCSS(
		'font-family',
		'Outfit, system-ui, sans-serif'
	);
	await expect(page.locator('body')).toHaveCSS('color', 'rgb(22, 18, 16)');
	await expect(page.locator('.profile-band')).toHaveCSS(
		'background-color',
		'rgb(21, 53, 214)'
	);
	await expect(page.locator('.post-card').first()).toHaveCSS(
		'border-radius',
		'14px'
	);
	for (const width of [320, 375, 800, 1280]) {
		await page.setViewportSize({ width, height: 900 });
		expect(
			await page.evaluate(
				() => document.documentElement.scrollWidth <= window.innerWidth
			)
		).toBe(true);
	}
	await page.emulateMedia({ colorScheme: 'dark' });
	await expect(page.locator('body')).toHaveCSS('color', 'rgb(245, 239, 228)');
});

test('a page remains readable when images cannot load', async ({ page }) => {
	await page.route('https://cdn.example/**', (route) => route.abort());
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText(
		'My linen dress'
	);
	await expect(page.getByAltText('A handmade blue linen dress')).toBeAttached();
});

test('root opens sign-in and private routes require a session', async ({
	page,
	request
}) => {
	await page.goto('/');
	await expect(page).toHaveURL('/sign-in');
	await expect(
		page.getByRole('heading', { name: 'Join CraftSky' })
	).toBeVisible();
	await expect(page.getByLabel('Your Atmosphere Handle')).toBeVisible();
	for (const path of [
		'/feed',
		'/notifications',
		'/messages/conversations/example',
		'/profile',
		'/settings'
	]) {
		const response = await request.get(path, { maxRedirects: 0 });
		expect(response.status()).toBe(303);
		expect(response.headers().location).toBe('/sign-in');
	}
	for (const path of ['/app/home', '/home', '/activity', '/unknown']) {
		expect((await request.get(path)).status()).toBe(404);
	}
});
