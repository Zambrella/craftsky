import { test, expect } from '@playwright/test';
test('AT-005 guest engagement identities and project feed controls remain usable', async ({
	page
}) => {
	await page.goto('/posts/did:plc:alice/post1/likes');
	await expect(
		page.getByRole('heading', { name: /^Likes(?: \(\d+\))?$/, level: 1 })
	).toBeVisible();
	await expect(page.getByRole('link', { name: /Alice/ }).first()).toBeVisible();
	await expect(
		page.getByRole('button', { name: /Follow|Like post|Repost/ })
	).toHaveCount(0);
	await page.goto('/feed?craft=sewing&sort=popular');
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	await page.getByRole('button', { name: 'Filter projects' }).click();
	await expect(
		page.getByRole('heading', { name: /Filter .* projects/ })
	).toBeVisible();
	await page.getByRole('button', { name: /Apply/ }).click();
	await expect(page).toHaveURL(/\/feed\?/);
});

// AT-001 / NFR-003: public list state is scoped to the guest lifetime.
test('Back restores the already loaded public feed without refetching', async ({
	page
}) => {
	let reads = 0;
	page.on('request', (request) => {
		if (new URL(request.url()).pathname === '/api/public/projects') reads++;
	});
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	const initial = reads;
	await page.getByRole('link', { name: /Alice/ }).first().click();
	await expect(
		page.getByRole('heading', { name: 'Alice', exact: true })
	).toBeVisible();
	await page.goBack();
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	expect(reads).toBe(initial);
});

test('guest search category tabs retain already loaded results', async ({
	page
}) => {
	let posts = 0;
	page.on('request', (request) => {
		if (new URL(request.url()).pathname === '/api/public/search/posts') posts++;
	});
	await page.goto('/search?q=knitting');
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	await page.getByRole('link', { name: 'Projects', exact: true }).click();
	await expect(page).toHaveURL(/tab=projects/);
	await expect(page.locator('.social-loading')).toHaveCount(0);
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	await page.getByRole('link', { name: 'Posts', exact: true }).click();
	await expect(page).toHaveURL(/tab=posts/);
	await expect(page.locator('.social-loading')).toHaveCount(0);
	await expect(
		page.getByText('Guest knitting project', { exact: true })
	).toBeVisible();
	expect(posts).toBe(1);
});

test('AT-001 changing craft tabs changes the public feed request', async ({
	page
}) => {
	await page.goto('/feed?craft=sewing&sort=popular');
	await page.getByRole('button', { name: 'Explore CraftSky' }).click();
	const request = page.waitForRequest((request) => {
		const url = new URL(request.url());
		return (
			url.pathname === '/api/public/projects' &&
			url.searchParams
				.getAll('craftType')
				.some((craft) => craft.endsWith('knitting'))
		);
	});
	await page.getByRole('link', { name: 'Knitting', exact: true }).click();
	await request;
	await expect(page).toHaveURL(
		(url) =>
			url.searchParams.get('craft') === 'social.craftsky.feed.defs#knitting'
	);
	await expect(
		page.getByLabel('Sort projects: Newest', { exact: true })
	).toBeVisible();
});
