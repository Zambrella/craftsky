import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const maker = (crafts: string[] | null | undefined) => ({
	did: 'did:plc:bob',
	handle: 'bob.test',
	displayName: 'Bob Maker',
	crafts
});
const tag = { tag: 'CraftSkyTip', postCount: 2 };
const grouped = (items: ReturnType<typeof maker>[] = [], hasMore = false) => ({
	profiles: { items, hasMore },
	hashtags: { items: [tag, { tag: 'craftbeer', postCount: 1 }], hasMore }
});

test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
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

for (const [name, crafts] of [
	['null', null],
	['empty', []],
	['omitted', undefined]
] as const) {
	test(`autocomplete and submitted searches render a profile with ${name} crafts`, async ({
		page
	}) => {
		const errors: string[] = [];
		page.on('pageerror', (error) => errors.push(error.message));
		const author = maker(crafts as string[] | null | undefined);
		await page.route('**/api/appview/v1/search/suggestions*', (route) =>
			route.fulfill({ json: grouped([author]) })
		);
		await page.route('**/api/appview/v1/search/profiles*', (route) =>
			route.fulfill({ json: { items: [author] } })
		);
		await page.goto('/search');
		await page.getByRole('searchbox', { name: 'Search CraftSky' }).fill('bob');
		await expect(
			page.getByRole('region', { name: 'Profiles' }).getByText('Bob Maker')
		).toBeVisible();
		await expect(
			page.getByRole('link', { name: 'View all', exact: true })
		).toHaveCount(0);
		await page.goto('/search?q=bob&tab=profiles');
		await expect(
			page.locator('.maker-result').getByText('Bob Maker')
		).toBeVisible();
		await expect(page.getByText('We couldn’t load this page.')).toHaveCount(0);
		expect(errors).toEqual([]);
	});
}

test('autocomplete matches native grouping, counts, result limits and discovery visibility', async ({
	page
}) => {
	let request!: URL;
	await page.route('**/api/appview/v1/search/recent', (route) =>
		route.fulfill({
			json: {
				items: [
					{
						id: 'recent',
						type: 'query',
						displayLabel: 'Past search',
						payload: { q: 'past' }
					}
				]
			}
		})
	);
	await page.route('**/api/appview/v1/search/suggestions*', (route) => {
		request = new URL(route.request().url());
		return route.fulfill({
			json: grouped(
				[
					maker(null),
					{
						...maker(['knitting']),
						did: 'did:plc:carol',
						handle: 'carol.test',
						displayName: 'Carol Maker'
					}
				],
				true
			)
		});
	});
	await page.goto('/search');
	await expect(
		page.getByText('Recent searches', { exact: true })
	).toBeVisible();
	await page.getByRole('searchbox', { name: 'Search CraftSky' }).fill('c');
	const profiles = page.getByRole('region', { name: 'Profiles', exact: true });
	const hashtags = page.getByRole('region', { name: 'Hashtags', exact: true });
	await expect(profiles.getByText('Bob Maker')).toBeVisible();
	await expect(profiles.getByText('Carol Maker')).toBeVisible();
	await expect(hashtags.getByText('2 posts', { exact: true })).toBeVisible();
	await expect(hashtags.getByText('1 post', { exact: true })).toBeVisible();
	await expect(hashtags.locator('.hashtag-icon')).toHaveCount(2);
	await expect(page.getByText('Recent searches', { exact: true })).toHaveCount(
		0
	);
	await expect(page.getByText('New makers', { exact: true })).toHaveCount(0);
	await expect(page.locator('.suggestions')).toHaveCSS(
		'background-color',
		'rgba(0, 0, 0, 0)'
	);
	await expect(
		page.getByRole('button', { name: 'Search', exact: true })
	).toHaveClass(/\bvisually-hidden\b/);
	expect(request.searchParams.get('profileLimit')).toBe('5');
	expect(request.searchParams.get('hashtagLimit')).toBe('5');
	await expect(
		hashtags.getByRole('link', { name: 'View all' })
	).toHaveAttribute('href', '/search?q=c&tab=hashtags');
	await page.evaluate(() => document.fonts.ready);
	for (const colorScheme of ['light', 'dark'] as const) {
		await page.emulateMedia({ colorScheme });
		const audit = await new AxeBuilder({ page })
			.include('.search-form')
			.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
			.analyze();
		expect(audit.violations).toEqual([]);
	}
	await page.emulateMedia({ colorScheme: 'light' });
	await page.screenshot({ path: '/tmp/craftsky-search-fixed-desktop.png' });
	await page.setViewportSize({ width: 402, height: 874 });
	await page.screenshot({ path: '/tmp/craftsky-search-fixed-mobile.png' });
	await page.setViewportSize({ width: 1280, height: 900 });
	await profiles.getByRole('link', { name: 'View all' }).click();
	await expect(page).toHaveURL(/\/search\?q=c&tab=profiles$/);
	await expect(
		page.getByRole('navigation', { name: 'Search categories' })
	).toBeVisible();
});

