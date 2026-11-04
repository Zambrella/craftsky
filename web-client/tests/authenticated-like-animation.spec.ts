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

const ready = '.like-reaction-host[data-asset-ready="true"]';
for (const reduced of [false, true]) {
	test(`heart uses the real native Rive asset with reduced motion=${reduced}`, async ({
		page
	}) => {
		await page.emulateMedia({
			reducedMotion: reduced ? 'reduce' : 'no-preference'
		});
		let release!: () => void;
		const held = new Promise<void>((resolve) => (release = resolve));
		await page.route('**/api/appview/v1/posts/*/*/likes', async (route) => {
			if (route.request().method() === 'POST') await held;
			await route.continue();
		});
		const requests: string[] = [];
		page.on('request', (request) => requests.push(request.url()));
		await page.goto('/app/home');
		await expect(page.locator(ready).first()).toBeAttached();
		// Keep the 400 ms reduced-motion reaction alive while inspecting its pixels.
		await page.clock.install();
		await page.clock.pauseAt(new Date());
		const card = page.locator('.member-post').first();
		const like = card.getByRole('button', { name: 'Like', exact: true });
		await like.click();
		await page.clock.runFor(80);
		const reaction = card.locator('.like-reaction');
		await expect(reaction).toBeVisible();
		await expect(page.locator('.like-reaction')).toHaveCount(1);
		await expect(like).toBeDisabled();
		await expect(reaction).toHaveCSS('pointer-events', 'none');
		await expect(reaction).toHaveAttribute('aria-hidden', 'true');
		await expect(reaction).toHaveAttribute('data-ready', 'true');
		const canvas = reaction.locator('canvas');
		const pixels = () =>
			canvas.evaluate((node) => {
				const canvas = node as HTMLCanvasElement;
				return Array.from(
					canvas
						.getContext('2d')!
						.getImageData(0, 0, canvas.width, canvas.height).data
				);
			});
		if (!reduced) await page.clock.runFor(180);
		const first = await pixels();
		expect(first.some((value, i) => i % 4 === 3 && value > 0)).toBe(true);
		await page.clock.runFor(reduced ? 60 : 120);
		if (reduced) expect(await pixels()).toEqual(first);
		else expect(await pixels()).not.toEqual(first);
		release();
		await page.clock.runFor(1500);
		await expect(like).toHaveAttribute('aria-pressed', 'true');
		await expect(reaction).toHaveCount(0, { timeout: 2000 });
		expect(requests.filter((url) => /like\.[^/]+\.riv/.test(url))).toHaveLength(
			1
		);
		expect(
			requests.filter((url) => /rive\.[^/]+\.wasm/.test(url))
		).toHaveLength(1);
	});
}

test('unlike cancels feedback and liking again starts a new playback', async ({
	page
}) => {
	await page.goto('/app/home');
	await expect(page.locator(ready).first()).toBeAttached();
	const card = page.locator('.member-post').first(),
		like = card.getByRole('button', { name: 'Like', exact: true });
	await like.click();
	await expect(like).toHaveAttribute('aria-pressed', 'true');
	await expect(card.locator('.like-reaction')).toHaveAttribute(
		'data-sequence',
		'1'
	);
	await like.click();
	await expect(card.locator('.like-reaction')).toHaveCount(0);
	await expect(like).toHaveAttribute('aria-pressed', 'false');
	await like.click();
	await expect(card.locator('.like-reaction')).toHaveAttribute(
		'data-sequence',
		'2'
	);
});

