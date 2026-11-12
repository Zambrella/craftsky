import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const maker = {
	did: 'did:plc:bob',
	handle: 'bob.test',
	displayName: 'Bob Maker',
	crafts: ['social.craftsky.feed.defs#knitting', 'crochet']
};
const post = (rkey = 'first', overrides: Record<string, unknown> = {}) => ({
	uri: `at://${maker.did}/social.craftsky.feed.post/${rkey}`,
	cid: 'cid-' + rkey,
	rkey,
	text: 'Weekend making.',
	author: maker,
	createdAt: '2026-11-01T12:00:00Z',
	indexedAt: '2026-11-01T12:00:00Z',
	tags: [],
	langs: ['en'],
	sponsored: false,
	likeCount: 0,
	repostCount: 0,
	quoteCount: 0,
	replyCount: 0,
	viewerHasLiked: false,
	viewerHasReposted: false,
	viewerHasReplied: false,
	viewerHasSaved: false,
	...overrides
});
async function timeline(page: Page, value: ReturnType<typeof post>) {
	await page.route('**/api/appview/v1/posts/**/comments*', (route) =>
		route.fulfill({ json: { comments: { items: [], cursor: null } } })
	);
	await page.route(/\/api\/appview\/v1\/posts\/[^/]+\/[^/?]+$/, (route) => {
		const parts = new URL(route.request().url()).pathname
			.split('/')
			.map(decodeURIComponent);
		return parts.at(-1) === value.rkey && parts.at(-2) === value.author.did
			? route.fulfill({ json: value })
			: route.continue();
	});
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({ json: { items: [{ itemKey: 'stable', post: value }] } })
	);
}

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

test('hashtag Back returns to the originating feed or post and restores paginated feed position', async ({
	page
}) => {
	const runtimeErrors: string[] = [];
	await page.exposeFunction('recordNavigationError', (name: string) =>
		runtimeErrors.push(name)
	);
	await page.addInitScript(() => {
		const record = (error: unknown) => {
			void (
				window as unknown as {
					recordNavigationError: (name: string) => Promise<void>;
				}
			).recordNavigationError(
				error instanceof Error ? error.name : 'UnknownError'
			);
		};
		// The app's privacy handler suppresses native error reporting. Observe the
		// original events too, without recording fixture messages or payloads.
		window.addEventListener('error', (event) => record(event.error));
		window.addEventListener('unhandledrejection', (event) =>
			record(event.reason)
		);
	});
	const values = Array.from({ length: 20 }, (_, i) =>
		post('scroll-' + i, {
			text: '#sewing ' + 'A detailed craft update. '.repeat(20),
			facets: [
				{
					index: { byteStart: 0, byteEnd: 7 },
					features: [{ $type: 'app.bsky.richtext.facet#tag', tag: 'sewing' }]
				}
			]
		})
	);
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		await new Promise((resolve) => setTimeout(resolve, 150));
		const more = new URL(route.request().url()).searchParams.has('cursor');
		await route.fulfill({
			json: {
				items: values
					.slice(more ? 10 : 0, more ? 20 : 10)
					.map((value) => ({ itemKey: value.rkey, post: value })),
				cursor: more ? null : 'page-2'
			}
		});
	});
	await page.route(/\/api\/appview\/v1\/posts\/[^/]+\/[^/?]+$/, (route) =>
		route.fulfill({
			json:
				values.find((value) => route.request().url().endsWith(value.rkey)) ||
				values[0]
		})
	);
	await page.route('**/api/appview/v1/posts/**/comments*', (route) =>
		route.fulfill({ json: { comments: { items: [], cursor: null } } })
	);
	await page.route('**/api/appview/v1/search/hashtags/sewing/posts*', (route) =>
		route.fulfill({
			status: 503,
			json: { error: 'unavailable', message: 'Temporary hashtag outage.' }
		})
	);
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(20);
	const card = page.locator('.member-post').nth(15);
	await card.scrollIntoViewIfNeeded();
	const y = await page.evaluate(() => scrollY);
	await card.getByRole('link', { name: '#sewing', exact: true }).click();
	await expect(page).toHaveURL(/\/search\/tags\?tag=sewing$/);
	await expect(page.locator('.snackbar.warning')).toContainText(
		'CraftSky is having trouble right now. Please try again.'
	);
	await expect(page.locator('body')).not.toContainText(
		'Temporary hashtag outage.'
	);
	await page.getByRole('link', { name: 'Back', exact: true }).click();
	await expect(page).toHaveURL(/\/feed$/);
	await expect(page.locator('.member-post')).toHaveCount(20);
	await expect(page.locator('.snackbar.warning')).toHaveCount(0);
	await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(y, 0);
	await card.getByRole('link', { name: 'Open post', exact: true }).click();
	await expect(page).toHaveURL(/scroll-15$/);
	await page
		.locator('.member-post')
		.first()
		.getByRole('link', { name: '#sewing', exact: true })
		.click();
	await page.getByRole('link', { name: 'Back', exact: true }).click();
	await expect(page).toHaveURL(/scroll-15$/);
	await page.getByRole('link', { name: 'Back', exact: true }).click();
	await expect(page).toHaveURL(/\/feed$/);
	await expect(page.locator('.member-post')).toHaveCount(20);
	await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(y, 0);
	await page.goForward();
	await expect(page).toHaveURL(/scroll-15$/);
	await page.goBack();
	await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(y, 0);
	expect(runtimeErrors).toEqual([]);
});

