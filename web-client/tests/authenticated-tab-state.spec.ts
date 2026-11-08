import { test, expect, type Locator } from '@playwright/test';
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

const author = {
	did: 'did:plc:alice',
	handle: 'alice.test',
	displayName: 'Alice Maker',
	crafts: ['knitting']
};
function posts(prefix: string, count = 12) {
	return Array.from({ length: count }, (_, i) => ({
		uri: `at://${author.did}/social.craftsky.feed.post/${prefix}-${i}`,
		cid: `${prefix}-cid-${i}`,
		rkey: `${prefix}-${i}`,
		author,
		text: `${prefix} ${i}. A little knitting for the weekend.`,
		createdAt: '2026-11-01T12:00:00Z',
		indexedAt: '2026-11-01T12:00:00Z',
		images: [],
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
		viewerHasSaved: false
	}));
}

// Click the sticky strip in place; locator.click's scrollIntoView can move sticky tabs.
async function switchTab(link: Locator) {
	await expect(link).toBeVisible();
	const box = (await link.boundingBox())!;
	await link.page().mouse.click(box.x + box.width / 2, box.y + box.height / 2);
}

function networkPosts(prefix: string, did = 'did:plc:visual-martha') {
	return posts(prefix).map((post) => ({
		itemKey: post.uri.replace(author.did, did),
		post: {
			...post,
			uri: post.uri.replace(author.did, did),
			author: { did, handle: 'maker.test', displayName: 'Network Maker' }
		},
		followedBy: {
			did: 'did:plc:bob',
			handle: 'bob.test',
			displayName: 'Bob Maker'
		}
	}));
}

for (const action of ['follow', 'unfollow'] as const) {
	test(`extended network invalidates history after ${action} from Notifications`, async ({
		page
	}) => {
		let changed = false,
			requests = 0;
		const actor = {
			did: 'did:plc:visual-martha',
			handle: 'martha.makes.test',
			displayName: 'Martha Jones',
			viewerIsFollowing: action === 'unfollow'
		};
		await page.route('**/api/appview/v1/feed/extended-network*', (route) => {
			requests++;
			return route.fulfill({
				json: {
					items: changed
						? []
						: networkPosts(
								'candidate',
								action === 'follow' ? actor.did : 'did:plc:cedar'
							).map((item) => ({
								...item,
								followedBy: action === 'unfollow' ? actor : item.followedBy
							}))
				}
			});
		});
		await page.route(
			(url) => url.pathname === '/api/appview/v1/notifications',
			(route) =>
				route.fulfill({
					json: {
						items: [
							{
								id: 'network-follow',
								type: 'follow',
								actor,
								isNew: true,
								createdAt: '2026-11-01T12:00:00Z'
							}
						]
					}
				})
		);
		await page.route('**/api/appview/v1/profiles/*/follows', (route) => {
			expect(route.request().method()).toBe(
				action === 'follow' ? 'POST' : 'DELETE'
			);
			changed = true;
			return route.fulfill({
				json: { ...actor, viewerIsFollowing: action === 'follow' }
			});
		});
		await page.goto('/feed?source=extended-network');
		await expect(page.locator('.member-post')).toHaveCount(12);
		await page
			.getByRole('navigation', { name: 'App navigation', exact: true })
			.locator('a[href="/notifications"]')
			.click();
		await page
			.getByRole('button', {
				name: action === 'follow' ? 'Follow' : 'Unfollow',
				exact: true
			})
			.click();
		await expect(
			page.getByRole('button', {
				name: action === 'follow' ? 'Unfollow' : 'Follow',
				exact: true
			})
		).toBeVisible();
		const refreshedFeed = page.waitForResponse(
			'**/api/appview/v1/feed/extended-network*'
		);
		await page.goBack();
		await expect(page).toHaveURL('/feed?source=extended-network');
		await refreshedFeed;
		await expect(
			page.getByText(
				'Posts from people followed by accounts you follow will appear here.',
				{ exact: true }
			)
		).toBeVisible();
		await expect(page.locator('.member-post')).toHaveCount(0);
		expect(requests).toBe(2);
	});
}

