import { test, expect } from '@playwright/test';

test('profile bio links open mentions, tags and external confirmation', async ({
	page
}) => {
	await page.goto('/profiles/alice.test');
	const bio = page.locator('.profile-bio');
	await expect(
		bio.getByRole('link', { name: '@bob.test', exact: true })
	).toHaveAttribute('href', '/profiles/bob.test');
	await expect(
		bio.getByRole('link', { name: '#編織', exact: true })
	).toHaveAttribute('href', '/search/tags?tag=%E7%B7%A8%E7%B9%94');
	await bio
		.getByRole('link', { name: 'https://example.org/yarn', exact: true })
		.click();
	const dialog = page.getByRole('dialog', { name: 'Open link?' });
	await expect(dialog).toBeVisible();
	await expect(
		dialog.getByRole('link', { name: 'Open link', exact: true })
	).toHaveAttribute('href', 'https://example.org/yarn');
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await bio.getByRole('link', { name: '@bob.test', exact: true }).click();
	await expect(page).toHaveURL(/profiles\/bob.test$/);
	await expect(
		page.getByRole('heading', { name: 'Bob', exact: true })
	).toBeVisible();
});

// AT-004 / FR-003 / AC-004, AC-006: public profile tabs and multi-author
// threads reuse the regular social UI without creating a viewer account.
test('reads profiles, comment tabs and multi-author post threads as a guest', async ({
	page
}) => {
	const browserFailures: string[] = [];
	page.on('pageerror', (error) => browserFailures.push(error.name));
	await page.goto('/profiles/alice.test');
	await expect(
		page.getByRole('heading', { name: 'Alice', exact: true })
	).toBeVisible();
	await expect(page.getByRole('dialog')).not.toBeVisible();
	await page
		.getByRole('navigation', { name: 'Profile content', exact: true })
		.getByRole('link', { name: /^Comments/ })
		.click();
	await expect(
		page.getByText('A public comment', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Follow', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Message', exact: true })
	).toBeVisible();
	await page.goto('/posts/did%3Aplc%3Aalice/first?show-comments=true');
	await expect(
		page.getByText('A public comment', { exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Bob @bob.test', exact: true })
	).toBeVisible();
	const composer = page.getByRole('link', { name: 'Add a comment' });
	if (page.viewportSize()!.width <= 600)
		await expect(composer).not.toBeVisible();
	else await expect(composer).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Like', exact: true }).first()
	).toBeVisible();
	expect(browserFailures).toEqual([]);
});

test('AT-004 eligible reply permalink survives an unavailable root', async ({
	page
}) => {
	await page.goto('/posts/did%3Aplc%3Abob/reply');
	await expect(
		page.getByText('A public comment', { exact: true }).first()
	).toBeVisible();
	await expect(page).toHaveURL(/bob\/reply/);
	await expect(
		page.getByText('An earlier post is unavailable.', { exact: true })
	).toBeVisible();
});

test('AT-004 public business sections remain available in initial HTML', async ({
	request
}) => {
	for (const [tab, text] of [
		['about', 'Public shop details'],
		['products', 'Public wool kit'],
		['events', 'Public craft fair']
	]) {
		const response = await request.get('/profiles/business.test?tab=' + tab);
		expect(response.status()).toBe(200);
		expect(await response.text()).toContain(text);
		expect(response.headers()['cache-control']).toBe('no-store');
	}
});