test('pattern text permits deleting the hash and plain names, with the self-drafted switch trailing', async ({
	page
}) => {
	await page.goto('/compose/project');
	const field = page.getByRole('textbox', {
		name: 'Pattern tag or name',
		exact: true
	});
	await field.fill('#');
	await expect(field).toHaveValue('#');
	await field.press('Backspace');
	await expect(field).toHaveValue('');
	await field.fill('Falconer pants');
	await page
		.getByRole('button', { name: 'Pattern details', exact: true })
		.click();
	const toggle = page.getByRole('switch', {
		name: 'Self-drafted pattern',
		exact: true
	});
	const label = page.locator('.switch-row > span');
	const switchBox = (await toggle.boundingBox())!;
	const labelBox = (await label.boundingBox())!;
	expect(switchBox.x).toBeGreaterThan(labelBox.x + labelBox.width);
	expect(
		Math.abs(
			switchBox.y + switchBox.height / 2 - (labelBox.y + labelBox.height / 2)
		)
	).toBeLessThan(2);
	await toggle.check();
	await page
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	await expect(field).toHaveValue('Falconer pants');
	await field.fill('#falconer');
	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page).toHaveURL(/\/feed$/);
	await page.getByRole('link', { name: 'Drafts', exact: true }).click();
	await page
		.getByRole('link', { name: 'Continue editing', exact: true })
		.first()
		.click();
	await expect(field).toHaveValue('#falconer');
});

test('comments use flat cards and shaded reply branches with working controls in both themes', async ({
	page
}) => {
	const root = post('root', { likeCount: 15, replyCount: 2 });
	const reference = { uri: root.uri, cid: root.cid };
	const comment = post('comment', {
		text: 'A lovely project!',
		replyCount: 1,
		reply: { root: reference, parent: reference }
	});
	const reply = post('reply', {
		text: 'Thank you for your kind comment.',
		reply: { root: reference, parent: { uri: comment.uri, cid: comment.cid } }
	});
	await timeline(page, root);
	await page.route('**/api/appview/v1/posts/**/comments*', (route) =>
		route.fulfill({
			json: {
				comments: {
					items: [
						{
							post: comment,
							replies: { loaded: true, items: [{ post: reply }], cursor: null }
						}
					],
					cursor: null
				}
			}
		})
	);
	await page.goto('/posts/did%3Aplc%3Abob/root');
	const branch = page.locator('.replies');
	await expect(branch).toBeVisible();
	const compose = page.getByRole('link', {
		name: 'Add a comment',
		exact: true
	});
	await expect(compose).toHaveCSS('border-radius', '999px');
	await expect(page.locator('.comment .post-actions').first()).toHaveCSS(
		'border-top-width',
		'0px'
	);
	await expect(page.locator('.comment')).toHaveCSS(
		'border-bottom-width',
		'0px'
	);
	await expect(branch).toHaveCSS('border-radius', '14px');
	for (const theme of ['light', 'dark']) {
		await page.evaluate(
			(theme) => (document.documentElement.dataset.theme = theme),
			theme
		);
		await compose.scrollIntoViewIfNeeded();
		await page.evaluate(() => document.fonts.ready);
		expect(
			(
				await new AxeBuilder({ page })
					.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
					.analyze()
			).violations
		).toEqual([]);
		await page.screenshot({
			path: test.info().outputPath('comments-' + theme + '.png')
		});
	}
	await page.getByRole('button', { name: 'Hide replies', exact: true }).click();
	await expect(branch).toHaveCount(0);
	await page.getByRole('button', { name: 'Show 1 reply', exact: true }).click();
	await expect(branch).toBeVisible();
	await expect(page.locator('.comment .post-card').first()).toHaveCSS(
		'cursor',
		'auto'
	);
	await branch.getByRole('link', { name: 'Reply', exact: true }).click();
	await expect(page).toHaveURL(/compose\/reply/);
});

