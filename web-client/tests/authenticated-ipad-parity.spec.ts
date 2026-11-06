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

for (const width of [402, 1376]) {
	test(`inbox identities and floating action match native at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 1032 });
		await page.goto('/messages');
		const inbox = page.locator('.conversation-list');
		await expect(inbox.locator('.compact-identity').first()).toBeVisible();
		await expect(inbox.locator('.inline-crafts').first()).toBeVisible();
		const button = page.getByRole('link', { name: 'New message', exact: true });
		const bounds = await button.boundingBox();
		expect(bounds).not.toBeNull();
		expect(bounds!.y).toBeGreaterThan(1032 - 180);
		expect(bounds!.height).toBeGreaterThanOrEqual(56);
		expect(
			await page.evaluate(
				() => document.documentElement.scrollWidth <= innerWidth
			)
		).toBe(true);
	});
}
test('page empty states center in the viewport and keep native copy', async ({
	page
}) => {
	await page.setViewportSize({ width: 1376, height: 1032 });
	for (const [route, detail] of [
		['drafts', 'No drafts'],
		['scheduled', 'No scheduled posts']
	]) {
		await page.goto('/' + route);
		// Scheduled fixture data is cleared so this exercises the empty state.
		if (route === 'scheduled') {
			await page.route('**/api/appview/v1/scheduled-posts*', (route) =>
				route.fulfill({ json: { items: [] } })
			);
			await page.reload();
		}
		const empty = page.locator('.social-empty.fill');
		await expect(empty.getByText(detail, { exact: true })).toBeVisible();
		const box = await empty.locator('.empty-icon').boundingBox();
		expect(box!.y).toBeGreaterThan(400);
		expect(box!.y).toBeLessThan(600);
	}
});
test('late profile previews stay inside bubbles and keep the latest message visible', async ({
	page
}) => {
	await page.setViewportSize({ width: 1376, height: 1032 });
	let release!: () => void;
	const previewReady = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route(
		(url) => decodeURIComponent(url.pathname).endsWith('did:plc:preview'),
		async (route) => {
			await previewReady;
			await route.fulfill({
				json: {
					did: 'did:plc:preview',
					handle: 'preview.test',
					displayName: 'Preview Maker',
					crafts: ['knitting'],
					description:
						'A profile preview that loads after the initial message history. '.repeat(
							10
						),
					links: [],
					viewerIsFollowing: false,
					isCraftskyProfile: true
				}
			});
		}
	);
	const messages = Array.from({ length: 40 }, (_, i) => ({
		id: 'delayed-' + i,
		revision: '1',
		kind: 'text',
		senderDid: 'did:plc:bob',
		text:
			i === 20
				? 'https://app.craftsky.social/profiles/did%3Aplc%3Apreview'
				: 'Message ' + i,
		sentAt: new Date(Date.UTC(2026, 10, 2, 10, i)).toISOString()
	}));
	await page.route(
		'**/api/appview/v1/chat/conversations/visual-accepted/messages*',
		(route) => route.fulfill({ json: { items: messages } })
	);
	await page.route(
		'**/api/appview/v1/chat/conversations/visual-accepted/details',
		async (route) => {
			const response = await route.fetch();
			const value = await response.json();
			await route.fulfill({
				json: { ...value, messages: { items: messages } }
			});
		}
	);
	await page.route(
		'**/api/appview/v1/chat/conversations/visual-accepted/read',
		async (route) => {
			const response = await page.request.get(
				'/api/appview/v1/chat/conversations/visual-accepted'
			);
			await route.fulfill({ json: await response.json() });
		}
	);
	await page.goto('/messages/conversations/visual-accepted');
	await expect(page.locator('.peer-heading small')).toHaveCSS(
		'font-weight',
		'400'
	);
	const history = page.locator('.message-history');
	const remaining = () =>
		history.evaluate(
			(node) => node.scrollHeight - node.scrollTop - node.clientHeight
		);
	await expect(page.locator('.chat-message')).toHaveCount(40);
	await expect.poll(remaining).toBeLessThan(3);
	release();
	await expect(page.locator('.message-bubble .profile-preview')).toHaveCount(1);
	await expect(page.locator('.profile-preview')).toContainText('Preview Maker');
	await expect(page.locator('.profile-preview')).toContainText('@preview.test');
	await expect.poll(remaining).toBeLessThan(3);
	await history.hover();
	await page.mouse.wheel(0, -500);
	await expect.poll(remaining).toBeGreaterThan(300);
	const priorTop = await history.evaluate((node) => node.scrollTop);
	await page.locator('.profile-preview').evaluate(async (node) => {
		(node as HTMLElement).style.paddingBottom = '80px';
		await new Promise<void>((resolve) =>
			requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
		);
	});
	expect(await history.evaluate((node) => node.scrollTop)).toBeCloseTo(
		priorTop,
		0
	);

	await page.screenshot({
		path: test.info().outputPath('ipad-web-conversation.png')
	});
});

test('video uploads are available without a release flag', async ({ page }) => {
	await page.goto('/compose');
	await expect(
		page.getByText('Add photos or a video', { exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Choose up to 8 photos or a video.', { exact: true })
	).toBeVisible();
	await expect(
		page.getByLabel('Choose a video', { exact: true })
	).toHaveAttribute('accept', /video\/mp4/);
});