test('empty autocomplete stays on search and clearing restores discovery', async ({
	page
}) => {
	await page.route('**/api/appview/v1/search/suggestions*', (route) =>
		route.fulfill({
			json: {
				profiles: { items: [], hasMore: false },
				hashtags: { items: [], hasMore: false }
			}
		})
	);
	await page.goto('/search');
	await page
		.getByRole('searchbox', { name: 'Search CraftSky' })
		.fill('no matches');
	await expect(
		page.getByText('No results found', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Search', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('region', { name: 'Profiles', exact: true })
	).toHaveCount(0);
	await page.getByRole('button', { name: 'Clear search' }).click();
	await expect(
		page.getByRole('searchbox', { name: 'Search CraftSky' })
	).toBeFocused();
	await expect(page.locator('.suggestions')).toHaveCount(0);
	await expect(page.getByText('New makers', { exact: true })).toBeVisible();
});

test('late suggestions cannot replace a newer query or reappear after clear', async ({
	page
}) => {
	let release!: () => Promise<void>;
	let started!: () => void;
	let waiting = new Promise<void>((resolve) => (started = resolve));
	await page.route('**/api/appview/v1/search/suggestions*', (route) => {
		const q = new URL(route.request().url()).searchParams.get('q');
		if (q === 'old') {
			release = () =>
				route.fulfill({
					json: grouped([{ ...maker([]), displayName: 'OLD RESULT' }])
				});
			started();
			return;
		}
		return route.fulfill({
			json: grouped([{ ...maker(null), displayName: 'NEW RESULT' }])
		});
	});
	await page.goto('/search');
	const input = page.getByRole('searchbox', { name: 'Search CraftSky' });
	await input.fill('old');
	await waiting;
	await input.fill('new');
	await expect(page.getByText('NEW RESULT', { exact: true })).toBeVisible();
	await release();
	await expect(page.getByText('OLD RESULT', { exact: true })).toHaveCount(0);
	waiting = new Promise<void>((resolve) => (started = resolve));
	await input.fill('old');
	await waiting;
	await page.getByRole('button', { name: 'Clear search' }).click();
	await release();
	await expect(page.locator('.suggestions')).toHaveCount(0);
	await expect(input).toHaveValue('');
});

for (const status of [400, 500]) {
	test(`failed autocomplete ${status} offers retry and respects diagnostic classification`, async ({
		page
	}) => {
		test.skip(
			process.env.WEB_CLIENT_DIAGNOSTICS !== '1',
			'Requires a build with the loopback diagnostic DSN'
		);
		const envelopes: string[] = [];
		await page.addInitScript(() =>
			localStorage.setItem('craftsky.diagnostics-consent', 'true')
		);
		await page.route(
			'http://127.0.0.1:19846/api/1/envelope/**',
			async (route) => {
				envelopes.push(route.request().postData() || '');
				await route.fulfill({ status: 200, body: '{}' });
			}
		);
		let failed = true;
		await page.route('**/api/appview/v1/search/suggestions*', (route) =>
			route.fulfill(
				failed
					? {
							status,
							headers: {
								'X-Request-ID': '12345678-1234-4123-8123-123456789012'
							},
							json: {
								error:
									status === 500 ? 'search_unavailable' : 'validation_error',
								message: 'PRIVATE_SEARCH_FAILURE_CANARY'
							}
						}
					: { json: grouped([maker(null)]) }
			)
		);
		await page.goto('/search');
		await page
			.getByRole('searchbox', { name: 'Search CraftSky' })
			.fill('PRIVATE_QUERY_CANARY');
		await expect(
			page.getByRole('button', { name: 'Retry', exact: true })
		).toBeVisible();
		if (status === 500) {
			await expect.poll(() => envelopes.length).toBe(1);
			expect(envelopes[0]).toContain('"httpStatus":500');
			expect(envelopes[0]).toContain('12345678-1234-4123-8123-123456789012');
			expect(envelopes[0]).toContain('"operation":"social.Search.suggest"');
			expect(envelopes[0]).not.toContain('PRIVATE_SEARCH_FAILURE_CANARY');
			expect(envelopes[0]).not.toContain('PRIVATE_QUERY_CANARY');
		} else expect(envelopes).toHaveLength(0);
		failed = false;
		await page.getByRole('button', { name: 'Retry', exact: true }).click();
		await expect(
			page.getByRole('region', { name: 'Profiles' }).getByText('Bob Maker')
		).toBeVisible();
		await expect(
			page.getByRole('button', { name: 'Retry', exact: true })
		).toHaveCount(0);
		expect(envelopes).toHaveLength(status === 500 ? 1 : 0);
	});
}

test('Cancel clears autocomplete on the current search URL and Enter submits', async ({
	page
}) => {
	await page.route('**/api/appview/v1/search/suggestions*', (route) =>
		route.fulfill({ json: grouped([maker(null)]) })
	);
	await page.goto('/search');
	const input = page.getByRole('searchbox', { name: 'Search CraftSky' });
	await input.fill('bob');
	await expect(page.getByRole('region', { name: 'Profiles' })).toBeVisible();
	await page.getByRole('link', { name: 'Cancel', exact: true }).click();
	await expect(input).toHaveValue('');
	await expect(page.locator('.suggestions')).toHaveCount(0);
	await input.fill('@bob');
	await input.press('Enter');
	await expect(page).toHaveURL('/search?q=%40bob&tab=profiles');
	await expect(page.locator('.suggestions')).toHaveCount(0);
});