test('extended network invalidates history after blocking from Messages', async ({
	page
}) => {
	let blocked = false,
		requests = 0;
	await page.route('**/api/appview/v1/feed/extended-network*', (route) => {
		requests++;
		return route.fulfill({
			json: {
				items: blocked
					? []
					: networkPosts('candidate', 'did:plc:bob').map((item) => ({
							...item,
							followedBy: {
								did: 'did:plc:visual-martha',
								handle: 'martha.makes.test',
								displayName: 'Martha Jones'
							}
						}))
			}
		});
	});
	await page.route('**/api/appview/v1/profiles/*/blocks', (route) => {
		blocked = true;
		return route.fulfill({
			json: {
				did: 'did:plc:bob',
				muted: false,
				blocking: true,
				blockedBy: false
			}
		});
	});
	await page.goto('/feed?source=extended-network');
	await expect(page.locator('.member-post')).toHaveCount(12);
	await page
		.getByRole('navigation', { name: 'App navigation', exact: true })
		.locator('a[href="/messages"]')
		.click();
	await page
		.locator('a[href="/messages/conversations/visual-accepted"]')
		.click();
	await page.getByLabel('Conversation actions', { exact: true }).click();
	await page
		.getByRole('button', { name: 'Block account', exact: true })
		.click();
	await page
		.getByRole('dialog', { name: 'Block this account?' })
		.getByRole('button', { name: 'Block account', exact: true })
		.click();
	await expect(
		page.getByText(
			'Account blocked. You can manage blocked accounts in Settings.',
			{ exact: true }
		)
	).toBeVisible();
	await page.goBack();
	await expect(page).toHaveURL('/messages');
	const refreshedFeed = page.waitForResponse(
		'**/api/appview/v1/feed/extended-network*'
	);
	await page.goBack();
	await expect(page).toHaveURL('/feed?source=extended-network');
	await refreshedFeed;
	await expect(
		page.getByText(
			'Posts from people followed by accounts you follow will appear here.',
			{ exact: true }
		)
	).toBeVisible();
	await expect(page.locator('.member-post')).toHaveCount(0);
	expect(requests).toBe(2);
});

test('extended network retains refreshed pages after muting from a card', async ({
	page
}) => {
	let muted = false,
		requests = 0;
	await page.route('**/api/appview/v1/feed/extended-network*', (route) => {
		requests++;
		const more = new URL(route.request().url()).searchParams.has('cursor');
		return route.fulfill({
			json: {
				items: networkPosts(
					(muted ? 'fresh' : 'old') + (more ? '-more' : ''),
					muted ? 'did:plc:cedar' : undefined
				),
				cursor: more ? undefined : 'next'
			}
		});
	});
	await page.route('**/api/appview/v1/profiles/*/mutes', (route) => {
		muted = true;
		return route.fulfill({
			json: {
				did: 'did:plc:visual-martha',
				muted: true,
				blocking: false,
				blockedBy: false
			}
		});
	});
	await page.goto('/feed?source=extended-network');
	await expect(page.locator('.member-post')).toHaveCount(12);
	await page
		.locator('.member-post')
		.first()
		.getByLabel('More post actions', { exact: true })
		.click();
	await page.getByRole('button', { name: 'Mute account', exact: true }).click();
	await expect(page.getByText(/^fresh 0\./)).toBeVisible();
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(24);
	await page
		.getByRole('link', { name: 'Followed by Bob Maker' })
		.first()
		.click();
	await expect(
		page.getByRole('heading', { name: 'Bob Maker', exact: true })
	).toBeVisible();
	await page.goBack();
	await expect(page.locator('.member-post')).toHaveCount(24);
	expect(requests).toBe(3);
});