test('changing the mobile tag query reloads results within the same route', async ({
	page
}) => {
	const tags: string[] = [];
	await page.route('**/api/appview/v1/search/hashtags/*/posts*', (route) => {
		const tag = new URL(route.request().url()).pathname.split('/').at(-2)!;
		tags.push(tag);
		return route.fulfill({
			json: {
				items: [
					post('tagged', {
						text: '#crochet',
						facets: [
							{
								index: { byteStart: 0, byteEnd: 8 },
								features: [
									{ $type: 'app.bsky.richtext.facet#tag', tag: 'crochet' }
								]
							}
						]
					})
				]
			}
		});
	});
	await page.goto('/search/tags?tag=sewing');
	await expect(
		page.getByRole('heading', { name: '#sewing', exact: true })
	).toBeVisible();
	await page
		.locator('.post-card')
		.getByRole('link', { name: '#crochet', exact: true })
		.click();
	await expect(page).toHaveURL('/search/tags?tag=crochet');
	await expect(
		page.getByRole('heading', { name: '#crochet', exact: true })
	).toBeVisible();
	expect(tags).toEqual(['sewing', 'crochet']);
});

for (const label of ['Followers', 'Following']) {
	test(`visitor profile menu opens and paginates ${label}`, async ({
		page
	}) => {
		await fetch('http://127.0.0.1:19847/__test/control', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ visual: true })
		});
		await page.goto('/profiles/did%3Aplc%3Abob');
		await page.getByLabel('More profile actions', { exact: true }).click();
		await page.getByRole('link', { name: label, exact: true }).click();
		await expect(page).toHaveURL(
			new RegExp('/profiles/did%3Aplc%3Abob/' + label.toLowerCase() + '$')
		);
		await expect(
			page.getByRole('heading', { name: label + ' (2)', exact: true })
		).toBeVisible();
		const paginatedMember = page
			.getByRole('main')
			.getByRole('link')
			.filter({ hasText: 'Martha Jones' });
		await expect(paginatedMember).toHaveCount(0);
		await page.getByRole('button', { name: 'Load more', exact: true }).click();
		await expect(paginatedMember).toBeVisible();
		await expect(
			page.getByRole('button', { name: 'Load more', exact: true })
		).not.toBeVisible();
		await page
			.getByRole('link', { name: 'Back to profile', exact: true })
			.click();
		await expect(page).toHaveURL(/profiles\/did%3Aplc%3Abob$/);
		await expect(
			page.getByRole('heading', { name: 'Bob Maker', exact: true })
		).toBeVisible();
	});
}

for (const width of [1280, 390]) {
	for (const kind of ['followers', 'following']) {
		test(`profile ${kind} matches Settings layout at ${width}px`, async ({
			page
		}) => {
			await page.setViewportSize({ width, height: 900 });
			await page.route(
				/\/api\/appview\/v1\/profiles\/[^/]+\/(followers|following)(?:\?|$)/,
				(route) => route.fulfill({ json: { items: [maker], totalCount: 1 } })
			);
			const metrics = async () => {
				await expect(
					page.locator('.relationship-list .identity')
				).toBeVisible();
				return page.locator('.settings-workspace').evaluate((workspace) => {
					const heading = workspace.querySelector('h1')!;
					const back = workspace.querySelector('.settings-back')!;
					const row = workspace.querySelector('.relationship-list .row')!;
					const identity = row.querySelector('.identity')!;
					const avatar = identity.querySelector('.avatar')!;
					const style = (element: Element) => {
						const css = getComputedStyle(element);
						return {
							display: css.display,
							alignItems: css.alignItems,
							gap: css.gap,
							padding: css.padding,
							fontSize: css.fontSize,
							fontWeight: css.fontWeight
						};
					};
					const headingBox = heading.getBoundingClientRect();
					const backBox = back.getBoundingClientRect();
					return {
						heading: style(heading),
						row: style(row),
						identity: style(identity),
						avatarWidth: avatar.getBoundingClientRect().width,
						headingHeight: headingBox.height,
						backOffset: backBox.top - headingBox.top
					};
				});
			};
			// Load the profile route first so Settings CSS cannot mask missing dependencies.
			await page.goto(`/profiles/did%3Aplc%3Abob/${kind}`);
			const profile = await metrics();
			expect(profile.identity.display).toBe('flex');
			expect(profile.backOffset).toBe(0);
			await page.goto(`/settings/${kind}`);
			expect(await metrics()).toEqual(profile);
		});
	}
}

