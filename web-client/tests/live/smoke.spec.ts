import {
	test,
	expect,
	request,
	type Page,
	type Locator
} from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { LiveAPI } from './api';
import type { Post } from '../../src/lib/api/types';
const fixtures = ['01-coral.png', '02-teal.png'].map((name) =>
	fileURLToPath(
		new URL('../../../app/patrol_test/fixtures/' + name, import.meta.url)
	)
);

async function signIn(page: Page) {
	await page.goto('/sign-in');
	await page.getByLabel('Your Atmosphere Handle').fill(process.env.E2E_HANDLE!);
	const start = page.waitForResponse(
		(response) => new URL(response.url()).pathname === '/api/auth/start'
	);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	const response = await start;
	console.log(`OAuth handoff status: ${response.status()}.`);
	expect(response.ok()).toBe(true);
	// Password entry is allowed only on the same provider origin as the native smoke test.
	await page.waitForURL(
		(url) =>
			url.origin === 'https://bsky.social' &&
			url.pathname === '/oauth/authorize'
	);
	console.log('OAuth provider reached.');
	const password = page.locator('input[type=password]');
	await password.waitFor();
	console.log('OAuth password form ready.');
	if (new URL(page.url()).origin !== 'https://bsky.social')
		throw new Error('Unexpected OAuth provider.');
	await password.fill(process.env.E2E_PASSWORD!);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	console.log('OAuth password submitted.');
	// Consent selectors mirror the provider controls used by Patrol. Do not follow arbitrary prompts.
	await expect(async () => {
		if (new URL(page.url()).origin === process.env.E2E_BASE_URL) return;
		if (new URL(page.url()).origin !== 'https://bsky.social')
			throw new Error('Waiting for OAuth callback');
		const approve = page
			.getByRole('button', { name: /^(Authorize|Accept|Continue)$/ })
			.first();
		if (await approve.isVisible()) await approve.click();
		expect(new URL(page.url()).origin).toBe(process.env.E2E_BASE_URL);
	}).toPass({ timeout: 90_000, intervals: [1000] });
	await expect(
		page.getByRole('heading', { name: 'Set up your CraftSky profile' })
	).toBeVisible();
}

