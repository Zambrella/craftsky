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

test('mixed-ratio images keep the feed frame stable through lazy decoding and selection', async ({
	page
}) => {
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		body.items[0].post.images = [
			{ width: 400, height: 500 },
			{ width: 1000, height: 100 }
		].map((aspectRatio, index) => ({
			cid: `mixed-${index}`,
			mime: 'image/png',
			size: 1,
			alt: `Mixed photo ${index + 1}`,
			fullsize: `https://fixture.craftsky.test/mixed-${index}.png`,
			thumb: `https://fixture.craftsky.test/mixed-${index}.png`,
			aspectRatio
		}));
		await route.fulfill({ response, json: body });
	});
	await page.goto('/feed');
	const card = page.locator('.member-post').first();
	const frame = card.getByRole('region', { name: 'Post images' });
	await expect(frame).toBeVisible();
	const before = (await frame.boundingBox())!;
	await card
		.getByRole('button', { name: 'Show image 2 of 2', exact: true })
		.click();
	await expect(card.locator('.image-counter')).toHaveText('2/2');
	await expect(frame.locator('img').nth(1)).toHaveJSProperty('complete', true);
	const after = (await frame.boundingBox())!;
	expect(after.height).toBeCloseTo(before.height, 1);
	expect(after.y).toBeCloseTo(before.y, 1);
	expect(after.height).toBeLessThanOrEqual(
		page.viewportSize()!.height * 0.65 + 1
	);
});

test('height-limited carousels keep mixed-ratio photos centered across paging and resizing', async ({
	page
}) => {
	const dimensions = [
		{ width: 600, height: 900 },
		{ width: 1200, height: 600 }
	];
	await page.route('https://fixture.craftsky.test/centered-*.svg', (route) => {
		const index = Number(
			new URL(route.request().url()).pathname.match(/\d+/)![0]
		);
		const { width, height } = dimensions[index];
		return route.fulfill({
			contentType: 'image/svg+xml',
			body: `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}"><rect width="100%" height="100%" fill="red"/></svg>`
		});
	});
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		body.items[0].post.images = dimensions.map((aspectRatio, index) => ({
			cid: `centered-${index}`,
			mime: 'image/svg+xml',
			size: 1,
			alt: `Centered photo ${index + 1}`,
			fullsize: `https://fixture.craftsky.test/centered-${index}.svg`,
			thumb: `https://fixture.craftsky.test/centered-${index}.svg`,
			aspectRatio
		}));
		await route.fulfill({ response, json: body });
	});
	await page.setViewportSize({ width: 1440, height: 1100 });
	await page.goto('/feed');
	const card = page.locator('.member-post').first();
	const frame = card.getByRole('region', { name: 'Post images' });
	const container = card.locator('.post-images');
	const expectCentered = async (index: number) => {
		const image = frame.locator('img').nth(index);
		await expect(image).toHaveJSProperty(
			'naturalWidth',
			dimensions[index].width
		);
		await expect
			.poll(async () => {
				const parent = (await container.boundingBox())!;
				const gallery = (await frame.boundingBox())!;
				const photo = (await image.boundingBox())!;
				return Math.max(
					Math.abs(gallery.width - parent.width),
					Math.abs(gallery.x + gallery.width / 2 - parent.x - parent.width / 2),
					Math.abs(photo.x + photo.width / 2 - parent.x - parent.width / 2)
				);
			})
			.toBeLessThanOrEqual(1);
	};
	await expect(frame).toBeVisible();
	expect((await container.boundingBox())!.width).toBeGreaterThan(700);
	await expectCentered(0);
	await card
		.getByRole('button', { name: 'Show image 2 of 2', exact: true })
		.click();
	await expect(card.locator('.image-counter')).toHaveText('2/2');
	await expectCentered(1);
	await card
		.getByRole('button', { name: 'Show image 1 of 2', exact: true })
		.click();
	await expectCentered(0);
	await page.setViewportSize({ width: 402, height: 874 });
	await expectCentered(0);
	await card
		.getByRole('button', { name: 'Show image 2 of 2', exact: true })
		.click();
	await expect(card.locator('.image-counter')).toHaveText('2/2');
	await expectCentered(1);
});