// IR-001 / AT-004–005 / FR-003 / AC-004–005: client route reuse cannot
// preserve an earlier subject under a new canonical permalink.
test('canonical guest navigation replaces profile and post subjects', async ({
	page
}) => {
	await page.goto('/profiles/alice.test?tab=reposts');
	await expect(
		page.getByRole('heading', { name: 'Alice', exact: true })
	).toBeVisible();
	await page.getByRole('link', { name: 'Bob @bob.test', exact: true }).click();
	await expect(page).toHaveURL(/profiles\/did%3Aplc%3Abob$/);
	await expect(page).toHaveTitle('Bob · CraftSky');
	await expect(
		page.getByRole('heading', { name: 'Bob', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Alice', exact: true })
	).toHaveCount(0);
	await page.goBack();
	await expect(
		page.getByRole('heading', { name: 'Alice', exact: true })
	).toBeVisible();
	await page.goForward();
	await expect(
		page.getByRole('heading', { name: 'Bob', exact: true })
	).toBeVisible();
	await page.goto('/posts/did%3Aplc%3Aalice/navigation');
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	await page
		.locator('aside[aria-label="Quoted post"] a.summary-content')
		.click();
	await expect(page).toHaveURL(/posts\/did%3Aplc%3Abob\/reply$/);
	await expect(page).toHaveTitle('Post by Bob · CraftSky');
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toHaveCount(0);
	await expect(
		page.getByText('A public comment', { exact: true })
	).toBeVisible();
	await page.goBack();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	await page.goForward();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toHaveCount(0);
});

test('automatic profile pagination merges overlapping posts without losing the profile', async ({
	page
}) => {
	const nextPage = page.waitForResponse(
		(response) =>
			new URL(response.url()).searchParams.get('cursor') ===
			'profile-reposts-next'
	);
	await page.goto('/profiles/alice.test?tab=reposts');
	await nextPage;
	await expect(
		page.getByRole('button', { name: 'Load more', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('heading', { name: 'Alice', exact: true })
	).toBeVisible();
	await expect(page.locator('.member-post')).toHaveCount(1);
});

// IR-002 / IT-010, REG-003 / FR-003, FR-011 / AC-004, AC-011:
// enhancement consumes the successful SSR snapshot without refetching it.
test('successful public SSR stays readable when browser API requests fail', async ({
	page
}) => {
	const duplicateReads: string[] = [];
	await page.route('**/api/public/**', async (route) => {
		const path = new URL(route.request().url()).pathname;
		if (path.includes('/profiles/') || /\/posts\/[^/]+\/[^/]+$/.test(path))
			duplicateReads.push(path);
		await route.fulfill({
			status: 503,
			json: { error: 'unavailable', message: 'Temporary browser read failure' }
		});
	});
	const profile = await page.goto('/profiles/alice.test');
	expect(await profile!.text()).toContain('Making things in public.');
	// These controls are provided by the hydrated shared components.
	await expect(
		page.getByRole('button', { name: 'Follow', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Alice', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	const post = await page.goto('/posts/did%3Aplc%3Aalice/first');
	expect(await post!.text()).toContain('Guest knitting project');
	await expect(
		page.getByText('CraftSky is having trouble right now. Please try again.', {
			exact: true
		})
	).toBeVisible();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Temporary browser read failure', { exact: true })
	).toHaveCount(0);
	expect(duplicateReads).toEqual([]);
});

test('a public profile cursor retains its SSR page after hydration and history navigation', async ({
	page
}) => {
	const response = await page.goto(
		'/profiles/alice.test?cursor=profile-page-two'
	);
	expect(await response!.text()).toContain('Second public profile page');
	await expect(
		page.getByRole('button', { name: 'Follow', exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Second public profile page', { exact: true })
	).toBeVisible();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toHaveCount(0);
	await page.getByRole('link', { name: 'Projects', exact: true }).click();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	await page.goBack();
	await expect(
		page.getByText('Second public profile page', { exact: true })
	).toBeVisible();
	await page.goForward();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
});

// IR-001 / AT-004 / FR-003: a previous profile's outstanding page cannot
// publish into a new canonical subject after its component is disposed.
test('late previous-profile pagination stays isolated after navigation', async ({
	page
}) => {
	let requested!: () => void;
	let release!: () => void;
	const started = new Promise<void>((resolve) => (requested = resolve));
	const barrier = new Promise<void>((resolve) => (release = resolve));
	await page.route('**/api/public/**', async (route) => {
		if (
			new URL(route.request().url()).searchParams.get('cursor') !==
			'profile-reposts-next'
		)
			return route.continue();
		requested();
		await barrier;
		await route.fulfill({ json: { items: [] } });
	});
	try {
		await page.goto('/profiles/alice.test?tab=reposts');
		await expect(page.locator('.member-post').first()).toBeVisible();
		await page.evaluate(() =>
			scrollTo({
				top: document.documentElement.scrollHeight,
				behavior: 'instant'
			})
		);
		await started;
		await page
			.getByRole('link', { name: 'Bob @bob.test', exact: true })
			.click();
		await expect(
			page.getByRole('heading', { name: 'Bob', exact: true })
		).toBeVisible();
		const completed = page.waitForResponse(
			(response) =>
				new URL(response.url()).searchParams.get('cursor') ===
				'profile-reposts-next'
		);
		release();
		await completed;
		await expect(
			page.getByText('A public comment', { exact: true })
		).toBeVisible();
		await expect(
			page.getByRole('heading', { name: 'Bob', exact: true })
		).toBeVisible();
		await expect(
			page.getByRole('heading', { name: 'Alice', exact: true })
		).toHaveCount(0);
		await expect(page).toHaveTitle('Bob · CraftSky');
	} finally {
		release();
	}
});

// IR-005 / IT-010 / FR-003, FR-010, NFR-004 / AC-004, AC-010–011.
for (const navigation of ['author', 'posts'] as const) {
	test(`fresh same-profile loader data replaces identity and excluded quotes via ${navigation}`, async ({
		page,
		request
	}) => {
		await request.get('http://127.0.0.1:19851/_profile-refresh/reset');
		try {
			await page.goto('/profiles/refresh.test');
			await expect(
				page.getByRole('button', { name: 'Follow', exact: true })
			).toBeVisible();
			await expect(
				page.getByText('EXCLUDED_AFTER_REFRESH', { exact: true })
			).toBeVisible();
			await request.get('http://127.0.0.1:19851/_profile-refresh/advance');
			let resolveIncoming!: (payload: string) => void;
			const incoming = new Promise<string>((resolve) => {
				resolveIncoming = resolve;
			});
			await page.route('**/__data.json*', async (route) => {
				const response = await route.fetch();
				const body = await response.text();
				resolveIncoming(body);
				await route.fulfill({ response, body });
			});
			const more = page.waitForRequest(
				(req) =>
					new URL(req.url()).searchParams.get('cursor') === 'fresh-cursor'
			);
			if (navigation === 'author')
				await page
					.locator('a[href="/profiles/did%3Aplc%3Arefresh"]')
					.first()
					.click();
			else
				await page
					.getByRole('navigation', { name: 'Profile content', exact: true })
					.getByRole('link', { name: 'Posts', exact: true })
					.click();
			const payload = await incoming;
			expect(payload).toContain('Updated safe body');
			expect(payload).not.toContain('EXCLUDED_AFTER_REFRESH');
			await expect(page).toHaveTitle('Updated reader · CraftSky');
			await expect(
				page.getByRole('heading', { name: 'Updated reader', exact: true })
			).toBeVisible();
			await expect(
				page.getByText('Updated safe body', { exact: true })
			).toBeVisible();
			await expect(
				page.getByText('Original public body', { exact: true })
			).toHaveCount(0);
			await expect(
				page.getByText('EXCLUDED_AFTER_REFRESH', { exact: true })
			).toHaveCount(0);
			await page.evaluate(() =>
				scrollTo({
					top: document.documentElement.scrollHeight,
					behavior: 'instant'
				})
			);
			await more;
			await expect(
				page.getByRole('button', { name: 'Load more', exact: true })
			).toHaveCount(0);
			await page.reload();
			await expect(
				page.getByText('Updated safe body', { exact: true })
			).toBeVisible();
			await expect(
				page.getByText('EXCLUDED_AFTER_REFRESH', { exact: true })
			).toHaveCount(0);
		} finally {
			await request.get('http://127.0.0.1:19851/_profile-refresh/reset');
		}
	});
}
