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

test.use({ viewport: { width: 402, height: 874 }, reducedMotion: 'reduce' });
const host = 'https://carousel.craftsky.test/';
const images = Array.from({ length: 6 }, (_, i) => ({
	cid: 'preload-' + i,
	mime: 'image/png',
	size: 1000,
	alt: 'Photo ' + i,
	fullsize: host + 'full-' + i + '.png',
	thumb: host + 'thumb-' + i + '.png',
	aspectRatio: { width: 512, height: 512 }
}));
test.beforeEach(async ({ page }) => {
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		const response = await route.fetch(),
			body = await response.json();
		body.items[0].post.images = images;
		await route.fulfill({ response, json: body });
	});
	await page.route(
		/\/api\/appview\/v1\/posts\/[^/]+\/[^/?]+$/,
		async (route) => {
			const response = await route.fetch(),
				post = await response.json();
			post.images = images;
			await route.fulfill({ response, json: post });
		}
	);
	await page.route(host + '**', (route) =>
		route.fulfill({
			path: resolve('static/app-icon.png'),
			contentType: 'image/png',
			headers: { 'cache-control': 'public, max-age=600' }
		})
	);
});
function requests(page: import('@playwright/test').Page) {
	const seen = new Set<string>();
	page.on('request', (request) => {
		if (request.url().startsWith(host))
			seen.add(request.url().slice(host.length));
	});
	return () => [...seen].sort();
}

test('inline thumbnails wait for current decode and warm only the changing adjacent window', async ({
	page
}) => {
	const seen = requests(page);
	let release!: () => void;
	const held = new Promise<void>((resolve) => (release = resolve));
	await page.route(host + 'thumb-0.png', async (route) => {
		await held;
		await route.fulfill({
			path: resolve('static/app-icon.png'),
			contentType: 'image/png'
		});
	});
	await page.goto('/app/home');
	const card = page.locator('.member-post').first();
	await card.locator('.gallery').scrollIntoViewIfNeeded();
	await expect.poll(seen).toEqual(['thumb-0.png']);
	release();
	await expect.poll(seen).toEqual(['thumb-0.png', 'thumb-1.png']);
	await expect(card.locator('.gallery img').nth(1)).toHaveJSProperty(
		'complete',
		true
	);
	expect(seen()).toEqual(['thumb-0.png', 'thumb-1.png']);
	await card.getByRole('button', { name: 'Next image', exact: true }).click();
	await expect
		.poll(seen)
		.toEqual(['thumb-0.png', 'thumb-1.png', 'thumb-2.png']);
	await card
		.getByRole('button', { name: 'Show image 6 of 6', exact: true })
		.click();
	await expect(card.locator('.image-counter')).toHaveText('6/6');
	await expect
		.poll(seen)
		.toEqual([
			'thumb-0.png',
			'thumb-1.png',
			'thumb-2.png',
			'thumb-4.png',
			'thumb-5.png'
		]);
	await expect(card.locator('.image-arrow.next')).toBeDisabled();
});

test('middle-of-album detail and fullscreen views warm full-size neighbours without cascading', async ({
	page
}) => {
	const seen = requests(page);
	await page.goto('/app/posts/did%3Aplc%3Aalice/first?imageCid=preload-3');
	const card = page.locator('.member-post').first();
	await expect(card.locator('.image-counter')).toHaveText('4/6');
	await expect.poll(seen).toEqual(['full-2.png', 'full-3.png', 'full-4.png']);
	await card.locator('.gallery a').nth(3).click();
	const gallery = page.getByRole('dialog', {
		name: 'Image gallery',
		exact: true
	});
	await expect(gallery).toBeVisible();
	await gallery.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(gallery.getByRole('img')).toHaveAttribute(
		'src',
		host + 'full-4.png'
	);
	await expect
		.poll(seen)
		.toEqual(['full-2.png', 'full-3.png', 'full-4.png', 'full-5.png']);
	await gallery.getByRole('button', { name: 'Previous', exact: true }).click();
	await expect(gallery.getByRole('img')).toHaveAttribute(
		'src',
		host + 'full-3.png'
	);
	expect(seen()).toEqual([
		'full-2.png',
		'full-3.png',
		'full-4.png',
		'full-5.png'
	]);
});

for (const leave of [false, true]) {
	test(`stale decode does not warm obsolete neighbours when leaving=${leave}`, async ({
		page
	}) => {
		await page.addInitScript(() => {
			const current = window as typeof window & {
				releaseDecode: () => void;
				decodeWaiting: boolean;
			};
			let release!: () => void;
			const held = new Promise<void>((resolve) => (release = resolve));
			current.releaseDecode = release;
			const decode = HTMLImageElement.prototype.decode;
			HTMLImageElement.prototype.decode = async function () {
				await decode.call(this);
				if (this.src.endsWith('/thumb-0.png') && this.closest('.gallery')) {
					current.decodeWaiting = true;
					await held;
				}
			};
		});
		const seen = requests(page);
		await page.goto('/app/home');
		const card = page.locator('.member-post').first();
		await card.locator('.gallery').scrollIntoViewIfNeeded();
		await expect
			.poll(() =>
				page.evaluate(
					() =>
						(window as typeof window & { decodeWaiting: boolean }).decodeWaiting
				)
			)
			.toBe(true);
		expect(seen()).toEqual(['thumb-0.png']);
		if (leave) {
			await page
				.getByRole('navigation', { name: 'Main tabs', exact: true })
				.getByRole('link', { name: 'Search', exact: true })
				.click();
			await expect(page).toHaveURL('/app/search');
		} else {
			await card
				.getByRole('button', { name: 'Show image 4 of 6', exact: true })
				.click();
			await expect
				.poll(seen)
				.toEqual(['thumb-0.png', 'thumb-2.png', 'thumb-3.png', 'thumb-4.png']);
		}
		await page.evaluate(() =>
			(window as typeof window & { releaseDecode: () => void }).releaseDecode()
		);
		await page.evaluate(
			() =>
				new Promise<void>((resolve) =>
					requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
				)
		);
		expect(seen()).not.toContain('thumb-1.png');
	});
}

test('a failed speculative neighbour is silent and remains a normal visible-image error if selected', async ({
	page
}) => {
	const seen = requests(page);
	await page.route(host + 'full-4.png', (route) =>
		route.fulfill({ status: 404, body: 'Not found' })
	);
	await page.goto('/app/posts/did%3Aplc%3Aalice/first?imageCid=preload-3');
	const card = page.locator('.member-post').first();
	await expect.poll(seen).toEqual(['full-2.png', 'full-3.png', 'full-4.png']);
	await expect(page.locator('.snackbar')).toHaveCount(0);
	await card.locator('.gallery a').nth(3).click();
	const gallery = page.getByRole('dialog', {
		name: 'Image gallery',
		exact: true
	});
	await gallery.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(gallery.getByRole('alert')).toContainText(
		'This image could not be loaded.'
	);
	expect(seen()).not.toContain('full-5.png');
});
