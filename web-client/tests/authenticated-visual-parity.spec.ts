import {
	test,
	expect,
	type Page,
	type Request,
	type TestInfo
} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { mkdir } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';

const fixture = 'http://127.0.0.1:19847';
const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const screens = [
	['home', '/app/home'],
	['search', '/app/search'],
	['search-results', '/app/search?q=knitting&tab=posts'],
	['hashtag', '/app/tags/knitting'],
	['activity', '/app/activity'],
	['profile', '/app/profile/did%3Aplc%3Aalice'],
	['profile-handle', '/app/profile/alice.test'],
	['profile-posts', '/app/profile/did%3Aplc%3Aalice?tab=posts'],
	['profile-about', '/app/profile/did%3Aplc%3Aalice?tab=about'],
	['profile-events', '/app/profile/did%3Aplc%3Aalice?tab=events'],
	['profile-visitor', '/app/profile/did%3Aplc%3Abob'],
	['profile-editor', '/app/profile/edit'],
	['thread', '/app/posts/did%3Aplc%3Aalice/first'],
	['project-detail', '/app/posts/did%3Aplc%3Aalice/visual-cardigan'],
	['saved', '/app/saved'],
	['saved-folder', '/app/saved?folder=20c69251-1dcc-4a47-8104-d1877c575f34'],
	['drafts', '/app/drafts'],
	['scheduled', '/app/scheduled'],
	['composer-standard', '/app/compose'],
	['composer-project', '/app/compose/project'],
	['chat-inbox', '/app/messages'],
	['chat-requests', '/app/messages/requests'],
	['chat-new', '/app/messages/new'],
	['chat-conversation', '/app/messages/visual-accepted'],
	['chat-request-conversation', '/app/messages/visual-request'],
	['chat-details', '/app/messages/visual-accepted/details'],
	['chat-report', '/app/messages/visual-accepted/report'],
	['chat-settings', '/app/messages/settings'],
	['business-event', '/app/events/did%3Aplc%3Aalice/visual-workshop'],
	['settings-index', '/app/settings'],
	['settings-appearance', '/app/settings/appearance'],
	['settings-languages', '/app/settings/languages'],
	['settings-notifications', '/app/settings/notifications'],
	['settings-customisation', '/app/settings/customisation'],
	['settings-privacy', '/app/settings/privacy'],
	['settings-messaging', '/app/settings/messaging'],
	['settings-growth', '/app/settings/growth'],
	['settings-followers', '/app/settings/followers'],
	['settings-following', '/app/settings/following'],
	['settings-muted', '/app/settings/muted'],
	['settings-blocked', '/app/settings/blocked'],
	['settings-account', '/app/settings/account'],
	['settings-moderation', '/app/settings/moderation'],
	['settings-about', '/app/settings/about'],
	['settings-billing', '/app/settings/subscriptions'],
	['settings-business', '/app/settings/business'],
	['settings-products', '/app/settings/products'],
	['settings-events', '/app/settings/events'],
	['settings-instagram', '/app/settings/instagram']
] as const;

test.use({
	viewport: { width: 402, height: 874 },
	colorScheme: 'light',
	locale: 'en-GB'
});
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

async function capture(page: Page, testInfo: TestInfo, name: string) {
	const directory = process.env.CRAFTSKY_VISUAL_ARTIFACT_DIR;
	const path = directory
		? resolve(directory, name + '.png')
		: testInfo.outputPath(name + '.png');
	await mkdir(dirname(path), { recursive: true });
	await page.screenshot({ path, fullPage: false });
	await testInfo.attach(name, { path, contentType: 'image/png' });
}

async function settleAccountView(page: Page, route: string) {
	const requests = new Set<Request>();
	const serverFailures: string[] = [];
	page.on('request', (request) => {
		if (request.url().includes('/api/appview/')) requests.add(request);
	});
	page.on('requestfinished', (request) => requests.delete(request));
	page.on('requestfailed', (request) => requests.delete(request));
	page.on('response', (response) => {
		if (response.url().includes('/api/appview/') && response.status() >= 400)
			serverFailures.push(
				response.status() + ' ' + new URL(response.url()).pathname
			);
	});
	await page.goto(route);
	await expect(page.locator('main h1:visible').first()).toBeVisible();
	await expect(
		page.getByText('Loading your CraftSky…', { exact: true })
	).toHaveCount(0);
	await expect.poll(() => requests.size).toBe(0);
	await expect(page.locator('.app-main [aria-busy="true"]')).toHaveCount(0);
	await expect(page.locator('.app-main [role="alert"]')).toHaveCount(0);
	await expect(
		page.getByText(
			/CraftSky returned an incomplete page|Messaging is temporarily unavailable/
		)
	).toHaveCount(0);
	await page.evaluate(() => document.fonts.ready.then(() => undefined));
	expect(
		serverFailures,
		'An account screen must not hide server errors behind its layout'
	).toEqual([]);
}