test('a relationship refresh supersedes an older in-flight network page', async ({
	page
}) => {
	let muted = false;
	let releaseOldPage!: () => void;
	let markStarted!: () => void;
	let markFinished!: () => void;
	const started = new Promise<void>((resolve) => {
		markStarted = resolve;
	});
	const finished = new Promise<void>((resolve) => {
		markFinished = resolve;
	});
	const oldPage = new Promise<void>((resolve) => {
		releaseOldPage = resolve;
	});
	await page.route(
		'**/api/appview/v1/feed/extended-network*',
		async (route) => {
			const more = new URL(route.request().url()).searchParams.has('cursor');
			if (more && !muted) {
				markStarted();
				await oldPage;
				await route.fulfill({ json: { items: networkPosts('obsolete-more') } });
				markFinished();
				return;
			}
			return route.fulfill({
				json: {
					items: networkPosts(
						muted ? 'fresh' : 'old',
						muted ? 'did:plc:cedar' : undefined
					),
					cursor: 'next'
				}
			});
		}
	);
	await page.route('**/api/appview/v1/profiles/*/mutes', (route) => {
		muted = true;
		return route.fulfill({
			json: {
				did: 'did:plc:visual-martha',
				muted: true,
				blocking: false,
				blockedBy: false
			}
		});
	});
	await page.goto('/feed?source=extended-network');
	await expect(page.locator('.member-post')).toHaveCount(12);
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await started;
	await page
		.locator('.member-post')
		.first()
		.getByLabel('More post actions', { exact: true })
		.click();
	await page.getByRole('button', { name: 'Mute account', exact: true }).click();
	await expect(page.getByText(/^fresh 0\./)).toBeVisible();
	const response = page.waitForResponse(
		(reply) =>
			new URL(reply.url()).pathname.endsWith('/feed/extended-network') &&
			new URL(reply.url()).searchParams.has('cursor')
	);
	releaseOldPage();
	await response;
	await finished;
	await page.evaluate(() => new Promise(requestAnimationFrame));
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect(page.getByText(/^obsolete-more /)).toHaveCount(0);
	await expect(page.getByText(/^fresh 0\./)).toBeVisible();
});

test('extended network refreshes retained history after muting its connection', async ({
	page
}) => {
	let muted = false;
	let networkRequests = 0;
	await page.route('**/api/appview/v1/feed/extended-network*', (route) => {
		networkRequests++;
		return route.fulfill({
			json: {
				items: muted
					? []
					: posts('network', 1).map((post) => ({
							itemKey: post.uri,
							post: {
								...post,
								uri: post.uri.replace(author.did, 'did:plc:cedar'),
								author: { ...author, did: 'did:plc:cedar' }
							},
							followedBy: {
								...author,
								did: 'did:plc:bob',
								displayName: 'Bob Maker'
							}
						}))
			}
		});
	});
	await page.route('**/api/appview/v1/profiles/*/mutes', (route) => {
		muted = true;
		return route.fulfill({
			json: {
				did: 'did:plc:bob',
				muted: true,
				blocking: false,
				blockedBy: false
			}
		});
	});
	await page.goto('/feed?source=extended-network');
	await page.getByRole('link', { name: 'Followed by Bob Maker' }).click();
	await expect(
		page.getByLabel('More profile actions', { exact: true })
	).toBeVisible();
	await page.goBack();
	await expect(
		page.getByRole('link', { name: 'Followed by Bob Maker' })
	).toBeVisible();
	expect(networkRequests).toBe(1);
	await page.getByRole('link', { name: 'Followed by Bob Maker' }).click();
	await page.getByLabel('More profile actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Mute account', exact: true }).click();
	await expect(page.getByText('Account muted.', { exact: true })).toBeVisible();
	const refreshedFeed = page.waitForResponse(
		'**/api/appview/v1/feed/extended-network*'
	);
	await page.goBack();
	await expect(page).toHaveURL('/feed?source=extended-network');
	await refreshedFeed;
	await expect(
		page.getByText(
			'Posts from people followed by accounts you follow will appear here.',
			{ exact: true }
		)
	).toBeVisible();
	await expect(page.locator('.member-post')).toHaveCount(0);
	expect(networkRequests).toBe(2);
});