async function onboarding(page: Page, bio: string) {
	await page.getByRole('textbox', { name: /^Bio/ }).fill(bio);
	await page.getByRole('button', { name: 'Save & next', exact: true }).click();
	await expect(page.getByRole('progressbar')).toHaveAttribute(
		'aria-valuenow',
		'2'
	);
	const next = page.getByRole('button', { name: 'Next', exact: true });
	try {
		await next.click();
	} catch (error) {
		// Static control state only: never print page contents, OAuth URLs, or account data.
		const message = error instanceof Error ? error.message : '';
		console.log(
			'Onboarding next state: ' +
				JSON.stringify({
					present: (await next.count()) > 0,
					visible: await next.isVisible(),
					enabled: (await next.count()) > 0 && (await next.isEnabled()),
					saveNext: await page
						.getByRole('button', { name: 'Save & next', exact: true })
						.isVisible(),
					pointerBlocked: message.includes('intercepts pointer events'),
					unstable: message.includes('not stable')
				})
		);
		throw error;
	}
	await page.getByRole('checkbox', { name: /at least 16 years old/ }).check();
	await page.getByRole('button', { name: 'Finish', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
}

async function imageState(image: Locator) {
	await expect(image).toHaveJSProperty('complete', true);
	return image.evaluate(async (node: HTMLImageElement) => {
		await node.decode();
		const canvas = document.createElement('canvas');
		canvas.width = node.naturalWidth;
		canvas.height = node.naturalHeight;
		const context = canvas.getContext('2d')!;
		context.drawImage(node, 0, 0);
		// Inspect decoded pixels without fetching blob: URLs (the app's CSP forbids that).
		const bytes = context.getImageData(0, 0, canvas.width, canvas.height).data;
		return {
			width: node.naturalWidth,
			height: node.naturalHeight,
			hash: Array.from(
				new Uint8Array(await crypto.subtle.digest('SHA-256', bytes))
			).join(','),
			base: Array.from(context.getImageData(10, 10, 1, 1).data).slice(0, 3),
			alternate: Array.from(context.getImageData(50, 10, 1, 1).data).slice(0, 3)
		};
	});
}
async function photos(page: Page) {
	await page.locator('input[type=file][multiple]').setInputFiles(fixtures);
	const details = page.getByRole('region', {
		name: 'Photo details',
		exact: true
	});
	await expect(details).toBeVisible();
	const first = details.getByRole('group', { name: 'Photo 1', exact: true });
	const second = details.getByRole('group', { name: 'Photo 2', exact: true });
	const before = await imageState(first.locator('.photo-preview img'));
	const untouched = await imageState(second.locator('.photo-preview img'));
	for (const [value, base, alternate] of [
		[before, [226, 102, 91], [255, 214, 147]],
		[untouched, [46, 135, 138], [197, 232, 216]]
	] as const) {
		expect([value.width, value.height]).toEqual([640, 480]);
		for (let i = 0; i < 3; i++) {
			expect(Math.abs(value.base[i] - base[i])).toBeLessThanOrEqual(10);
			expect(Math.abs(value.alternate[i] - alternate[i])).toBeLessThanOrEqual(
				10
			);
		}
	}
	// Return to overview and re-enter details, matching Patrol's editor entry point.
	await page.getByRole('button', { name: 'Back to post', exact: true }).click();
	await page.locator('.photo-summary').click();
	await first.getByRole('button', { name: 'Edit photo', exact: true }).click();
	const editor = page.getByRole('dialog', { name: 'Edit photo', exact: true });
	await editor
		.getByRole('button', { name: 'Rotate photo', exact: true })
		.click();
	await editor.getByRole('button', { name: 'Apply edit', exact: true }).click();
	await expect(editor).not.toBeVisible();
	const rotated = await imageState(first.locator('.photo-preview img'));
	expect(rotated.hash).not.toBe(before.hash);
	expect(rotated.width).toBeGreaterThan(0);
	expect(rotated.height).toBeGreaterThan(0);
	expect(await imageState(second.locator('.photo-preview img'))).toEqual(
		untouched
	);
	await page.getByRole('button', { name: 'Back to post', exact: true }).click();
	return { width: rotated.width, height: rotated.height };
}

async function publishedImages(
	page: Page,
	post: Post,
	rotated: { width: number; height: number }
) {
	expect(post.project).toBeFalsy();
	expect(post.images).toHaveLength(2);
	// Download with an independent request context: no account cookies or authorization headers.
	const media = await request.newContext();
	try {
		for (const [index, image] of post.images!.entries()) {
			const url = new URL(image.fullsize!);
			if (
				url.protocol !== 'https:' &&
				!(url.protocol === 'http:' && url.hostname === '127.0.0.1')
			)
				throw new Error('Unexpected media URL');
			const response = await media.get(url.href);
			expect(response.status()).toBe(200);
			const bytes = await response.body();
			expect(bytes.length).toBeGreaterThan(0);
			const size = await page.evaluate(async (data) => {
				const bitmap = await createImageBitmap(
					new Blob([Uint8Array.from(atob(data), (c) => c.charCodeAt(0))])
				);
				const result = { width: bitmap.width, height: bitmap.height };
				bitmap.close();
				return result;
			}, bytes.toString('base64'));
			expect(size).toEqual(index === 0 ? rotated : { width: 640, height: 480 });
		}
	} finally {
		await media.dispose();
	}
}

async function openPost(page: Page, post: Post) {
	await page.goto(
		`/posts/${encodeURIComponent(post.author.did)}/${encodeURIComponent(post.rkey)}`
	);
	const card = page
		.locator('.member-post')
		.filter({ hasText: post.text })
		.first();
	await expect(card).toBeVisible();
	return card;
}

// A separate teardown budget still permits UI cleanup when the journey itself times out.
let cleanupThisRun: (() => Promise<void>) | undefined;
test.afterEach(async ({}, info) => {
	info.setTimeout(150_000);
	try {
		await cleanupThisRun?.();
	} finally {
		cleanupThisRun = undefined;
	}
});

test('OAuth, onboarding bio, rotate photo, two-image post, like, edit and delete (Patrol parity)', async ({
	page,
	context
}) => {
	const run = crypto.randomUUID();
	const bio = `CraftSky web smoke ${run}`,
		text = `CraftSky web E2E smoke ${run}`;
	await test.step('Real browser OAuth', () => signIn(page));
	await test.step('Onboarding bio and policy acceptance', () =>
		onboarding(page, bio));
	const session = (await (await page.request.get('/api/session')).json()) as {
		accounts: { id: string; did: string }[];
		active: string;
	};
	const account = session.accounts.find((item) => item.id === session.active)!;
	expect(account.did).toBe(process.env.E2E_DID);
	const api = new LiveAPI(page, account.id, account.did);
	await test.step('Fresh AppView bio verification', () => api.bio(bio));
	let post: Post | undefined,
		attemptedPublish = false;
	cleanupThisRun = async () => {
		if (attemptedPublish)
			await test.step('Delete only this run’s post through UI and verify eventual 404', async () => {
				// Also recover a publication that indexed just after an earlier assertion failed.
				post ??= await api.published(text);
				api.path(post); // Refuse any record outside the authenticated test account.
				const cleanup = await context.newPage();
				try {
					await cleanup.goto(
						`/posts/${encodeURIComponent(post.author.did)}/${encodeURIComponent(post.rkey)}`
					);
					const card = cleanup.locator('.member-post').first();
					await card.getByLabel('More post actions').click();
					await card
						.getByRole('button', { name: 'Delete', exact: true })
						.click();
					await cleanup
						.getByRole('dialog')
						.getByRole('button', { name: 'Delete', exact: true })
						.click();
					await api.deleted(post);
				} finally {
					await cleanup.close();
				}
			});
	};
	await page.goto('/compose');
	await page
		.getByRole('textbox', { name: 'What would you like to share?' })
		.fill(text);
	const rotated =
		await test.step('Select both Patrol fixtures and rotate the first photo', () =>
			photos(page));
	await test.step('Publish and verify both downloaded images', async () => {
		attemptedPublish = true;
		await page.getByRole('button', { name: 'Post', exact: true }).click();
		post = await api.published(text);
		await publishedImages(page, post, rotated);
	});
	await test.step('Like through UI and verify persistence', async () => {
		const card = await openPost(page, post!);
		await card.getByRole('button', { name: 'Like', exact: true }).click();
		await api.liked(post!);
	});
	await test.step('Edit text, remove second image, retain first CID and like', async () => {
		const card = page.locator('.member-post').filter({ hasText: text }).first();
		await card.getByLabel('More post actions').click();
		await card.getByRole('link', { name: 'Edit', exact: true }).click();
		await page
			.getByRole('textbox', { name: 'What would you like to share?' })
			.fill(text + ' edited');
		await page.locator('.photo-summary').click();
		await page
			.getByRole('button', { name: 'Remove image 2', exact: true })
			.click();
		await page
			.getByRole('button', { name: 'Back to post', exact: true })
			.click();
		await page
			.getByRole('button', { name: 'Save changes', exact: true })
			.click();
		await api.edited(post!, text + ' edited');
	});
});