test('a rejected like cancels feedback without changing the saved like state', async ({
	page
}) => {
	let release!: () => void;
	const held = new Promise<void>((resolve) => (release = resolve));
	await page.route('**/api/appview/v1/posts/*/*/likes', async (route) => {
		await held;
		await route.fulfill({
			status: 403,
			json: { error: 'not_allowed', message: 'This like was not saved.' }
		});
	});
	await page.goto('/app/home');
	await expect(page.locator(ready).first()).toBeAttached();
	const card = page.locator('.member-post').first(),
		like = card.getByRole('button', { name: 'Like', exact: true });
	await like.click();
	await expect(card.locator('.like-reaction')).toBeVisible();
	release();
	await expect(page.locator('.snackbar')).toContainText(
		'This like was not saved.'
	);
	await expect(card.locator('.like-reaction')).toHaveCount(0);
	await expect(like).toHaveAttribute('aria-pressed', 'false');
});

test('late asset readiness keeps the immediate static fallback for that reaction', async ({
	page
}) => {
	let release!: () => void;
	const held = new Promise<void>((resolve) => (release = resolve));
	await page.route('**/like.*.riv', async (route) => {
		await held;
		await route.continue();
	});
	await page.goto('/app/home');
	const card = page.locator('.member-post').first();
	await card.getByRole('button', { name: 'Like', exact: true }).click();
	const reaction = card.locator('.like-reaction');
	await expect(reaction.locator('.fallback')).toBeVisible();
	release();
	await expect(card.locator(ready)).toBeAttached();
	await expect(reaction.locator('canvas')).toHaveCount(0);
	await expect(
		card.getByRole('button', { name: 'Like', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
});

test('heart tracks the visible part of a tall post while scrolling without restarting playback', async ({
	page
}) => {
	await page.setViewportSize({ width: 470, height: 844 });
	await page.goto('/app/home');
	await expect(page.locator(ready).first()).toBeAttached();
	const card = page.locator('.member-post').first();
	await card
		.locator('.post-card')
		.evaluate((node) => ((node as HTMLElement).style.minHeight = '1800px'));
	await card.getByRole('button', { name: 'Like', exact: true }).click();
	const reaction = card.locator('.like-reaction');
	await expect(reaction).toHaveAttribute('data-ready', 'true');
	const canvas = reaction.locator('canvas');
	await canvas.evaluate(
		(node) => ((node as HTMLCanvasElement).dataset.identity = 'original')
	);
	const centre = async () => {
		const bounds = await card.boundingBox(),
			heart = await reaction.boundingBox();
		expect(bounds).not.toBeNull();
		expect(heart).not.toBeNull();
		const top = Math.max(0, bounds!.y),
			bottom = Math.min(844, bounds!.y + bounds!.height);
		expect(
			Math.abs(heart!.y + heart!.height / 2 - (top + bottom) / 2)
		).toBeLessThan(2);
		expect(heart!.y).toBeGreaterThanOrEqual(top - 1);
		expect(heart!.y + heart!.height).toBeLessThanOrEqual(bottom + 1);
	};
	await centre();
	await page.evaluate(() => scrollBy(0, -120));
	await expect
		.poll(async () => {
			const bounds = await card.boundingBox(),
				heart = await reaction.boundingBox();
			return Math.abs(
				heart!.y +
					heart!.height / 2 -
					(Math.max(0, bounds!.y) + Math.min(844, bounds!.y + bounds!.height)) /
						2
			);
		})
		.toBeLessThan(2);
	await expect(canvas).toHaveAttribute('data-identity', 'original');
	await expect(reaction).toHaveAttribute('data-sequence', '1');
});

test('a malformed animation retains static feedback and never prevents a like', async ({
	page
}) => {
	await page.route('**/like.*.riv', (route) =>
		route.fulfill({
			contentType: 'application/octet-stream',
			body: Buffer.from('invalid animation fixture')
		})
	);
	await page.goto('/app/home');
	await expect(page.locator(ready).first()).toBeAttached();
	const card = page.locator('.member-post').first(),
		like = card.getByRole('button', { name: 'Like', exact: true });
	await like.click();
	await expect(card.locator('.like-reaction .fallback')).toBeVisible();
	await expect(like).toHaveAttribute('aria-pressed', 'true');
});