test('extended network deep links retain their own pages, connection attribution and scroll', async ({
	page
}, testInfo) => {
	let networkRequests = 0,
		followingRequests = 0;
	await page.route('**/api/appview/v1/feed/extended-network*', (route) => {
		networkRequests++;
		const more = new URL(route.request().url()).searchParams.has('cursor');
		return route.fulfill({
			json: {
				items: posts(more ? 'network-more' : 'network').map((post, index) => ({
					itemKey: post.uri,
					post: {
						...post,
						uri: post.uri.replace(author.did, 'did:plc:cedar'),
						author: {
							...author,
							did: 'did:plc:cedar',
							handle: 'cedar.test',
							displayName: 'Cedar Maker'
						}
					},
					followedBy:
						index === 0
							? { ...author, did: 'did:plc:bob', displayName: 'Bob Maker' }
							: {
									...author,
									did: 'did:plc:dana',
									displayName: ' ',
									handle: 'dana.test'
								}
				})),
				cursor: more ? undefined : 'network-next'
			}
		});
	});
	await page.route('**/api/appview/v1/feed/timeline*', (route) => {
		followingRequests++;
		return route.fulfill({
			json: {
				items: posts('following').map((post) => ({ itemKey: post.uri, post }))
			}
		});
	});
	await page.goto('/feed?source=extended-network');
	const tabs = page.getByRole('navigation', { name: 'Home feeds' });
	const networkTab = tabs.getByRole('link', {
		name: 'Extended network',
		exact: true
	});
	await expect(networkTab).toHaveAttribute('aria-current', 'page');
	await expect(
		tabs.getByRole('link', { name: 'Following', exact: true })
	).not.toHaveAttribute('aria-current');
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect(page.locator('.network-attribution').first()).toHaveText(
		'Followed by Bob Maker'
	);
	await expect(page.locator('.network-attribution').first()).toHaveAttribute(
		'href',
		'/profiles/did%3Aplc%3Abob'
	);
	await expect(page.locator('.network-attribution').nth(1)).toHaveText(
		'Followed by dana.test'
	);
	await page.screenshot({ path: testInfo.outputPath('extended-network.png') });
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(24);
	await page.evaluate(() => scrollTo({ top: 1150, behavior: 'instant' }));
	await page.evaluate(() => new Promise(requestAnimationFrame));
	const networkY = await page.evaluate(() => scrollY);
	await switchTab(tabs.getByRole('link', { name: 'Following', exact: true }));
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect(page.locator('.network-attribution')).toHaveCount(0);
	await expect(page.getByText(/^following 0\./)).toBeVisible();
	await switchTab(networkTab);
	await expect(page.locator('.member-post')).toHaveCount(24);
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(networkY);
	expect(networkRequests).toBe(2);
	expect(followingRequests).toBe(1);
	await page.evaluate(() =>
		window.dispatchEvent(new Event('craftsky-refresh'))
	);
	await expect(page.locator('.member-post')).toHaveCount(12);
	expect(networkRequests).toBe(3);
});