for (const [name, route] of screens) {
	test(`native-sized layout and accessibility: ${name}`, async ({
		page
	}, testInfo) => {
		await settleAccountView(page, route);
		if (name.startsWith('composer-')) {
			await expect(page.locator('textarea.post-text')).toHaveCSS(
				'font-size',
				'16px'
			);
			await expect(page.locator('.close-composer svg')).not.toHaveCSS(
				'fill',
				'none'
			);
		}
		expect(
			await page.evaluate(
				() => document.documentElement.scrollWidth <= innerWidth
			)
		).toBe(true);
		const navigation = page.getByRole('navigation', { name: 'Main tabs' });
		if (
			![
				'/app/home',
				'/app/search',
				'/app/activity',
				'/app/profile/did%3Aplc%3Aalice',
				'/app/profile/alice.test',
				'/app/messages',
				'/app/messages/requests',
				'/app/messages/new'
			].includes(route.split('?')[0])
		) {
			await expect(navigation).toBeHidden();
		} else {
			await expect(navigation).toBeVisible();
			expect(
				await navigation.evaluate((el) =>
					Math.round(el.getBoundingClientRect().height)
				)
			).toBe(64);
		}
		await capture(page, testInfo, name);
		if (name === 'profile-visitor') {
			await expect(
				page.getByRole('link', { name: 'Back', exact: true })
			).toBeVisible();
		}
		if (name === 'saved-folder') {
			await expect(
				page.getByRole('button', { name: 'Open navigation', exact: true })
			).toBeHidden();
			await expect(
				page.getByRole('link', { name: 'Back to saved posts', exact: true })
			).toBeVisible();
		}
		const audit = await new AxeBuilder({ page })
			.include('.app-main')
			.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
			.analyze();
		expect(audit.violations).toEqual([]);
	});
}

test('native welcome hierarchy and usable hydrated controls', async ({
	page
}, testInfo) => {
	await page.goto('/sign-in');
	await expect(
		page.getByRole('heading', { name: 'Join CraftSky', exact: true })
	).toBeVisible();
	await expect(page.getByLabel('Your Atmosphere Handle')).toBeEnabled();
	await expect(
		page.getByRole('button', { name: 'Register', exact: true })
	).toBeEnabled();
	await expect(
		page.getByRole('button', { name: 'Sign in', exact: true })
	).toBeEnabled();
	await expect(page.locator('.site-header')).toHaveCount(0);
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
	await capture(page, testInfo, 'welcome');
	const audit = await new AxeBuilder({ page })
		.include('main')
		.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
		.analyze();
	expect(audit.violations).toEqual([]);
});

test('native post type chooser opens the selected fullscreen editor', async ({
	page
}) => {
	await settleAccountView(page, '/app/home');
	await page
		.getByRole('button', { name: 'Open navigation', exact: true })
		.click();
	await page
		.getByRole('dialog', { name: 'Navigation', exact: true })
		.getByRole('button', { name: 'Create post', exact: true })
		.click();
	const chooser = page.getByRole('dialog', {
		name: 'What would you like to share?'
	});
	await expect(chooser).toBeVisible();
	await expect(
		page.getByRole('dialog', { name: 'Navigation', exact: true })
	).toHaveCount(0);
	await chooser.getByRole('link', { name: /Project post/ }).click();
	await expect(
		page.getByRole('heading', { name: 'Project post', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('navigation', { name: 'Main tabs' })
	).toBeHidden();
	await expect(page.locator('.kind-switch')).toHaveCount(0);
	await expect(page.locator('.app-main')).not.toHaveAttribute('inert', '');
});

test('native onboarding remains outside the signed-in navigation', async ({
	page
}, testInfo) => {
	await fetch(fixture + '/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ owners: { alice: { completed: false } } })
	});
	await settleAccountView(page, '/app/home');
	await expect(
		page.getByRole('navigation', { name: 'Main tabs' })
	).toBeHidden();
	await expect(page.locator('.app-sidebar')).toBeHidden();
	await expect(
		page.getByRole('button', { name: 'Open navigation' })
	).toBeHidden();
	await capture(page, testInfo, 'onboarding');
});

test('compact navigation and account sheets retain keyboard focus and dismiss with Escape', async ({
	page
}) => {
	await settleAccountView(page, '/app/home');
	const opener = page.getByRole('button', {
		name: 'Open navigation',
		exact: true
	});
	await opener.click();
	const drawer = page.getByRole('dialog', { name: 'Navigation', exact: true });
	await expect(drawer).toBeVisible();
	await expect(page.locator('.app-main')).toHaveAttribute('inert', '');
	await page.keyboard.press('Shift+Tab');
	expect(
		await drawer.evaluate((el) => el.contains(document.activeElement))
	).toBe(true);
	await page.keyboard.press('Escape');
	await expect(drawer).toHaveCount(0);
	await expect(opener).toBeFocused();
	await page
		.getByRole('navigation', { name: 'Main tabs' })
		.getByRole('link', { name: 'Profile', exact: true })
		.click({ button: 'right' });
	const accounts = page.getByRole('dialog', { name: 'Accounts', exact: true });
	await expect(accounts).toBeVisible();
	const audit = await new AxeBuilder({ page })
		.include('.account-menu')
		.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
		.analyze();
	expect(audit.violations).toEqual([]);
	await page.keyboard.press('Escape');
	await expect(accounts).toHaveCount(0);
});

for (const [name, route] of [
	['home', '/app/home'],
	['settings', '/app/settings'],
	['composer', '/app/compose']
] as const) {
	test(`wide dark layout and accessibility: ${name}`, async ({
		page
	}, testInfo) => {
		await page.setViewportSize({ width: 1280, height: 900 });
		await page.emulateMedia({ colorScheme: 'dark', reducedMotion: 'reduce' });
		await settleAccountView(page, route);
		await expect(
			page.getByRole('navigation', { name: 'App navigation' })
		).toBeVisible();
		await expect(
			page.getByRole('navigation', { name: 'Main tabs' })
		).toBeHidden();
		expect(
			await page.evaluate(
				() => document.documentElement.scrollWidth <= innerWidth
			)
		).toBe(true);
		await capture(page, testInfo, 'desktop-dark-' + name);
		const audit = await new AxeBuilder({ page })
			.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
			.analyze();
		expect(audit.violations).toEqual([]);
	});
}

test('saved private draft renders a native compact row and reopens its editor', async ({
	page
}, testInfo) => {
	await settleAccountView(page, '/app/compose');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('A knitted blue cardigan');
	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page.getByText('Draft saved', { exact: true })).toBeVisible();
	await page.goto('/app/drafts');
	await expect(
		page.getByText('A knitted blue cardigan', { exact: true })
	).toBeVisible();
	await capture(page, testInfo, 'drafts-populated');
	await page
		.getByRole('link', { name: 'Continue editing', exact: true })
		.click();
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toHaveValue('A knitted blue cardigan');
});

