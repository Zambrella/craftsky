import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { resolve } from 'node:path';

for (const [width, height] of [
	[600, 2400],
	[2400, 600]
]) {
	test(`fullscreen fits a ${width}x${height} photo`, async ({ page }) => {
		await page.route('https://fixture.craftsky.test/**', (route) =>
			route.fulfill({
				contentType: 'image/svg+xml',
				body: `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}"><rect width="100%" height="100%" fill="red"/></svg>`
			})
		);
		await page.goto('/feed');
		await page
			.locator('.member-post')
			.first()
			.getByRole('link', { name: 'CraftSky illustration 1', exact: true })
			.click();
		await expect(page).toHaveURL(/\/posts\/.*\?imageCid=bafyimage1$/);
		await page
			.getByRole('link', { name: 'CraftSky illustration 1', exact: true })
			.click();
		const gallery = page.getByRole('dialog', { name: 'Image gallery' });
		await expect(gallery).toBeVisible();
		await expect(gallery.locator('img')).toHaveJSProperty('complete', true);
		const bounds = (await gallery.locator('img').boundingBox())!;
		const viewport = page.viewportSize()!;
		expect(bounds.x).toBeGreaterThanOrEqual(0);
		expect(bounds.y).toBeGreaterThanOrEqual(0);
		expect(bounds.x + bounds.width).toBeLessThanOrEqual(viewport.width);
		expect(bounds.y + bounds.height).toBeLessThanOrEqual(viewport.height);
	});
}

const fixture = 'http://127.0.0.1:19847';
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
test.use({ viewport: { width: 402, height: 874 }, reducedMotion: 'reduce' });
test.beforeEach(async ({ context, page }) => {
	await fetch(fixture + '/__test/reset');
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
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await page.route('https://fixture.craftsky.test/**', (route) =>
		route.fulfill({
			path: resolve('static/app-icon.png'),
			contentType: 'image/png'
		})
	);
	let picturedPost: unknown;
	await page.route(/\/api\/appview\/v1\/posts\/[^/]+\/[^/?]+$/, (route) =>
		route.fulfill({ json: picturedPost })
	);
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		const response = await route.fetch(),
			body = await response.json();
		body.items[0].post.images = [1, 2, 3].map((index) => ({
			cid: 'bafyimage' + index,
			mime: 'image/png',
			size: 1000,
			alt: `CraftSky illustration ${index}`,
			fullsize: `https://fixture.craftsky.test/image-${index}.png`,
			thumb: `https://fixture.craftsky.test/image-${index}.png`,
			aspectRatio: { width: 512, height: 512 }
		}));
		picturedPost = body.items[0].post;
		await route.fulfill({ response, json: body });
	});
});

test('native photo counter, dots and description follow selection, scrolling and keyboard focus', async ({
	page
}) => {
	await page.goto('/feed');
	const card = page.locator('.member-post').first();
	await expect(card.locator('.image-counter')).toHaveText('1/3');
	await card
		.getByRole('button', { name: 'Show image 3 of 3', exact: true })
		.click();
	await expect(card.locator('.image-counter')).toHaveText('3/3');
	await expect(
		card.getByRole('button', { name: 'Show image 3 of 3', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	await card
		.getByRole('button', { name: 'Image description', exact: true })
		.click();
	const description = page.getByRole('dialog', {
		name: 'Image description',
		exact: true
	});
	await expect(description).toContainText('CraftSky illustration 3');
	await page.keyboard.press('Escape');
	await expect(description).toBeHidden();
	await expect(
		card.getByRole('button', { name: 'Image description', exact: true })
	).toBeFocused();
	await card.locator('.gallery').evaluate((node) => {
		node.scrollLeft = 0;
	});
	await expect(card.locator('.image-counter')).toHaveText('1/3');
	await card
		.getByRole('link', { name: 'CraftSky illustration 1', exact: true })
		.focus();
	await page.keyboard.press('ArrowRight');
	await expect(card.locator('.image-counter')).toHaveText('2/3');
	await expect(
		card.getByRole('link', { name: 'CraftSky illustration 2', exact: true })
	).toBeFocused();
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
	const audit = await new AxeBuilder({ page })
		.include('.app-main')
		.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
		.analyze();
	expect(audit.violations).toEqual([]);
});

test('opening the selected photo retains the existing fullscreen gallery and keyboard navigation', async ({
	page
}) => {
	await page.goto('/feed');
	const card = page.locator('.member-post').first();
	await card
		.getByRole('button', { name: 'Show image 2 of 3', exact: true })
		.click();
	await expect(card.locator('.image-counter')).toHaveText('2/3');
	await card
		.getByRole('link', { name: 'CraftSky illustration 2', exact: true })
		.click();
	await expect(page).toHaveURL(/\/posts\/.*\?imageCid=bafyimage2$/);
	await expect(card.locator('.image-counter')).toHaveText('2/3');
	await card
		.getByRole('link', { name: 'CraftSky illustration 2', exact: true })
		.click();
	const gallery = page.locator('.image-dialog[open]');
	await expect(gallery).toBeVisible();
	await expect(gallery.getByRole('img')).toHaveAttribute(
		'src',
		'https://fixture.craftsky.test/image-2.png'
	);
	await page.keyboard.press('ArrowLeft');
	await expect(gallery.getByRole('img')).toHaveAttribute(
		'src',
		'https://fixture.craftsky.test/image-1.png'
	);
	await page.keyboard.press('Escape');
	await expect(gallery).toBeHidden();
});

test('compact navigation displays and announces capped unread counts', async ({
	page
}) => {
	await page.route('**/api/appview/v1/notifications/new-count', (route) =>
		route.fulfill({ json: { newCount: 123 } })
	);
	await page.route('**/api/appview/v1/chat/unread-counts', (route) =>
		route.fulfill({ json: { accepted: 6, requests: 2 } })
	);
	await page.goto('/feed');
	const navigation = page.getByRole('navigation', {
		name: 'Main tabs',
		exact: true
	});
	const activity = navigation.getByRole('link', {
			name: 'Activity',
			exact: true
		}),
		messages = navigation.getByRole('link', { name: 'Messages', exact: true });
	await expect(activity.locator('.bottom-count')).toHaveText('99+');
	await expect(activity).toHaveAccessibleDescription('123 new activities');
	await expect(messages.locator('.bottom-count')).toHaveText('6');
	await expect(messages).toHaveAccessibleDescription('6 unread conversations');
});

for (const target of ['body', 'date']) {
	test(`opening a carousel post through its ${target} retains the selected image`, async ({
		page
	}) => {
		await page.goto('/feed');
		const card = page.locator('.member-post').first();
		await card
			.getByRole('button', { name: 'Show image 3 of 3', exact: true })
			.click();
		await expect(card.locator('.image-counter')).toHaveText('3/3');
		if (target === 'body') await card.locator('.prose').first().click();
		else
			await card.getByRole('link', { name: 'Open post', exact: true }).click();
		await expect(page).toHaveURL(/\?imageCid=bafyimage3$/);
		await expect(
			page.locator('.member-post').first().locator('.image-counter')
		).toHaveText('3/3');
	});
}
