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

for (const route of ['/app/messages', '/app/messages/visual-accepted']) {
	test(`loading fills the page from the top: ${route}`, async ({ page }) => {
		await page.setViewportSize({ width: 1376, height: 1032 });
		let release!: () => void;
		const pending = new Promise<void>((resolve) => {
			release = resolve;
		});
		await page.route('**/api/appview/v1/chat/status', async (route) => {
			await pending;
			await route.continue();
		});
		try {
			await page.goto(route);
			const loading = page.locator('.messages-workspace .loading');
			await expect(loading).toBeVisible();
			if (route === '/app/messages') {
				await expect(
					page.getByRole('link', { name: 'Requests', exact: true })
				).toBeVisible();
				await expect(
					page.getByRole('link', {
						name: 'Direct message settings',
						exact: true
					})
				).toBeVisible();
			}

			const skeleton = loading.locator('.inbox-skeleton,.history-skeleton');
			const box = await skeleton.boundingBox();
			expect(box!.y).toBeLessThan(100);
			expect(box!.height).toBeGreaterThan(700);
			await expect(
				page.getByText('You cannot send to this conversation right now.', {
					exact: true
				})
			).toHaveCount(0);
		} finally {
			release();
		}
	});
}
test('message settings load with a spinner rather than skeleton rows', async ({
	page
}) => {
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route('**/api/appview/v1/chat/status', async (route) => {
		await pending;
		await route.continue();
	});
	try {
		await page.goto('/app/messages/settings/');
		await expect(page.locator('.loading-spinner')).toBeVisible();
		await expect(page.locator('.inbox-skeleton,.history-skeleton')).toHaveCount(
			0
		);
	} finally {
		release();
	}
	await expect(
		page.getByText('Enable messaging', { exact: true })
	).toBeVisible();
});
test('Requests has an app-bar back button that returns to the inbox', async ({
	page
}) => {
	await page.goto('/app/messages/requests');
	await page
		.getByRole('link', { name: 'Back to messages', exact: true })
		.click();
	await expect(page).toHaveURL(/\/app\/messages$/);
});
test('marking visible messages read never flashes a false sending restriction', async ({
	page
}) => {
	let read = false;
	await page.route(
		'**/api/appview/v1/chat/conversations/visual-accepted/read',
		async (route) => {
			const response = await route.fetch();
			const value = await response.json();
			read = true;
			await route.fulfill({ json: { ...value, canSend: false } });
		}
	);
	await page.addInitScript(() => {
		(window as any).sawSendingRestriction = false;
		new MutationObserver(() => {
			if (
				document.body?.innerText.includes(
					'You cannot send to this conversation right now.'
				)
			)
				(window as any).sawSendingRestriction = true;
		}).observe(document, {
			subtree: true,
			childList: true,
			characterData: true
		});
	});
	await page.goto('/app/messages/visual-accepted');
	await expect(
		page.getByRole('textbox', { name: 'Message', exact: true })
	).toBeVisible();
	await expect.poll(() => read).toBe(true);
	await expect(
		page.getByRole('textbox', { name: 'Message', exact: true })
	).toBeVisible();
	expect(await page.evaluate(() => (window as any).sawSendingRestriction)).toBe(
		false
	);
});
for (const width of [402, 1376]) {
	test(`conversation and message actions are aligned and bounded at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 874 });
		await page.goto('/app/messages/visual-accepted');
		await page.getByLabel('Conversation actions', { exact: true }).click();
		const actions = page.locator('.conversation-actions .menu-items');
		await expect(actions).toBeVisible();
		await expect(actions.locator('button,a')).toHaveText([
			'Mute conversation',
			'Block account',
			'Report conversation',
			'Leave conversation'
		]);
		await expect(page).toHaveURL(/\/app\/messages\/visual-accepted$/);
		await page.keyboard.press('Escape');
		const summary = page.locator('.chat-message.own summary').first();
		await summary.click();
		const menu = page.locator('.chat-message.own details[open] .menu-items');
		await expect(menu).toBeVisible();
		const box = (await menu.boundingBox())!;
		expect(box.x).toBeGreaterThanOrEqual(0);
		expect(box.y).toBeGreaterThanOrEqual(0);
		expect(box.x + box.width).toBeLessThanOrEqual(width + 1);
		expect(box.y + box.height).toBeLessThanOrEqual(875);
		for (const row of await menu.locator('button,a').all())
			await expect(row).toHaveCSS('text-align', 'left');
		expect(
			await menu.evaluate((node) => {
				const box = node.getBoundingClientRect();
				return node.contains(
					document.elementFromPoint(
						box.x + box.width / 2,
						box.y + box.height / 2
					)
				);
			})
		).toBe(true);
		expect(
			await page.evaluate(
				() => document.documentElement.scrollWidth <= innerWidth
			)
		).toBe(true);
		await page.screenshot({
			path: test.info().outputPath(`message-menu-${width}.png`)
		});
		await page.keyboard.press('Escape');
		await expect(summary).toBeFocused();
	});
}

test('wide page columns keep their width between scrolling feeds and messages', async ({
	page
}) => {
	await page.setViewportSize({ width: 1376, height: 874 });
	await page.goto('/app/home');
	// Give non-overlay scrollbars a known size; force an overflowing feed below.
	await page.addStyleTag({
		content: 'html::-webkit-scrollbar { width: 16px; }'
	});
	await expect(page.locator('.member-post').first()).toBeVisible();
	const columns = () =>
		page.evaluate(() => {
			const main = document.querySelector('main')!.getBoundingClientRect();
			const aside = document
				.querySelector('.app-discovery')!
				.getBoundingClientRect();
			return {
				mainLeft: main.left,
				mainWidth: main.width,
				asideLeft: aside.left,
				asideWidth: aside.width
			};
		});
	await page.locator('main').evaluate((node) => {
		node.style.minHeight = '2000px';
	});
	await expect
		.poll(() =>
			page.evaluate(() => document.documentElement.scrollHeight > innerHeight)
		)
		.toBe(true);
	const home = await columns();
	for (let i = 0; i < 2; i++) {
		await page
			.getByRole('navigation', { name: 'App navigation' })
			.getByRole('link', { name: /^Messages/ })
			.click();
		await expect(
			page.locator('.conversation-list .conversation-row').first()
		).toBeVisible();
		await page.locator('main').evaluate((node) => {
			node.style.minHeight = '';
		});
		expect(await columns()).toEqual(home);
		await page
			.getByRole('navigation', { name: 'App navigation' })
			.getByRole('link', { name: 'Home', exact: true })
			.click();
		await expect(page.locator('.member-post').first()).toBeVisible();
		await page.locator('main').evaluate((node) => {
			node.style.minHeight = '2000px';
		});
		expect(await columns()).toEqual(home);
	}
});

test('inbox navigation icons appear before status resolves when navigating from Home', async ({
	page
}) => {
	await page.goto('/app/home');
	await expect(page.locator('.member-post').first()).toBeVisible();
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route('**/api/appview/v1/chat/status', async (route) => {
		await pending;
		await route.continue();
	});
	try {
		await page
			.getByRole('navigation', { name: 'App navigation' })
			.getByRole('link', { name: /^Messages/ })
			.click();
		await expect(page.locator('.messages-workspace .loading')).toBeVisible();
		await expect(
			page.getByRole('link', { name: 'Requests', exact: true })
		).toBeVisible();
		await expect(
			page.getByRole('link', { name: 'Direct message settings', exact: true })
		).toBeVisible();
		await expect(
			page.getByRole('link', { name: 'Back to messages', exact: true })
		).toHaveCount(0);
	} finally {
		release();
	}
	await expect(page.locator('.messages-workspace .loading')).toHaveCount(0);
	await expect(
		page.getByRole('link', { name: 'Requests', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Direct message settings', exact: true })
	).toBeVisible();
});

test('message input keeps text without a clear button', async ({ page }) => {
	await page.goto('/app/messages/visual-accepted');
	const input = page.getByRole('textbox', { name: 'Message', exact: true });
	await input.fill('A draft to keep');
	await expect(input).toHaveValue('A draft to keep');
	await expect(
		page.getByRole('button', { name: 'Discard draft', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Send message', exact: true })
	).toBeEnabled();
});

for (const route of ['/app/messages', '/app/messages/visual-accepted']) {
	test(`craft icons stay loaded during typing and polling: ${route}`, async ({
		page
	}) => {
		await page.clock.install();
		const profileRequests: string[] = [];
		page.on('request', (request) => {
			if (
				/\/api\/appview\/v1\/profiles\/@did:/.test(
					decodeURIComponent(new URL(request.url()).pathname)
				)
			)
				profileRequests.push(request.url());
		});
		await page.goto(route);
		const icons = page.locator('.messages-workspace .inline-crafts').first();
		await expect(icons).toBeVisible();
		const original = await icons.elementHandle();
		const initialRequests = profileRequests.length;
		expect(initialRequests).toBeGreaterThan(0);
		if (route !== '/app/messages') {
			await page
				.getByRole('textbox', { name: 'Message', exact: true })
				.pressSequentially('Knitting today');
			// Flush rendering before advancing to the idle poll.
			await page.evaluate(() => new Promise(requestAnimationFrame));
			expect(profileRequests).toHaveLength(initialRequests);
			expect(await original!.evaluate((node) => node.isConnected)).toBe(true);
		}
		const refreshed = page.waitForResponse((response) =>
			response
				.url()
				.includes(
					route === '/app/messages'
						? '/chat/conversations?'
						: '/chat/conversations/visual-accepted/details'
				)
		);
		await page.clock.fastForward(10_000);
		await refreshed;
		await page.evaluate(() => new Promise(requestAnimationFrame));
		expect(profileRequests).toHaveLength(initialRequests);
		expect(await original!.evaluate((node) => node.isConnected)).toBe(true);
		await expect(icons).toBeVisible();
	});
}