test('feed bodies expand and collapse without cutting an accessible facet', async ({
	page
}) => {
	const prefix = '🧶'.repeat(295);
	const link = 'https://example.com/pattern';
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		body.items[0].post.text = prefix + link + ' All the pattern details.';
		body.items[0].post.facets = [
			{
				index: {
					byteStart: new TextEncoder().encode(prefix).length,
					byteEnd: new TextEncoder().encode(prefix + link).length
				},
				features: [{ $type: 'app.bsky.richtext.facet#link', uri: link }]
			}
		];
		await route.fulfill({ response, json: body });
	});
	await settleAccountView(page, '/app/home');
	const card = page.locator('.member-post').first();
	await expect(card.getByRole('link', { name: link, exact: true })).toHaveCount(
		0
	);
	await card.getByRole('button', { name: 'Show more', exact: true }).click();
	await expect(
		card.getByRole('link', { name: link, exact: true })
	).toHaveAttribute('href', link);
	await expect(card).toContainText('All the pattern details.');
	await card.getByRole('button', { name: 'Show less', exact: true }).click();
	await expect(card).not.toContainText('All the pattern details.');
	await card.locator('.prose').click();
	await expect(page).toHaveURL(/\/app\/posts\/did%3Aplc%3Aalice\/first$/);
});

test('profile header collapses to a readable identity and retains Back navigation', async ({
	page
}, testInfo) => {
	await page.route('**/api/appview/v1/profiles/*/posts*', async (route) => {
		const response = await route.fetch();
		const body = await response.json();
		body.items = Array.from({ length: 5 }, (_, index) => ({
			...body.items[0],
			uri: `at://did:plc:bob/social.craftsky.feed.post/visual-${index}`,
			rkey: 'visual-' + index
		}));
		await route.fulfill({ response, json: body });
	});
	await settleAccountView(page, '/app/profile/did%3Aplc%3Abob?tab=posts');
	await page.evaluate(() => window.scrollTo(0, 500));
	await expect(page.locator('.compact-profile-identity')).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Back', exact: true })
	).toBeInViewport();
	await capture(page, testInfo, 'profile-collapsed');
	await page.evaluate(() => window.scrollTo(0, 0));
	await expect(page.locator('.profile-compact-header')).toHaveCount(0);
});

test('project detail page has the native compact app bar and accessible fields', async ({
	page
}, testInfo) => {
	await settleAccountView(page, '/app/compose/project');
	await page
		.getByRole('button', { name: 'Pattern details', exact: true })
		.click();
	const detail = page.getByRole('dialog', {
		name: 'Pattern details',
		exact: true
	});
	await expect(detail).toBeVisible();
	await expect(
		detail.getByRole('button', { name: 'Back to project', exact: true })
	).toBeVisible();
	expect(
		await detail
			.locator('header')
			.evaluate((el) => Math.round(el.getBoundingClientRect().height))
	).toBe(56);
	await capture(page, testInfo, 'composer-project-details');
	const audit = await new AxeBuilder({ page })
		.include('.project-detail-dialog[open]')
		.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
		.analyze();
	expect(audit.violations).toEqual([]);
});