for (const source of ['timeline', 'extended-network']) {
	test(`${source} refresh preserves paginated rows and scroll position`, async ({
		page
	}) => {
		let refreshing = false;
		let finish!: () => void;
		const delayed = new Promise<void>((resolve) => {
			finish = resolve;
		});
		await page.route(`**/api/appview/v1/feed/${source}*`, async (route) => {
			const response = await route.fetch({
				url: route
					.request()
					.url()
					.replace('/feed/extended-network', '/feed/timeline')
			});
			const body = await response.json();
			const start = Number(
				new URL(route.request().url()).searchParams.get('cursor') || 0
			);
			if (refreshing && start === 6) await delayed;
			const template = body.items[0];
			body.items = Array.from({ length: 3 }, (_, i) => {
				const n = start + i;
				const post = {
					...template.post,
					uri: `at://${template.post.author.did}/social.craftsky.feed.post/retained${n}`,
					rkey: `retained${n}`,
					text: `${refreshing ? 'Reloaded' : 'Retained'} feed post ${n}`
				};
				return { ...template, itemKey: `post:${post.uri}`, post };
			});
			body.cursor = start < 6 ? String(start + 3) : null;
			await route.fulfill({ response, json: body });
		});
		await page.goto(
			source === 'timeline' ? '/feed' : '/feed?source=extended-network'
		);
		await expect(page.locator('.member-post').nth(2)).toBeAttached();
		await page.evaluate(() =>
			scrollTo({
				top: document.documentElement.scrollHeight,
				behavior: 'instant'
			})
		);
		await expect(page.locator('.member-post').nth(5)).toBeAttached();
		await page.evaluate(() =>
			scrollTo({
				top: document.documentElement.scrollHeight,
				behavior: 'instant'
			})
		);
		await expect(page.locator('.member-post')).toHaveCount(9);
		await page.locator('.member-post').nth(7).scrollIntoViewIfNeeded();
		const before = await page.evaluate(() => window.scrollY);
		expect(before).toBeGreaterThan(500);
		refreshing = true;
		const lastPage = page.waitForRequest(
			(request) =>
				request.url().includes(`/feed/${source}`) &&
				new URL(request.url()).searchParams.get('cursor') === '6'
		);
		await page.evaluate(() =>
			window.dispatchEvent(new Event('craftsky-refresh'))
		);
		await lastPage;
		await expect(page.locator('.member-post')).toHaveCount(9);
		expect(await page.evaluate(() => window.scrollY)).toBeCloseTo(before, 0);
		finish();
		await expect(page.locator('.member-post').nth(7)).toContainText(
			'Reloaded feed post 7'
		);
		await expect(page.locator('.member-post')).toHaveCount(9);
		expect(await page.evaluate(() => window.scrollY)).toBeCloseTo(before, 0);
	});
}

for (const removeVisible of [false, true]) {
	test(`feed anchors a surviving post after insertion and removal (${removeVisible})`, async ({
		page
	}) => {
		let refreshed = false;
		await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
			const response = await route.fetch();
			const body = await response.json();
			const template = body.items[0];
			const old = Array.from({ length: 15 }, (_, i) => String(i));
			const ids = refreshed
				? [
						'new-a',
						'new-b',
						...old.filter((id) => id !== (removeVisible ? '7' : '0'))
					]
				: old;
			body.items = ids.map((id) => ({
				...template,
				itemKey: `anchor-${id}`,
				post: {
					...template.post,
					uri: `at://${template.post.author.did}/social.craftsky.feed.post/anchor${id}`,
					rkey: `anchor${id}`,
					text: `Anchor post ${id}`
				}
			}));
			body.cursor = null;
			await route.fulfill({ response, json: body });
		});
		await page.goto('/feed');
		const target = page.locator('[data-scroll-anchor="anchor-7"]');
		await target.scrollIntoViewIfNeeded();
		await page.evaluate(() => {
			const node = document.querySelector('[data-scroll-anchor="anchor-7"]')!;
			window.scrollBy(0, node.getBoundingClientRect().top + 50);
		});
		const survivor = page.locator('[data-scroll-anchor="anchor-8"]');
		const before = (await survivor.boundingBox())!.y;
		refreshed = true;
		await page.evaluate(() =>
			window.dispatchEvent(new Event('craftsky-refresh'))
		);
		await expect(
			page.locator('[data-scroll-anchor="anchor-new-a"]')
		).toHaveCount(1);
		await expect
			.poll(async () => (await survivor.boundingBox())!.y)
			.toBeCloseTo(before, 0);
	});
}

test('saved posts keep the reading position when an earlier removal finishes', async ({
	page
}) => {
	await page.route('**/api/appview/v1/saved-posts*', async (route) => {
		const response = await route.fetch({
			url: new URL('/api/appview/v1/feed/timeline', route.request().url()).href
		});
		const body = await response.json();
		const template = body.items[0].post;
		await route.fulfill({
			json: {
				items: Array.from({ length: 12 }, (_, i) => ({
					savedAt: '2026-11-01T12:00:00Z',
					post: {
						...template,
						viewerHasSaved: true,
						uri: `at://${template.author.did}/social.craftsky.feed.post/save${i}`,
						rkey: `save${i}`,
						text: `Saved row ${i}`
					}
				}))
			}
		});
	});
	let finish!: () => void;
	const pending = new Promise<void>((resolve) => {
		finish = resolve;
	});
	await page.route('**/api/appview/v1/posts/*/*/saves', async (route) => {
		await pending;
		await route.fulfill({ status: 204 });
	});
	await page.goto('/saved?all=true');
	await expect(page.locator('.member-post')).toHaveCount(12);
	const first = page.locator('.member-post').first();
	await first.getByLabel('Saved post actions', { exact: true }).click();
	await first
		.getByRole('button', { name: 'Remove from saved posts', exact: true })
		.click();
	await page.locator('.member-post').nth(6).scrollIntoViewIfNeeded();
	const survivor = page.locator('[data-scroll-anchor$="/save7"]');
	const before = (await survivor.boundingBox())!.y;
	finish();
	await expect(page.locator('.member-post')).toHaveCount(11);
	await expect
		.poll(async () => (await survivor.boundingBox())!.y)
		.toBeCloseTo(before, 0);
});