test('extended network retries initial and subsequent pages without losing loaded posts', async ({
	page
}) => {
	let requests = 0;
	const cursors: (string | null)[] = [];
	await page.route('**/api/appview/v1/feed/extended-network*', (route) => {
		const cursor = new URL(route.request().url()).searchParams.get('cursor');
		cursors.push(cursor);
		if (++requests === 1 || requests === 3)
			return route.fulfill({
				status: 503,
				json: { error: 'unavailable', message: 'Temporary fixture failure' }
			});
		return route.fulfill({
			json: {
				items: posts(cursor ? 'network-next' : 'network').map((post) => ({
					itemKey: post.uri,
					post
				})),
				cursor: cursor ? undefined : 'network-next'
			}
		});
	});
	await page.goto('/feed?source=extended-network');
	await page.getByRole('button', { name: 'Retry', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(12);
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Retry', exact: true })
	).toBeVisible();
	await expect(page.locator('.member-post')).toHaveCount(12);
	await page.getByRole('button', { name: 'Retry', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(24);
	expect(cursors).toEqual([null, null, 'network-next', 'network-next']);
});

test('extended network has its own empty state and never inserts a newly published own post', async ({
	page
}) => {
	await page.goto('/feed?source=extended-network');
	await expect(
		page.getByText(
			'Posts from people followed by accounts you follow will appear here.'
		)
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Connect Instagram' })
	).toHaveCount(0);
	await page.goto('/compose/standard');
	await page
		.getByRole('textbox', { name: 'What would you like to share?' })
		.fill('My own post stays in Following');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL('/feed');
	await expect(
		page.getByText('My own post stays in Following', { exact: true })
	).toBeVisible();
	await page
		.getByRole('navigation', { name: 'Home feeds' })
		.getByRole('link', {
			name: 'Extended network',
			exact: true
		})
		.click();
	await expect(page.locator('.member-post')).toHaveCount(0);
	await expect(
		page.getByText('My own post stays in Following', { exact: true })
	).toHaveCount(0);
});

test('feed tabs retain independent scroll positions and all loaded pages', async ({
	page
}) => {
	let timelineRequests = 0,
		craftRequests = 0;
	await page.route('**/api/appview/v1/feed/timeline*', (route) => {
		timelineRequests++;
		const more = new URL(route.request().url()).searchParams.has('cursor');
		return route.fulfill({
			json: {
				items: posts(more ? 'following-more' : 'following').map((post) => ({
					itemKey: post.uri,
					post
				})),
				cursor: more ? null : 'next-following'
			}
		});
	});
	await page.route('**/api/appview/v1/projects?*', (route) => {
		craftRequests++;
		return route.fulfill({ json: { items: posts('knitting'), cursor: null } });
	});
	await page.goto('/feed');
	await expect(page.locator('.member-post')).toHaveCount(12);
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(24);
	await page.evaluate(() => scrollTo({ top: 1250, behavior: 'instant' }));
	await page.evaluate(() => new Promise(requestAnimationFrame));
	const followingY = await page.evaluate(() => scrollY);
	const tabs = page.getByRole('navigation', { name: 'Home feeds' });
	await switchTab(tabs.getByRole('link', { name: 'Knitting', exact: true }));
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(0);
	await page.evaluate(() => scrollTo({ top: 650, behavior: 'instant' }));
	const knittingY = await page.evaluate(() => scrollY);
	await switchTab(tabs.getByRole('link', { name: 'Following', exact: true }));
	await expect(page.locator('.member-post')).toHaveCount(24);
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(followingY);
	expect(timelineRequests).toBe(2);
	await switchTab(tabs.getByRole('link', { name: 'Knitting', exact: true }));
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(knittingY);
	expect(craftRequests).toBe(1);
	await page.goBack();
	await expect(page.locator('.member-post')).toHaveCount(24);
	await expect.poll(() => page.evaluate(() => scrollY)).toBe(followingY);
	// Returning from another feature fetches fresh content rather than reusing tab data.
	const navigation = page.getByRole('navigation', { name: 'App navigation' });
	await navigation.getByRole('link', { name: 'Search', exact: true }).click();
	await expect(
		page.getByRole('searchbox', { name: 'Search CraftSky', exact: true })
	).toBeVisible();
	await navigation.getByRole('link', { name: 'Home', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(12);
	expect(timelineRequests).toBe(3);
});

test('profile tab changes retain the header, scroll and loaded lists; replies use flat rows', async ({
	page
}) => {
	let profileRequests = 0,
		postRequests = 0;
	page.on('request', (request) => {
		if (
			decodeURIComponent(new URL(request.url()).pathname) ===
			'/api/appview/v1/profiles/@did:plc:alice'
		)
			profileRequests++;
	});
	await page.route('**/api/appview/v1/profiles/*/posts*', (route) => {
		postRequests++;
		return route.fulfill({
			json: { items: posts('profile-post'), cursor: null }
		});
	});
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route('**/api/appview/v1/profiles/*/comments*', async (route) => {
		await pending;
		const parent = {
			uri: 'at://did:plc:bob/social.craftsky.feed.post/parent',
			cid: 'parent-cid'
		};
		await route.fulfill({
			json: {
				items: posts('profile-comment').map((post) => ({
					...post,
					reply: { root: parent, parent }
				})),
				cursor: null
			}
		});
	});
	await page.goto('/profiles/did%3Aplc%3Aalice?tab=posts');
	await expect(page.locator('.member-post')).toHaveCount(12);
	const identity = await page.locator('.profile-identity').elementHandle();
	const initialProfileRequests = profileRequests;
	await page.evaluate(() => scrollTo({ top: 350, behavior: 'instant' }));
	await page.evaluate(() => new Promise(requestAnimationFrame));
	const originalY = await page.evaluate(() => scrollY);
	try {
		await switchTab(
			page
				.getByRole('navigation', { name: 'Profile content' })
				.getByRole('link', { name: 'Comments & replies', exact: true })
		);
		await expect(
			page.locator('.profile-content .skeleton').first()
		).toBeVisible();
		expect(await identity!.evaluate((node) => node.isConnected)).toBe(true);
		expect(await page.evaluate(() => scrollY)).toBe(originalY);
		expect(profileRequests).toBe(initialProfileRequests);
	} finally {
		release();
	}
	await expect(page.locator('.member-post')).toHaveCount(12);
	const comment = page.locator('.member-post').first();
	await expect(comment).toHaveCSS('border-top-width', '0px');
	await expect(comment).toHaveCSS('border-radius', '0px');
	await expect(comment).toHaveCSS('box-shadow', 'none');
	await expect(comment).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
	await expect(comment.locator('.post-actions')).toHaveCSS(
		'border-top-width',
		'0px'
	);
	await expect(
		comment.getByRole('link', { name: 'Reply', exact: true })
	).toBeVisible();
	await switchTab(
		page
			.getByRole('navigation', { name: 'Profile content' })
			.getByRole('link', { name: 'Posts', exact: true })
	);
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect(page.locator('.member-post').first()).not.toHaveCSS(
		'border-top-width',
		'0px'
	);
	expect(postRequests).toBe(1);
	expect(await identity!.evaluate((node) => node.isConnected)).toBe(true);
	expect(await page.evaluate(() => scrollY)).toBe(originalY);
});

test('switching away from a loading profile tab ignores its late response and can retry that tab', async ({
	page
}) => {
	await page.route('**/api/appview/v1/profiles/*/posts*', (route) =>
		route.fulfill({ json: { items: posts('initial') } })
	);
	await page.route('**/api/appview/v1/profiles/*/reposts*', (route) =>
		route.fulfill({ json: { items: posts('repost', 1) } })
	);
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	let requests = 0;
	await page.route('**/api/appview/v1/profiles/*/comments*', async (route) => {
		requests++;
		await pending;
		await route.fulfill({ json: { items: posts('comment', 1) } });
	});
	await page.goto('/profiles/did%3Aplc%3Aalice?tab=posts');
	await expect(page.locator('.member-post')).toHaveCount(12);
	const tabs = page.getByRole('navigation', { name: 'Profile content' });
	try {
		await tabs
			.getByRole('link', { name: 'Comments & replies', exact: true })
			.click();
		await expect.poll(() => requests).toBe(1);
		await tabs.getByRole('link', { name: 'Reposts', exact: true }).click();
		await expect(page.locator('.member-post')).toHaveCount(1);
		await expect(page.locator('.member-post')).toContainText('repost 0.');
	} finally {
		release();
	}
	await tabs
		.getByRole('link', { name: 'Comments & replies', exact: true })
		.click();
	await expect(page.locator('.member-post')).toHaveCount(1);
	await expect(page.locator('.member-post')).toContainText('comment 0.');
	expect(requests).toBe(2);
});

test('search categories retain loaded results and independent positions without rebuilding the search form or sorting', async ({
	page
}) => {
	const requests = new Map<string, number>();
	await page.route('**/api/appview/v1/search/*?*', async (route) => {
		const url = new URL(route.request().url());
		const category = url.pathname.split('/').at(-1)!;
		if (!['posts', 'projects', 'profiles', 'hashtags'].includes(category))
			return route.continue();
		expect(url.searchParams.has('sort')).toBe(false);
		requests.set(category, (requests.get(category) || 0) + 1);
		const more = url.searchParams.has('cursor');
		const items =
			category === 'profiles'
				? Array.from({ length: 30 }, (_, i) => ({
						...author,
						did: `did:plc:maker${i}`,
						handle: `maker${i}.test`
					}))
				: category === 'hashtags'
					? Array.from({ length: 30 }, (_, i) => ({
							tag: `knitting${i}`,
							postCount: i
						}))
					: posts(
							`${url.searchParams.get('q')}-${category}${more ? '-more' : ''}`
						);
		await route.fulfill({
			json: {
				items,
				cursor: category === 'posts' && !more ? 'next-posts' : null
			}
		});
	});
	await page.goto('/search?q=test&sort=popular');
	await expect(page.locator('.member-post')).toHaveCount(12);
	const input = await page
		.getByRole('searchbox', { name: 'Search CraftSky', exact: true })
		.elementHandle();
	await page.getByRole('button', { name: 'Load more', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(24);
	const tabs = page.getByRole('navigation', { name: 'Search categories' });
	const positions = new Map<string, number>();
	for (const [category, position] of [
		['Posts', 1200],
		['Projects', 650],
		['Profiles', 400],
		['Tags', 700]
	] as const) {
		if (category !== 'Posts')
			await switchTab(tabs.getByRole('link', { name: category, exact: true }));
		const results =
			category === 'Profiles'
				? page.locator('.maker-result')
				: category === 'Tags'
					? page.getByRole('link', { name: /^#knitting\d+\b/ })
					: page.locator('.member-post');
		await expect(results).toHaveCount(
			category === 'Posts' ? 24 : category === 'Projects' ? 12 : 30
		);
		await expect(page.locator('.choice-menu')).toHaveCount(0);
		await page.evaluate(
			(y) => scrollTo({ top: y, behavior: 'instant' }),
			position
		);
		await page.evaluate(() => new Promise(requestAnimationFrame));
		positions.set(category, await page.evaluate(() => scrollY));
		if (category === 'Posts') {
			// Normalise a legacy URL that has sort but no explicit tab, then keep scrolling.
			await switchTab(tabs.getByRole('link', { name: 'Posts', exact: true }));
			await expect(page).toHaveURL('/search?q=test&tab=posts');
			await page.evaluate(() => new Promise(requestAnimationFrame));
			await expect
				.poll(() => page.evaluate(() => scrollY))
				.toBe(positions.get(category));
			await page.evaluate(() => scrollTo({ top: 1350, behavior: 'instant' }));
			positions.set(category, await page.evaluate(() => scrollY));
		}
		expect(await input!.evaluate((node) => node.isConnected)).toBe(true);
	}
	for (const category of ['Posts', 'Projects', 'Profiles', 'Tags']) {
		await switchTab(tabs.getByRole('link', { name: category, exact: true }));
		await expect
			.poll(() => page.evaluate(() => scrollY))
			.toBe(positions.get(category));
		expect(await input!.evaluate((node) => node.isConnected)).toBe(true);
	}
	expect(Object.fromEntries(requests)).toEqual({
		posts: 2,
		projects: 1,
		profiles: 1,
		hashtags: 1
	});
	await page.goBack();
	await expect
		.poll(() => page.evaluate(() => scrollY))
		.toBe(positions.get('Profiles'));
	await page
		.getByRole('searchbox', { name: 'Search CraftSky', exact: true })
		.fill('different');
	await page
		.getByRole('searchbox', { name: 'Search CraftSky', exact: true })
		.press('Enter');
	await expect(page.locator('.member-post')).toHaveCount(12);
	await expect(page.locator('.member-post').first()).toContainText(
		'different-posts'
	);
	expect(requests.get('posts')).toBe(3);
	expect(await input!.evaluate((node) => node.isConnected)).toBe(true);
});

test('switching away from a loading search tab ignores its late result', async ({
	page
}) => {
	await page.route('**/api/appview/v1/search/posts?*', (route) =>
		route.fulfill({ json: { items: posts('search-post', 1) } })
	);
	await page.route('**/api/appview/v1/search/hashtags?*', (route) =>
		route.fulfill({ json: { items: [{ tag: 'knitting', postCount: 10 }] } })
	);
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	let requests = 0;
	await page.route('**/api/appview/v1/search/projects?*', async (route) => {
		requests++;
		await pending;
		await route.fulfill({ json: { items: posts('search-project', 1) } });
	});
	await page.goto('/search?q=test');
	await expect(page.locator('.member-post')).toHaveCount(1);
	const tabs = page.getByRole('navigation', { name: 'Search categories' });
	try {
		await tabs.getByRole('link', { name: 'Projects', exact: true }).click();
		await expect.poll(() => requests).toBe(1);
		await tabs.getByRole('link', { name: 'Tags', exact: true }).click();
		await expect(
			page.getByRole('link', { name: '#knitting 10 posts', exact: true })
		).toBeVisible();
	} finally {
		release();
	}
	await page.evaluate(() => new Promise(requestAnimationFrame));
	await expect(page.locator('.member-post')).toHaveCount(0);
	await tabs.getByRole('link', { name: 'Projects', exact: true }).click();
	await expect(page.locator('.member-post')).toHaveCount(1);
	await expect(page.locator('.member-post')).toContainText('search-project');
	expect(requests).toBe(2);
});

for (const kind of ['comment', 'reply']) {
	test(`profile ${kind} menu opens its own likes list`, async ({ page }) => {
		const root = {
			uri: 'at://did:plc:bob/social.craftsky.feed.post/root',
			cid: 'root-cid'
		};
		const responses = posts('profile-response', 2).map((post, index) => ({
			...post,
			likeCount: index === 0 ? 2 : 0,
			reply: {
				root,
				parent:
					kind === 'reply'
						? {
								...root,
								uri: 'at://did:plc:bob/social.craftsky.feed.post/parent'
							}
						: root
			}
		}));
		await page.route('**/api/appview/v1/profiles/*/comments*', (route) =>
			route.fulfill({ json: { items: responses } })
		);
		await page.route('**/api/appview/v1/posts/**/likes*', (route) =>
			route.fulfill({
				json: {
					items: [
						{
							...author,
							did: 'did:plc:carol',
							handle: 'carol.test',
							displayName: 'Carol Maker'
						}
					]
				}
			})
		);
		await page.route('**/api/appview/v1/posts/*/profile-response-0', (route) =>
			route.fulfill({ json: responses[0] })
		);
		await page.goto('/profiles/did%3Aplc%3Aalice?tab=comments');
		const rows = page.locator('.profile-content .member-post');
		await expect(rows).toHaveCount(2);
		await rows
			.nth(1)
			.locator('summary[aria-label="More post actions"]')
			.click();
		await expect(
			rows.nth(1).getByRole('link', { name: 'View likes', exact: true })
		).toHaveCount(0);
		await page.keyboard.press('Escape');
		await rows
			.first()
			.locator('summary[aria-label="More post actions"]')
			.click();
		const likes = rows
			.first()
			.getByRole('link', { name: 'View likes', exact: true });
		await expect(likes).toHaveAttribute(
			'href',
			'/posts/did%3Aplc%3Aalice/profile-response-0/likes'
		);
		await likes.click();
		await expect(page).toHaveURL(
			/\/posts\/did%3Aplc%3Aalice\/profile-response-0\/likes$/
		);
		await expect(page.locator('.thread-person')).toContainText('Carol Maker');
	});
}