for (const width of [1280, 390]) {
	for (const kind of ['followers', 'following']) {
		test(`profile and Settings ${kind} share loading states at ${width}px`, async ({
			page
		}) => {
			await page.setViewportSize({ width, height: 900 });
			const states: unknown[] = [];
			for (const path of [
				`/profiles/did%3Aplc%3Abob/${kind}`,
				`/settings/${kind}`
			]) {
				let release!: () => void;
				const pending = new Promise<void>((resolve) => (release = resolve));
				const endpoint =
					/\/api\/appview\/v1\/profiles\/[^/]+\/(followers|following)(?:\?|$)/;
				await page.route(
					endpoint,
					async (route) => {
						await pending;
						await route.fulfill({
							json: { items: [maker], totalCount: 2, cursor: 'next' }
						});
					},
					{ times: 1 }
				);
				await page.goto(path);
				try {
					await page.addStyleTag({
						content: '.settings-workspace { --profile-accent: #1535d6; }'
					});
					await expect(
						page.getByRole('status', { name: 'Loading content' })
					).toBeVisible();
					await expect(page.locator('.skeletons .skeleton')).toHaveCount(6);
					await expect(
						page.locator('.skeletons .skeleton-avatar').first()
					).toHaveCSS('border-top-width', '0px');
					await expect(page.locator('.relationship-list')).toHaveCount(0);
					states.push(
						await page.locator('.settings-workspace').evaluate((workspace) => {
							const heading = workspace.querySelector('h1')!;
							const loading = workspace.querySelector('.skeletons')!;
							const css = getComputedStyle(loading);
							const shimmer = getComputedStyle(loading.querySelector('.bone')!);
							return {
								fontSize: css.fontSize,
								lineHeight: css.lineHeight,
								margin: css.margin,
								color: css.color,
								animation: shimmer.animationName,
								duration: shimmer.animationDuration,
								offset:
									loading.getBoundingClientRect().top -
									heading.getBoundingClientRect().bottom
							};
						})
					);
				} finally {
					release();
				}
				await expect(
					page.locator('.relationship-list .identity')
				).toBeVisible();
				await expect(
					page.locator('.settings-workspace .skeletons')
				).toHaveCount(0);
				let finishMore!: () => void;
				const morePending = new Promise<void>(
					(resolve) => (finishMore = resolve)
				);
				await page.route(
					endpoint,
					async (route) => {
						await morePending;
						await route.fulfill({
							json: {
								items: [
									{ ...maker, did: 'did:plc:carol', handle: 'carol.test' }
								],
								totalCount: 2
							}
						});
					},
					{ times: 1 }
				);
				const loadMore = page.getByRole('button', {
					name: 'Load more',
					exact: true
				});
				await expect(loadMore).toHaveCSS('border-top-width', '0px');
				await expect(loadMore).toHaveCSS(
					'background-color',
					'rgba(0, 0, 0, 0)'
				);
				const centered = await loadMore.evaluate((button) => {
					const buttonBox = button.getBoundingClientRect();
					const listBox = button
						.closest('.relationship-list')!
						.getBoundingClientRect();
					return Math.abs(
						buttonBox.x + buttonBox.width / 2 - listBox.x - listBox.width / 2
					);
				});
				expect(centered).toBeLessThan(1);
				await loadMore.click();
				try {
					await expect(
						page.getByRole('button', { name: 'Load more', exact: true })
					).toHaveCount(0);
					await expect(
						page.getByRole('status', { name: 'Loading more accounts' })
					).toBeVisible();
					const spinner = page
						.getByRole('status', { name: 'Loading more accounts' })
						.locator('svg');
					await expect(spinner).toHaveAttribute('width', '36');
					await expect(spinner).not.toHaveCSS('animation-name', 'none');
					await page.emulateMedia({ reducedMotion: 'reduce' });
					await expect(spinner).toHaveCSS('animation-name', 'none');
					await page.emulateMedia({ reducedMotion: 'no-preference' });
					await expect(
						page.locator('.relationship-list .identity')
					).toHaveCount(1);
				} finally {
					finishMore();
				}
				await expect(page.locator('.relationship-list .identity')).toHaveCount(
					2
				);
			}
			expect(states[1]).toEqual(states[0]);
			expect(states[0]).toMatchObject({
				animation: expect.stringContaining('shimmer')
			});
		});
	}
}
