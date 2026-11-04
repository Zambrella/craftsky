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

async function newPost(page: import('@playwright/test').Page) {
	await page
		.getByRole('button', { name: 'Create post', exact: true })
		.filter({ visible: true })
		.click();
	await page
		.getByRole('link', { name: /^Regular post/ })
		.filter({ visible: true })
		.click();
	await expect(page).toHaveURL(/app\/compose$/);
}
for (const reduced of [false, true]) {
	test(`publishing uses the real Rive asset and returns to an updated feed: reduced=${reduced}`, async ({
		page
	}) => {
		await page.emulateMedia({
			reducedMotion: reduced ? 'reduce' : 'no-preference'
		});
		await page.addInitScript(() =>
			localStorage.setItem('craftsky.theme', 'dark')
		);
		let initial: unknown;
		await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
			if (!initial) initial = await (await route.fetch()).json();
			await route.fulfill({ json: initial });
		});
		let release!: () => void;
		const held = new Promise<void>((resolve) => (release = resolve));
		await page.route('**/api/appview/v1/posts', async (route) => {
			if (route.request().method() === 'POST') await held;
			await route.continue();
		});
		const errors: string[] = [];
		page.on('pageerror', (error) => errors.push(error.message));
		await page.goto('/app/home');
		await expect(page.locator('.post-card').first()).toBeVisible();
		const before = await page.locator('.post-card').count();
		await newPost(page);
		await page
			.getByRole('textbox', {
				name: 'What would you like to share?',
				exact: true
			})
			.fill('Confirmed, visible before indexing');
		await page.getByRole('button', { name: 'Post', exact: true }).click();
		const overlay = page.getByRole('dialog', {
			name: 'Publishing post…',
			exact: true
		});
		await expect(overlay).toBeVisible();
		await expect(overlay.locator('.publishing-animation')).toHaveAttribute(
			'data-ready',
			'true',
			{ timeout: 15000 }
		);
		const canvas = overlay.locator('canvas');
		const pixels = () =>
			canvas.evaluate((node) => {
				const data = Array.from(
					(node as HTMLCanvasElement)
						.getContext('2d')!
						.getImageData(
							0,
							0,
							(node as HTMLCanvasElement).width,
							(node as HTMLCanvasElement).height
						).data
				);
				return {
					drawn: data.some((value, i) => i % 4 === 3 && value > 0),
					hash: data.join(','),
					themed: data.some(
						(value, i) =>
							i % 4 === 0 &&
							value === 120 &&
							data[i + 1] === 144 &&
							data[i + 2] === 255
					)
				};
			});
		const first = await pixels();
		expect(first.drawn).toBe(true);
		expect(first.themed).toBe(true);
		await page.waitForTimeout(250);
		const second = await pixels();
		if (reduced) expect(second.hash).toBe(first.hash);
		else expect(second.hash).not.toBe(first.hash);
		await page.keyboard.press('Escape');
		await expect(overlay).toBeVisible();
		await expect(page).toHaveURL(/app\/compose$/);
		release();
		await expect(page).toHaveURL('/app/home');
		await expect(
			page
				.locator('.post-card')
				.filter({ hasText: 'Confirmed, visible before indexing' })
		).toHaveCount(1);
		await expect(page.locator('.post-card')).toHaveCount(before + 1);
		await page
			.getByRole('navigation', { name: 'App navigation', exact: true })
			.getByRole('link', { name: 'Home', exact: true })
			.click();
		await expect(
			page
				.locator('.post-card')
				.filter({ hasText: 'Confirmed, visible before indexing' })
		).toHaveCount(1);
		expect(errors).toEqual([]);
	});
}

test('publishing from a profile returns to that exact tab with the accepted post', async ({
	page
}) => {
	await page.goto('/app/profile/did%3Aplc%3Aalice?tab=posts');
	await expect(page.locator('.profile-content')).toBeVisible();
	await newPost(page);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Posted from my profile');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL('/app/profile/did%3Aplc%3Aalice?tab=posts');
	await expect(page.locator('.profile-content')).toContainText(
		'Posted from my profile'
	);
});

test('a confirmed comment returns to the original thread and appears before indexed comments', async ({
	page
}) => {
	await page.route('**/api/appview/v1/posts/*/*/comments*', (route) =>
		route.fulfill({ json: { comments: { items: [], cursor: null } } })
	);
	await page.goto('/app/posts/did%3Aplc%3Aalice/first?show-comments=true');
	await page.getByRole('link', { name: 'Add a comment', exact: true }).click();
	await page
		.getByRole('textbox', { name: 'Write your comment', exact: true })
		.fill('A confirmed new comment');
	await page.getByRole('button', { name: 'Comment', exact: true }).click();
	await expect(page).toHaveURL(
		'/app/posts/did%3Aplc%3Aalice/first?show-comments=true'
	);
	await expect(page.locator('.thread-comments')).toContainText(
		'A confirmed new comment'
	);
});

for (const theme of ['light', 'dark']) {
	test(`delete confirmation uses destructive styling in ${theme}`, async ({
		page
	}) => {
		await page.addInitScript(
			(theme) => localStorage.setItem('craftsky.theme', theme),
			theme
		);
		await page.goto('/app/posts/did%3Aplc%3Aalice/first');
		await page.getByLabel('More post actions', { exact: true }).first().click();
		await page.getByRole('button', { name: 'Delete', exact: true }).click();
		const dialog = page.getByRole('dialog', {
			name: 'Delete post?',
			exact: true
		});
		const button = dialog.getByRole('button', { name: 'Delete', exact: true });
		await page.mouse.move(0, 0);
		await expect(button).toHaveCSS('background-color', 'rgb(184, 32, 22)');
		await expect(button).toHaveCSS('color', 'rgb(255, 255, 255)');
		await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(dialog).toHaveCount(0);
	});
}

test('animation loading failure retains the publishing overlay and does not prevent acceptance', async ({
	page
}) => {
	await page.route('**/publishing.*.riv', (route) => route.abort());
	let release!: () => void;
	const held = new Promise<void>((resolve) => (release = resolve));
	await page.route('**/api/appview/v1/posts', async (route) => {
		if (route.request().method() === 'POST') await held;
		await route.continue();
	});
	await page.goto('/app/compose');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Published with fallback animation');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	const overlay = page.getByRole('dialog', {
		name: 'Publishing post…',
		exact: true
	});
	await expect(overlay.locator('.stitches')).toBeVisible();
	release();
	await expect(page).toHaveURL('/app/home');
	await expect(
		page
			.locator('.post-card')
			.filter({ hasText: 'Published with fallback animation' })
	).toHaveCount(1);
});

test('an unconfirmed write closes the overlay, retains the exact draft and adds no feed row', async ({
	page
}) => {
	await page.route('**/api/appview/v1/posts', (route) =>
		route.fulfill({
			status: 202,
			json: { status: 'ambiguous' },
			headers: { 'retry-after': '1' }
		})
	);
	await page.goto('/app/home');
	await expect(page.locator('.post-card').first()).toBeVisible();
	await newPost(page);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Unconfirmed publication');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Waiting for confirmation', exact: true })
	).toBeVisible();
	await expect(page.locator('.publishing-overlay')).not.toBeVisible();
	await page
		.getByRole('navigation', { name: 'App navigation', exact: true })
		.getByRole('link', { name: 'Home', exact: true })
		.click();
	await expect(page).toHaveURL('/app/home');
	await expect(
		page.locator('.post-card').filter({ hasText: 'Unconfirmed publication' })
	).toHaveCount(0);
});
