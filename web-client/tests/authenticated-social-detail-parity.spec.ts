import { test, expect, type Page } from '@playwright/test';

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

test('following feed retains distinct original and repost entries for the same post', async ({
	page
}) => {
	const value = post('repeated', {
		text: 'One post, three timeline activities.'
	});
	const carol = {
		did: 'did:plc:carol',
		handle: 'carol.test',
		displayName: 'Carol Maker'
	};
	const dana = {
		did: 'did:plc:dana',
		handle: 'dana.test',
		displayName: 'Dana Maker'
	};
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			json: {
				items: [
					{
						itemKey: 'repost:carol',
						post: value,
						reason: { type: 'repost', by: carol }
					},
					{ itemKey: 'post:original', post: value },
					{
						itemKey: 'repost:dana',
						post: value,
						reason: { type: 'repost', by: dana }
					}
				]
			}
		})
	);
	await page.goto('/feed');
	await expect(page.getByText(value.text, { exact: true })).toHaveCount(3);
	await expect(
		page.getByRole('link', { name: 'Reposted by Carol Maker' })
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Reposted by Dana Maker' })
	).toBeVisible();
	await expect(page.getByText('We couldn’t load this page.')).toHaveCount(0);
});

test('project feed summaries retain pattern facets, credits, size and compact edit metadata', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await timeline(
		page,
		post('project', {
			editedAt: '2026-11-01T13:00:00Z',
			project: {
				common: {
					title: 'Weekend cardigan',
					craftType: 'social.craftsky.feed.defs#knitting',
					status: 'social.craftsky.feed.defs#finished',
					pattern: {
						name: '#cardigan',
						nameFacets: [
							{
								index: { byteStart: 0, byteEnd: 9 },
								features: [
									{ $type: 'app.bsky.richtext.facet#tag', tag: 'cardigan' }
								]
							}
						],
						designer: 'Martha',
						publisher: 'Studio',
						selfDrafted: true
					}
				},
				details: { finishedSize: 'Medium' }
			}
		})
	);
	await page.goto('/feed');
	const card = page.locator('.post-card').first();
	const metadata = card.locator('.project-preview-metadata');
	await expect(metadata).toContainText('Pattern');
	await expect(metadata).toContainText('by Martha, Studio');
	await expect(metadata).toContainText('Studio · Self-drafted');
	await expect(metadata).toContainText('Finished size');
	await expect(metadata).toContainText('Medium');
	await expect(
		metadata.getByRole('link', { name: '#cardigan' })
	).toHaveAttribute('href', /cardigan/);
	await expect(card.locator('.author .edited-indicator')).toBeVisible();
	await expect(card.locator('.author .date')).toHaveAttribute(
		'title',
		/Edited/
	);
	await expect(card.locator('p.muted')).toHaveCount(0);
	await expect(page.locator('.post-actions').first()).toHaveCSS(
		'border-top-style',
		'solid'
	);
	await page.screenshot({
		path: test.info().outputPath('project-feed-summary.png')
	});
});

test('empty pattern metadata stays absent and unrecognised status remains outlined', async ({
	page
}) => {
	await timeline(
		page,
		post('project', {
			project: {
				common: {
					title: 'A project',
					craftType: 'sewing',
					status: 'planned',
					pattern: { name: ' ', designer: ' ' }
				},
				details: { sizeMade: ' ' }
			}
		})
	);
	await page.goto('/feed');
	await expect(page.getByRole('heading', { name: 'A project' })).toBeVisible();
	await expect(page.locator('.project-preview-metadata')).toHaveCount(0);
	await expect(page.locator('.status-badge')).toHaveCSS(
		'background-color',
		'rgba(0, 0, 0, 0)'
	);
});

test('quoted projects use native compact identity, text before image and four-line preview', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	const quoted = post('dinosaur', {
		project: {
			common: {
				title: 'It’s A Dinosaur',
				craftType: 'social.craftsky.feed.defs#crochet'
			}
		},
		text: 'This is a project still in the works using mosaic crochet. '.repeat(
			12
		),
		editedAt: '2026-11-01T13:00:00Z',
		images: [
			{
				cid: 'image-one',
				alt: 'A blue dinosaur blanket',
				thumb: 'https://fixture.craftsky.test/dinosaur.svg',
				fullsize: 'https://fixture.craftsky.test/dinosaur.svg',
				aspectRatio: { width: 800, height: 400 }
			}
		]
	});
	await timeline(
		page,
		post('quote', {
			text: '🦖',
			quote: { uri: quoted.uri, cid: quoted.cid },
			quoteView: { state: 'visible', revealable: false, post: quoted }
		})
	);
	await page.route('https://fixture.craftsky.test/dinosaur.svg', (route) =>
		route.fulfill({
			contentType: 'image/svg+xml',
			body: '<svg xmlns="http://www.w3.org/2000/svg" width="800" height="400"><rect width="800" height="400" fill="blue"/></svg>'
		})
	);
	await page.goto('/feed');
	const quote = page.getByRole('complementary', { name: 'Quoted post' });
	await expect(quote.locator('.avatar')).toBeVisible();
	await quote.screenshot({
		path: test.info().outputPath('native-quoted-post.png')
	});
	await expect(quote.getByText('@bob.test', { exact: true })).toBeVisible();
	await expect(
		quote.getByRole('img', { name: 'Crafts: Knitting, Crochet' })
	).toBeVisible();
	await expect(quote.locator('time svg')).toBeVisible();
	await expect(
		quote.getByRole('heading', { name: 'It’s A Dinosaur' })
	).toBeVisible();
	await expect(quote.locator('.summary-text')).toHaveCSS(
		'-webkit-line-clamp',
		'4'
	);
	await expect(quote.locator('a.summary-content')).toHaveAttribute(
		'href',
		'/posts/did%3Aplc%3Abob/dinosaur'
	);
	await expect(
		quote.getByRole('link', { name: 'View profile of Bob Maker', exact: true })
	).toHaveAttribute('href', '/profiles/did%3Aplc%3Abob');
	await expect(quote.locator('a a')).toHaveCount(0);
	expect(
		await quote.evaluate(
			(node) =>
				!!(
					node
						.querySelector('.summary-text')!
						.compareDocumentPosition(node.querySelector('img')!) &
					Node.DOCUMENT_POSITION_FOLLOWING
				)
		)
	).toBe(true);
	await quote
		.getByRole('link', { name: 'View profile of Bob Maker', exact: true })
		.click();
	await expect(page).toHaveURL(/\/profiles\/did%3Aplc%3Abob$/);
});

test('protected quoted content never enters the native summary', async ({
	page
}) => {
	const quoted = post('protected', {
		text: 'PRIVATE_QUOTE_CANARY',
		availability: 'blocked',
		images: [
			{
				fullsize: 'https://fixture.craftsky.test/PRIVATE_IMAGE_CANARY.png',
				alt: 'PRIVATE_ALT_CANARY'
			}
		]
	});
	await timeline(
		page,
		post('quote', {
			quote: { uri: quoted.uri, cid: quoted.cid },
			quoteView: { state: 'blocked', revealable: false, post: quoted }
		})
	);
	await page.goto('/feed');
	await expect(page.getByText('Quoted post blocked.')).toBeVisible();
	await expect(page.locator('.quote-card img')).toHaveCount(0);
	await expect(page.locator('.quote-card .avatar')).toHaveCount(0);
	expect(await page.locator('.quote-card').innerHTML()).not.toContain(
		'PRIVATE_'
	);
});

test('unsaved bookmark confirms before writing and Cancel leaves it unchanged', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	let writes = 0;
	await timeline(page, post());
	await page.route('**/api/appview/v1/posts/**/saves', (route) => {
		writes++;
		return route.fulfill({ json: {} });
	});
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Save post', exact: true });
	await expect(dialog).toBeVisible();
	await dialog.screenshot({
		path: test.info().outputPath('native-save-dialog.png')
	});
	await expect(
		dialog.getByRole('combobox', { name: 'Folder', exact: true })
	).toHaveValue('No folder');
	expect(writes).toBe(0);
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	expect(writes).toBe(0);
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	await dialog.getByRole('button', { name: 'Save post', exact: true }).click();
	await expect.poll(() => writes).toBe(1);
	await expect(
		page.getByRole('button', { name: 'Saved', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
});

test('compact share and overflow menus expose native actions and quote the displayed image CID', async ({
	page
}) => {
	await page.setViewportSize({ width: 375, height: 812 });
	await timeline(
		page,
		post('images', {
			images: [1, 2].map((i) => ({
				cid: 'image-' + i,
				alt: 'Craft image ' + i,
				thumb: 'https://fixture.craftsky.test/' + i + '.svg',
				fullsize: 'https://fixture.craftsky.test/' + i + '.svg',
				aspectRatio: { width: 100, height: 100 }
			}))
		})
	);
	await page.route('https://fixture.craftsky.test/*.svg', (route) =>
		route.fulfill({
			contentType: 'image/svg+xml',
			body: '<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"/>'
		})
	);
	await page.goto('/feed');
	await page
		.getByRole('button', { name: 'Show image 2 of 2', exact: true })
		.click();
	await page.getByLabel('Share post', { exact: true }).click();
	const share = page.locator('.post-share[open]');
	await expect(share.getByRole('switch')).toHaveCSS('width', '52px');
	await expect(share.getByRole('switch')).toHaveCSS('height', '32px');
	await expect(share.locator('.menu-items')).toHaveCSS('position', 'fixed');
	await expect(share.locator('.menu-items')).toHaveCSS(
		'border-top-left-radius',
		'22px'
	);
	await expect(
		share.getByRole('button', { name: 'Repost', exact: true })
	).toHaveCSS('min-height', '56px');
	await expect(
		share.getByRole('button', { name: 'Repost', exact: true })
	).toBeVisible();
	await share
		.getByRole('switch', { name: 'Use current image (2 of 2)', exact: true })
		.check();
	const quote = new URL(
		(await share
			.getByRole('link', { name: 'Quote post', exact: true })
			.getAttribute('href')) || '',
		'https://app.craftsky.social'
	);
	expect(quote.searchParams.get('imageCid')).toBe('image-2');
	await page.screenshot({
		path: test.info().outputPath('native-post-share.png')
	});
	await expect(page.locator('.image-counter')).toHaveText('2/2');
	await share
		.getByRole('button', { name: 'Close post menu' })
		.click({ position: { x: 2, y: 2 } });
	await page.getByLabel('More post actions', { exact: true }).click();
	const menu = page.locator('.post-overflow[open]');
	await page.setViewportSize({ width: 800, height: 1000 });
	await expect(menu.locator('.menu-items')).toHaveCSS('width', '800px');
	await expect(page.locator('html')).toHaveCSS('overflow', 'hidden');
	await menu.getByRole('button', { name: 'Copy link', exact: true }).focus();
	await page.keyboard.press('Shift+Tab');
	await expect(
		menu.getByRole('button', { name: 'Report post', exact: true })
	).toBeFocused();
	await page.keyboard.press('Tab');
	await expect(
		menu.getByRole('button', { name: 'Copy link', exact: true })
	).toBeFocused();
	await expect(
		menu.getByRole('button', { name: 'Copy link', exact: true })
	).toBeVisible();
	await expect(
		menu.getByRole('button', { name: 'Mute account', exact: true })
	).toBeVisible();
	await expect(
		menu.getByRole('button', { name: 'Block account', exact: true })
	).toBeVisible();
	await expect(
		menu.getByRole('button', { name: 'Report post', exact: true })
	).toBeVisible();
	const chat = new URL(
		(await menu
			.getByRole('link', { name: 'Send via chat' })
			.getAttribute('href')) || '',
		'https://app.craftsky.social'
	);
	expect(chat.searchParams.get('link')).toBe(
		'https://app.craftsky.social/posts/did%3Aplc%3Abob/images'
	);
	await menu
		.getByRole('button', { name: 'Close post menu' })
		.click({ position: { x: 2, y: 2 } });
	await expect(page.locator('html')).not.toHaveCSS('overflow', 'hidden');
	await page.setViewportSize({ width: 375, height: 812 });
	await page.getByRole('link', { name: 'Craft image 2', exact: true }).click();
	await expect(page).toHaveURL(/\/posts\/.*\/images\?imageCid=image-2$/);
	await expect(page.locator('.image-counter').first()).toHaveText('2/2');
	await page.getByRole('link', { name: 'Craft image 2', exact: true }).click();
	await expect(page.locator('.image-dialog[open] img')).toHaveAttribute(
		'src',
		'https://fixture.craftsky.test/2.svg'
	);
	await page.keyboard.press('ArrowLeft');
	await expect(page.locator('.image-dialog[open] img')).toHaveAttribute(
		'src',
		'https://fixture.craftsky.test/1.svg'
	);
	await page.keyboard.press('Escape');
	await page.getByLabel('Share post', { exact: true }).click();
	await share
		.getByRole('switch', { name: 'Use current image (2 of 2)', exact: true })
		.check();
	await expect(
		share.getByRole('switch', {
			name: 'Use current image (2 of 2)',
			exact: true
		})
	).toBeChecked();
	expect(
		new URL(
			(await share
				.getByRole('link', { name: 'Quote post', exact: true })
				.getAttribute('href')) || '',
			'https://app.craftsky.social'
		).searchParams.get('imageCid')
	).toBe('image-2');
});

test('report category/reason stages can change and cancel without submitting private details', async ({
	page
}) => {
	let reports = 0;
	await timeline(page, post());
	await page.route('**/api/appview/v1/posts/**/reports', (route) => {
		reports++;
		return route.fulfill({ json: { status: 'ok' } });
	});
	await page.goto('/feed');
	await page.getByLabel('More post actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Report post', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Report post', exact: true });
	await expect(dialog.getByRole('radio')).toHaveCount(8);
	await dialog
		.getByRole('radio', {
			name: 'Spam, misleading or off-topic content',
			exact: true
		})
		.click();
	await dialog.getByRole('radio', { name: 'Spam', exact: true }).check();
	await dialog
		.getByRole('textbox', { name: 'Details', exact: true })
		.fill('PRIVATE_REPORT_CANARY');
	await page.screenshot({
		path: test.info().outputPath('native-report-reasons.png')
	});
	await dialog.getByRole('button', { name: 'Change category' }).click();
	await expect(dialog.getByRole('radio')).toHaveCount(8);
	await expect(
		dialog.getByRole('button', { name: 'Submit report' })
	).toBeDisabled();
	await dialog.getByRole('button', { name: 'Cancel report' }).click();
	await expect(dialog).not.toBeVisible();
	expect(reports).toBe(0);
	expect(await page.locator('body').innerHTML()).not.toContain(
		'PRIVATE_REPORT_CANARY'
	);
});

test('thread loads initial replies, hides/reopens the cached branch, and targets a reply composer', async ({
	page
}) => {
	const root = post('thread', { likeCount: 6, replyCount: 1 });
	const comment = post('comment', {
		author: { ...maker, did: account.did, handle: account.handle },
		text: 'Parent comment.',
		replyCount: 1,
		reply: {
			root: { uri: root.uri, cid: root.cid },
			parent: { uri: root.uri, cid: root.cid }
		}
	});
	const reply = post('nested', {
		text: 'A nested reply.',
		reply: {
			root: { uri: root.uri, cid: root.cid },
			parent: { uri: comment.uri, cid: comment.cid }
		}
	});
	let replyReads = 0;
	await page.route('**/api/appview/v1/posts/**', (route) => {
		const path = new URL(route.request().url()).pathname;
		if (path.endsWith('/comments'))
			return route.fulfill({
				json: {
					post: root,
					sort: 'oldest',
					comments: {
						items: [
							{
								post: comment,
								placement: 'normal',
								replies: { loaded: false, items: [] }
							}
						]
					}
				}
			});
		if (path.endsWith('/replies')) {
			replyReads++;
			return route.fulfill({
				json: { loaded: true, items: [{ post: reply }] }
			});
		}
		return route.fulfill({ json: root });
	});
	await page.goto('/posts/did%3Aplc%3Abob/thread');
	await expect(
		page.getByRole('link', { name: '6 Likes', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'Show 1 reply', exact: true }).click();
	await expect(
		page.getByText('A nested reply.', { exact: true })
	).toBeVisible();
	await expect.poll(() => replyReads).toBe(1);
	await page.getByRole('button', { name: 'Hide replies', exact: true }).click();
	await expect(page.getByText('A nested reply.', { exact: true })).toHaveCount(
		0
	);
	await page.getByRole('button', { name: 'Show 1 reply', exact: true }).click();
	await expect(
		page.getByText('A nested reply.', { exact: true })
	).toBeVisible();
	expect(replyReads).toBe(1);
	const nested = page.locator('.replies .member-post');
	await expect(
		nested.getByRole('link', { name: 'Reply', exact: true })
	).toHaveAttribute('href', '/compose/reply?did=did%3Aplc%3Abob&rkey=nested');
});

test('mute removes protected author content and keeps the write bound to the selected account', async ({
	page
}) => {
	let muted = false;
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			json: {
				items: [
					{
						itemKey: 'stable',
						post: muted
							? {
									uri: post().uri,
									availability: 'muted',
									relationship: { state: 'muted', revealable: true }
								}
							: post('first', { text: 'PRIVATE_MUTED_CANARY' })
					}
				]
			}
		})
	);
	await page.route('**/api/appview/v1/profiles/**/mutes', (route) => {
		expect(route.request().headers()['x-craftsky-account-id']).toBe(account.id);
		muted = true;
		return route.fulfill({
			json: { did: maker.did, muted: true, blocking: false, blockedBy: false }
		});
	});
	await page.goto('/feed');
	await expect(
		page.getByText('PRIVATE_MUTED_CANARY', { exact: true })
	).toBeVisible();
	await page.getByLabel('More post actions', { exact: true }).click();
	await page.getByRole('button', { name: 'Mute account', exact: true }).click();
	await expect(page.getByText('This content is muted.')).toBeVisible();
	await expect(
		page.getByText('PRIVATE_MUTED_CANARY', { exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('navigation', { name: 'Post actions' })
	).toHaveCount(0);
});

test('unavailable activity suppresses actor identity, both previews, follow and content navigation', async ({
	page
}) => {
	await page.route('**/api/appview/v1/notifications*', (route) =>
		route.request().method() === 'POST'
			? route.fulfill({ status: 204 })
			: route.fulfill({
					json: {
						items: [
							{
								id: 'private-unavailable',
								type: 'quote',
								createdAt: '2026-11-01T12:00:00Z',
								indexedAt: '2026-11-01T12:00:00Z',
								isNew: true,
								actor: {
									...maker,
									available: false,
									displayName: 'PRIVATE_ACTOR_CANARY',
									avatar:
										'https://fixture.craftsky.test/PRIVATE_AVATAR_CANARY.png'
								},
								contentAvailable: false,
								preview: { text: 'PRIVATE_PREVIEW_CANARY', role: 'post' },
								quotedPreview: { text: 'PRIVATE_QUOTED_CANARY', role: 'post' }
							}
						]
					}
				})
	);
	await page.goto('/notifications');
	await expect(
		page.getByText('Activity unavailable', { exact: true })
	).toBeVisible();
	const notification = page.locator('.notification');
	expect(await notification.innerHTML()).not.toContain('PRIVATE_');
	await expect(notification.locator('a[href]')).toHaveCount(0);
	await expect(
		notification.getByRole('button', { name: /Follow/ })
	).toHaveCount(0);
});

const editableProfile = {
	did: account.did,
	handle: account.handle,
	displayName: 'Alice Maker',
	description: '',
	pronouns: '',
	crafts: ['knitting'],
	viewerIsFollowing: false,
	isCraftskyProfile: true,
	links: [
		{
			uri: `at://${account.did}/app.bsky.actor.link/alpha`,
			cid: 'cid-alpha',
			url: 'https://alpha.example/',
			title: 'Alpha',
			createdAt: '2026-10-01T12:00:00Z',
			iconBlob: {
				ref: { $link: 'icon-alpha' },
				mimeType: 'image/png',
				size: 20
			}
		},
		{
			uri: `at://${account.did}/app.bsky.actor.link/beta`,
			cid: 'cid-beta',
			url: 'https://beta.example/',
			title: 'Beta',
			createdAt: '2026-10-02T12:00:00Z'
		}
	]
};
async function profileEditor(page: Page) {
	const writes: { body: Record<string, unknown>; accountId: string }[] = [];
	await page.route('**/api/appview/v1/profiles/me', (route) => {
		if (route.request().method() === 'PUT') {
			writes.push({
				body: route.request().postDataJSON(),
				accountId: route.request().headers()['x-craftsky-account-id']
			});
		}
		return route.fulfill({
			json: { ...editableProfile, editableLinks: editableProfile.links }
		});
	});
	await page.goto('/profile/edit');
	await expect(
		page.getByRole('button', { name: 'Edit link Alpha', exact: true })
	).toBeVisible();
	return writes;
}

test('profile link edit and reorder stay local until the authoritative profile save', async ({
	page
}) => {
	const writes = await profileEditor(page);
	const profileSave = page.getByRole('button', {
		name: 'Save profile',
		exact: true
	});
	await expect(profileSave).toBeDisabled();
	await page
		.getByRole('button', { name: 'Edit link Alpha', exact: true })
		.click();
	const dialog = page.getByRole('dialog', { name: 'Edit link', exact: true });
	await dialog
		.getByRole('textbox', { name: 'Title (optional)', exact: true })
		.fill('Cancelled title');
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Edit link Alpha', exact: true })
	).toBeVisible();
	await expect(profileSave).toBeDisabled();
	expect(writes).toHaveLength(0);
	await page.getByLabel('Actions for Alpha', { exact: true }).click();
	await page
		.locator('.link-actions[open]')
		.getByRole('button', { name: 'Edit link', exact: true })
		.click();
	await dialog
		.getByRole('textbox', { name: 'URL', exact: true })
		.fill('shop.example/');
	await dialog
		.getByRole('textbox', { name: 'Title (optional)', exact: true })
		.fill('My shop');
	await dialog.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await page.getByLabel('Actions for Beta', { exact: true }).click();
	await page
		.locator('.link-actions[open]')
		.getByRole('button', { name: 'Move up', exact: true })
		.click();
	await expect(page.locator('.profile-link .link-card')).toHaveText([
		'Beta',
		'My shop'
	]);
	expect(writes).toHaveLength(0);
	await profileSave.click();
	await expect.poll(() => writes.length).toBe(1);
	expect(writes[0].accountId).toBe(account.id);
	expect(writes[0].body.links).toEqual([
		expect.objectContaining({
			rkey: 'beta',
			cid: 'cid-beta',
			url: 'https://beta.example/',
			createdAt: '2026-10-02T12:00:00Z'
		}),
		expect.objectContaining({
			rkey: 'alpha',
			cid: 'cid-alpha',
			url: 'https://shop.example/',
			title: 'My shop'
		})
	]);
	expect(
		(writes[0].body.links as Record<string, unknown>[])[1]
	).not.toHaveProperty('icon');
	expect(writes[0].body.removedLinks).toEqual([]);
});

test('link add Cancel and duplicate validation never write; removal retains the owned reference until profile save', async ({
	page
}) => {
	const writes = await profileEditor(page);
	await page.getByRole('button', { name: 'Add link', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Add link', exact: true });
	await dialog
		.getByRole('textbox', { name: 'URL', exact: true })
		.fill('https://beta.example/');
	await dialog.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog.getByRole('alert')).toHaveText(
		'This link is already on your profile.'
	);
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(page.locator('.profile-link')).toHaveCount(2);
	await expect(
		page.getByRole('button', { name: 'Save profile', exact: true })
	).toBeDisabled();
	await page.getByLabel('Actions for Alpha', { exact: true }).click();
	await page
		.locator('.link-actions[open]')
		.getByRole('button', { name: 'Remove link', exact: true })
		.click();
	await expect(page.locator('.profile-link .link-card')).toHaveText(['Beta']);
	expect(writes).toHaveLength(0);
	await page.getByRole('button', { name: 'Save profile', exact: true }).click();
	await expect.poll(() => writes.length).toBe(1);
	expect(writes[0].accountId).toBe(account.id);
	expect(writes[0].body.removedLinks).toEqual([
		{ uri: `at://${account.did}/app.bsky.actor.link/alpha`, cid: 'cid-alpha' }
	]);
	expect(writes[0].body.links).toEqual([
		expect.objectContaining({
			rkey: 'beta',
			cid: 'cid-beta',
			url: 'https://beta.example/'
		})
	]);
});

test('native profile editor app bar saves ordered flat craft rows and their default without duplicate controls', async ({
	page
}) => {
	const writes = await profileEditor(page);
	const heading = page.locator('.workspace-header');
	await expect(
		heading.getByRole('link', { name: 'Close editor', exact: true })
	).toBeVisible();
	const save = heading.getByRole('button', {
		name: 'Save profile',
		exact: true
	});
	await expect(save).toHaveText('Save');
	await expect(
		page.getByRole('button', { name: 'Save profile', exact: true })
	).toHaveCount(1);
	await expect(page.locator('.profile-form fieldset > legend')).toHaveText([
		'Links',
		'Crafts'
	]);
	const crafts = page.getByRole('group', { name: 'Crafts', exact: true });
	await expect(crafts.locator('.craft-name')).toHaveText([
		'Knitting',
		'Sewing',
		'Crochet',
		'Quilting',
		'Embroidery'
	]);
	await expect(
		crafts.getByRole('checkbox', { name: 'Knitting', exact: true })
	).toBeChecked();
	await expect(
		crafts
			.getByRole('group', { name: 'Knitting', exact: true })
			.getByText('Default', { exact: true })
	).toBeVisible();
	await crafts.getByRole('checkbox', { name: 'Sewing', exact: true }).check();
	await crafts.getByLabel('Reorder Sewing', { exact: true }).click();
	await crafts
		.locator('.craft-actions[open]')
		.getByRole('button', { name: 'Move up', exact: true })
		.click();
	await expect(
		crafts
			.getByRole('group', { name: 'Sewing', exact: true })
			.getByText('Default', { exact: true })
	).toBeVisible();
	await expect(
		crafts.getByRole('checkbox', { name: 'Sewing', exact: true })
	).toBeChecked();
	await expect(
		crafts.getByRole('checkbox', { name: 'Knitting', exact: true })
	).toBeChecked();
	await crafts
		.getByRole('checkbox', { name: 'Knitting', exact: true })
		.press('Alt+ArrowUp');
	await expect(
		crafts
			.getByRole('group', { name: 'Knitting', exact: true })
			.getByText('Default', { exact: true })
	).toBeVisible();
	await crafts
		.getByRole('checkbox', { name: 'Knitting', exact: true })
		.uncheck();
	await expect(
		crafts
			.getByRole('group', { name: 'Sewing', exact: true })
			.getByText('Default', { exact: true })
	).toBeVisible();
	await crafts.getByRole('checkbox', { name: 'Knitting', exact: true }).check();
	expect(writes).toHaveLength(0);
	await save.click();
	await expect.poll(() => writes.length).toBe(1);
	expect(writes[0].accountId).toBe(account.id);
	expect(writes[0].body.crafts).toEqual([
		'social.craftsky.feed.defs#sewing',
		'social.craftsky.feed.defs#knitting'
	]);
});

test('profile editor Close preserves the unsaved draft when discard is declined', async ({
	page
}) => {
	const writes = await profileEditor(page);
	await page
		.getByRole('textbox', { name: 'Display name', exact: true })
		.fill('Unsaved maker name');
	const close = page.getByRole('link', { name: 'Close editor', exact: true });
	await close.click();
	const discard = page.getByRole('dialog', {
		name: 'Discard changes?',
		exact: true
	});
	await expect(discard).toContainText("Your edits won't be saved.");
	await discard
		.getByRole('button', { name: 'Keep editing', exact: true })
		.click();
	await expect(page).toHaveURL(/\/profile\/edit$/);
	await expect(
		page.getByRole('textbox', { name: 'Display name', exact: true })
	).toHaveValue('Unsaved maker name');
	expect(writes).toHaveLength(0);
	await close.click();
	await discard.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(page).toHaveURL(/\/profiles\/did%3Aplc%3Aalice$/);
	expect(writes).toHaveLength(0);
});

test('browser Back can cancel and then discard profile edits without saving', async ({
	page
}) => {
	await page.goto('/feed');
	const writes = await profileEditor(page);
	// Establish an in-app history entry instead of a document navigation.
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await page.getByRole('link', { name: 'Edit profile', exact: true }).click();
	await page
		.getByRole('textbox', { name: 'Display name', exact: true })
		.fill('Unsaved history edit');
	await page.evaluate(() => history.back());
	const dialog = page.getByRole('dialog', {
		name: 'Discard changes?',
		exact: true
	});
	await expect(dialog).toBeVisible();
	await dialog
		.getByRole('button', { name: 'Keep editing', exact: true })
		.click();
	await expect(page).toHaveURL(/\/profile\/edit$/);
	await page.evaluate(() => history.back());
	await dialog.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(page).toHaveURL(/\/profiles\/did%3Aplc%3Aalice$/);
	expect(writes).toHaveLength(0);
});

test('loading skeletons stop shimmering for reduced motion and disappear when content arrives', async ({
	page
}) => {
	let release!: () => void;
	const waiting = new Promise<void>((resolve) => (release = resolve));
	await page.route('**/api/appview/v1/feed/timeline*', async (route) => {
		await waiting;
		await route.fulfill({
			json: { items: [{ itemKey: 'stable', post: post() }] }
		});
	});
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.goto('/feed');
	const skeleton = page.getByRole('status', { name: 'Loading content' });
	await expect(skeleton).toBeVisible();
	await expect(skeleton.locator('.bone').first()).toHaveCSS(
		'animation-name',
		'none'
	);
	await page.emulateMedia({ reducedMotion: 'no-preference' });
	await expect(skeleton.locator('.bone').first()).not.toHaveCSS(
		'animation-name',
		'none'
	);
	release();
	await expect(skeleton).not.toBeVisible();
	await expect(
		page.getByText('Weekend making.', { exact: true })
	).toBeVisible();
});

test('authoritative profile save immediately refreshes shell identity and ignores other-account update events', async ({
	page
}) => {
	await page.setViewportSize({ width: 375, height: 812 });
	let profile = {
		...editableProfile,
		avatar: undefined as string | undefined,
		customisation: { colour: 'cobalt', profileBackground: 'none' }
	};
	let reads = 0;
	const owners: string[] = [];
	await page.route('https://fixture.craftsky.test/saved-avatar.svg', (route) =>
		route.fulfill({
			contentType: 'image/svg+xml',
			body: '<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40"><rect width="40" height="40" fill="red"/></svg>'
		})
	);
	await page.route('**/api/appview/v1/profiles/me', (route) => {
		owners.push(route.request().headers()['x-craftsky-account-id']);
		if (route.request().method() === 'PUT') {
			const body = route.request().postDataJSON();
			profile = {
				...profile,
				displayName: body.displayName,
				avatar: 'https://fixture.craftsky.test/saved-avatar.svg',
				customisation: { colour: 'rose', profileBackground: 'none' }
			};
		} else reads++;
		return route.fulfill({
			json: { ...profile, editableLinks: profile.links }
		});
	});
	await page.goto('/profile/edit');
	await page
		.getByRole('textbox', { name: 'Display name', exact: true })
		.fill('Saved maker identity');
	await page.getByRole('button', { name: 'Save profile', exact: true }).click();
	await expect(page).toHaveURL(/\/profiles\/did%3Aplc%3Aalice$/);
	const bottomProfile = page
		.getByRole('navigation', { name: 'Main tabs', exact: true })
		.getByRole('link', { name: 'Profile', exact: true });
	await expect(bottomProfile.locator('.avatar img')).toHaveAttribute(
		'src',
		'https://fixture.craftsky.test/saved-avatar.svg'
	);
	await expect(page.locator('.app-topbar')).toHaveCSS(
		'--profile-accent',
		'#D61535'
	);
	await page
		.getByRole('button', { name: 'Open navigation', exact: true })
		.click();
	await page
		.getByRole('button', { name: 'Switch accounts, @alice.test', exact: true })
		.click();
	const identity = page.locator(
		'.account-menu [aria-current] .shell-identity strong'
	);
	await expect(identity).toHaveText('Saved maker identity');
	await expect(
		page.locator('.account-menu [aria-current] img')
	).toHaveAttribute('src', 'https://fixture.craftsky.test/saved-avatar.svg');
	const before = reads;
	profile = { ...profile, displayName: 'Fresh authoritative identity' };
	await page.evaluate(
		({ accountId, did }) => {
			window.dispatchEvent(
				new CustomEvent('craftsky-profile-updated', {
					detail: { accountId: 'c6dc7d6a-4bff-45f0-87ee-d42634646546', did }
				})
			);
			window.dispatchEvent(
				new CustomEvent('craftsky-profile-updated', {
					detail: { accountId, did: 'did:plc:bob' }
				})
			);
			window.dispatchEvent(
				new CustomEvent('craftsky-profile-updated', {
					detail: { accountId, did }
				})
			);
		},
		{ accountId: account.id, did: account.did }
	);
	await expect(identity).toHaveText('Fresh authoritative identity');
	expect(reads - before).toBe(1);
	expect(owners.every((owner) => owner === account.id)).toBe(true);
});

test('fullscreen gallery expands alt text, resets zoom across images, handles swipes and restores focus', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	const description =
		'A hand knitted blanket with alternating blue and gold cables. '.repeat(12);
	await timeline(
		page,
		post('gallery', {
			images: [1, 2].map((i) => ({
				cid: 'gallery-' + i,
				alt: i === 1 ? description : 'Second photograph',
				thumb: `https://fixture.craftsky.test/${i}.svg`,
				fullsize: `https://fixture.craftsky.test/${i}.svg`,
				aspectRatio: { width: 400, height: 600 }
			}))
		})
	);
	await page.route('https://fixture.craftsky.test/*.svg', (route) =>
		route.fulfill({
			contentType: 'image/svg+xml',
			body: '<svg xmlns="http://www.w3.org/2000/svg" width="400" height="600"><rect width="400" height="600" fill="#f7d46a"/></svg>'
		})
	);
	await page.goto('/feed');
	const open = page.locator('.gallery a').first();
	await open.click();
	await expect(page).toHaveURL(/\/posts\/.*\/gallery\?imageCid=gallery-1$/);
	await open.click();
	const gallery = page.getByRole('dialog', {
		name: 'Image gallery',
		exact: true
	});
	await expect(gallery).toHaveCSS('background-color', 'rgb(0, 0, 0)');
	expect(
		await gallery.evaluate((el) => [el.clientWidth, el.clientHeight])
	).toEqual([402, 874]);
	await expect(gallery.locator('.count')).toHaveText('1/2');
	await gallery.getByRole('button', { name: 'Show more', exact: true }).click();
	await expect(gallery.locator('.description p')).toHaveCSS(
		'max-height',
		'349.6px'
	);
	await expect(
		gallery.getByRole('button', { name: 'Show less', exact: true })
	).toHaveAttribute('aria-expanded', 'true');
	await gallery.screenshot({
		path: test.info().outputPath('native-fullscreen-gallery.png')
	});
	await page.keyboard.press('+');
	await expect(gallery.locator('img')).toHaveCSS(
		'transform',
		'matrix(1.5, 0, 0, 1.5, 0, 0)'
	);
	await page.keyboard.press('ArrowRight');
	await expect(gallery.locator('.count')).toHaveText('2/2');
	await expect(gallery.locator('img')).toHaveCSS(
		'transform',
		'matrix(1, 0, 0, 1, 0, 0)'
	);
	await expect(gallery.getByRole('button', { name: 'Show more' })).toHaveCount(
		0
	);
	// Drag right on the unzoomed image returns to the first page.
	await page.mouse.move(80, 400);
	await page.mouse.down();
	await page.mouse.move(300, 410, { steps: 6 });
	await page.mouse.up();
	await expect(gallery.locator('.count')).toHaveText('1/2');
	await expect(
		gallery.getByRole('button', { name: 'Show more' })
	).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(gallery).toHaveCount(0);
	await expect(open).toBeFocused();
	await expect(page.locator('html')).not.toHaveCSS('overflow', 'hidden');
});

test('external links show their destination in a native confirmation and Cancel restores focus', async ({
	page,
	context: _context
}) => {
	const href = 'https://fixture.craftsky.test/pattern';
	await timeline(
		page,
		post('link', {
			text: 'Pattern link',
			facets: [
				{
					index: { byteStart: 0, byteEnd: 12 },
					features: [{ $type: 'app.bsky.richtext.facet#link', uri: href }]
				}
			]
		})
	);
	await page.route(href, (route) =>
		route.fulfill({
			contentType: 'text/html',
			body: '<title>Pattern destination</title><p>Pattern</p>'
		})
	);
	await page.goto('/feed');
	const link = page.getByRole('link', { name: 'Pattern link', exact: true });
	await link.click();
	const dialog = page.getByRole('dialog', { name: 'Open link?', exact: true });
	await expect(dialog).toContainText('This will open outside CraftSky.');
	await expect(dialog.locator('.destination')).toHaveText(href);
	await expect(dialog).toHaveCSS('border-radius', '14px');
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(link).toBeFocused();
	await expect(page).toHaveURL(/\/feed$/);
	await link.click();
	await dialog.getByRole('link', { name: 'Open link', exact: true }).click();
	await expect(page).toHaveURL(href);
});

test('project details link confirms its destination before leaving CraftSky', async ({
	page
}) => {
	const href = 'https://fixture.craftsky.test/pattern';
	await timeline(
		page,
		post('project-link', {
			project: {
				common: {
					title: 'Weekend cardigan',
					craftType: 'social.craftsky.feed.defs#knitting',
					status: 'social.craftsky.feed.defs#finished',
					pattern: { url: href }
				},
				details: {}
			}
		})
	);
	await page.route(href, (route) =>
		route.fulfill({
			contentType: 'text/html',
			body: '<title>Pattern destination</title>'
		})
	);
	await page.goto('/posts/did:plc:bob/project-link');
	const link = page.locator('.project-detail-metadata a');
	await link.click();
	const dialog = page.getByRole('dialog', { name: 'Open link?', exact: true });
	await expect(dialog.locator('.destination')).toHaveText(href);
	await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(link).toBeFocused();
	await expect(page).toHaveURL(/\/project-link$/);
	await link.click();
	await dialog.getByRole('link', { name: 'Open link', exact: true }).click();
	await expect(page).toHaveURL(href);
});

test('notification post Back returns to the notification list', async ({
	page
}) => {
	const value = post('liked');
	await timeline(page, value);
	await page.route(/\/api\/appview\/v1\/notifications(?:\?.*)?$/, (route) =>
		route.fulfill({
			json: {
				items: [
					{
						id: 'liked',
						type: 'like',
						actor: maker,
						subjectPost: value,
						contentAvailable: true,
						createdAt: value.createdAt,
						indexedAt: value.indexedAt,
						isNew: true
					}
				]
			}
		})
	);
	await page.goto('/notifications');
	await page.locator('.notification-main').click();
	await expect(page).toHaveURL(/\/posts\/.*\/liked$/);
	await page.getByRole('link', { name: 'Back', exact: true }).click();
	await expect(page).toHaveURL(/\/notifications$/);
	await expect(page.locator('.notification-main')).toBeVisible();
});

test('external preview confirms and nested hover leaves the outer post unchanged', async ({
	page
}) => {
	await timeline(
		page,
		post('external', {
			external: {
				uri: 'https://example.com/pattern',
				title: 'External pattern',
				description: 'A pattern preview'
			}
		})
	);
	await page.goto('/feed');
	const card = page.locator('.member-post').first();
	const link = card.locator('.external-card');
	await page.mouse.move(0, 0);
	const background = await card.evaluate(
		(el) => getComputedStyle(el).backgroundColor
	);
	await link.hover();
	await expect(card).toHaveCSS('background-color', background);
	await link.click();
	await expect(
		page.getByRole('dialog', { name: 'Open link?', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(page).toHaveURL(/\/feed$/);
});

test('inline carousel arrows and mouse swipe retain the feed and centre mixed ratios', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await timeline(
		page,
		post('carousel', {
			images: [1, 2].map((i) => ({
				cid: 'image-' + i,
				alt: 'Photo ' + i,
				thumb: `https://fixture.craftsky.test/${i}.svg`,
				fullsize: `https://fixture.craftsky.test/${i}.svg`,
				aspectRatio: { width: 400, height: i === 1 ? 500 : 200 }
			}))
		})
	);
	await page.route('https://fixture.craftsky.test/*.svg', (route) =>
		route.fulfill({
			contentType: 'image/svg+xml',
			body: `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="${route.request().url().endsWith('1.svg') ? 500 : 200}"><rect width="400" height="500" fill="#f7d46a"/></svg>`
		})
	);
	await page.goto('/feed');
	const carousel = page.locator('.post-images').first();
	await carousel
		.getByRole('button', { name: 'Next image', exact: true })
		.click();
	await expect(
		carousel.getByRole('button', { name: 'Show image 2 of 2', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	const bounds = (await carousel.locator('.gallery').boundingBox())!;
	await page.mouse.move(bounds.x + 50, bounds.y + 90);
	await page.mouse.down();
	await page.mouse.move(bounds.x + bounds.width - 20, bounds.y + 90, {
		steps: 12
	});
	await page.mouse.up();
	await expect(
		carousel.getByRole('button', { name: 'Show image 1 of 2', exact: true })
	).toHaveAttribute('aria-pressed', 'true');
	await expect(page).toHaveURL(/\/feed$/);
	await expect(carousel.locator('.gallery > a').first()).toHaveCSS(
		'align-items',
		'center'
	);
});

test('project card surface opens native metadata and material chips', async ({
	page
}) => {
	await timeline(
		page,
		post('details', {
			project: {
				common: {
					title: 'Linen trousers',
					craftType: 'social.craftsky.feed.defs#sewing',
					status: 'social.craftsky.feed.defs#finished',
					duration: '5 days',
					pattern: {
						name: '#trousers',
						designer: 'Maker',
						url: 'https://example.com/pattern',
						difficulty: 'beginner'
					},
					materials: [{ text: 'Blue linen' }],
					colors: ['blue']
				},
				details: {
					projectType: 'social.craftsky.project.defs#garment',
					projectSubtype: 'social.craftsky.project.sewing#trousers',
					sizeMade: 'M',
					fitNotes: 'Relaxed fit'
				}
			}
		})
	);
	await page.goto('/feed');
	await page.locator('.status-badge').click();
	await expect(page).toHaveURL(/\/posts\/.*\/details$/);
	await expect(
		page.getByRole('heading', { name: 'Linen trousers', exact: true })
	).toBeVisible();
	const metadata = page.locator('.project-detail-metadata');
	await expect(metadata.locator('dt')).toHaveText([
		'Pattern',
		'Duration',
		'Difficulty',
		'Link',
		'Project type',
		'Size made',
		'Fit notes'
	]);
	await expect(metadata).toContainText('Garment >');
	await expect(page.locator('.detail-chips').first()).toHaveText('Blue linen');
	await expect(
		page.getByText('The project details', { exact: true })
	).toHaveCount(0);
});

test('returning authors get the compact post context menu and matching search focus', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await page.addInitScript(() =>
		localStorage.setItem('craftsky.post-type.seen.did:plc:alice', 'true')
	);
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Create post', exact: true }).click();
	const menu = page.locator('.post-type-context[open] .menu-items');
	await expect(menu).toBeVisible();
	await expect(
		page.getByRole('dialog', { name: 'What would you like to share?' })
	).not.toBeVisible();
	await menu.getByRole('link', { name: /Regular post/ }).click();
	await expect(page).toHaveURL(/\/compose$/);
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await page.goto('/search');
	const search = page.locator('#search');
	await search.click();
	await expect(search).toHaveCSS('border-top-width', '0px');
	await expect(search).toHaveCSS('box-shadow', 'none');
	await expect(page.locator('.search-field')).toHaveCSS(
		'box-shadow',
		'rgb(22, 18, 16) 3px 3px 0px 0px'
	);
});

test('long lists offer scroll to top and account menus adapt to their trigger', async ({
	page
}) => {
	await page.route('**/api/appview/v1/feed/timeline*', (route) =>
		route.fulfill({
			json: {
				items: Array.from({ length: 12 }, (_, i) => ({
					itemKey: String(i),
					post: post('item' + i)
				}))
			}
		})
	);
	await page.goto('/feed');
	await expect(page.locator('.member-post')).toHaveCount(12);
	const top = page.getByRole('button', { name: 'Scroll to top', exact: true });
	await expect(top).toBeHidden();
	await page.evaluate(() => window.scrollTo(0, 600));
	await expect(top).toBeVisible();
	await top.click();
	await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0);
	await page.getByRole('button', { name: /^Switch accounts,/ }).click();
	await expect(page.locator('.account-menu.anchored')).toBeVisible();
	await page.keyboard.press('Escape');
	await page.setViewportSize({ width: 402, height: 874 });
	await page
		.getByRole('button', { name: 'Open navigation', exact: true })
		.click();
	await page.getByRole('button', { name: /^Switch accounts,/ }).click();
	const sheet = page.getByRole('dialog', { name: 'Accounts', exact: true });
	await expect(sheet).toBeVisible();
	expect(
		await sheet.evaluate((e) => Math.round(e.getBoundingClientRect().bottom))
	).toBe(874);
	await expect(sheet.locator('.person-row[aria-current=true]')).toBeDisabled();
});

test('post hover only advertises navigation and closing a comment does not leave a composer history entry', async ({
	page
}) => {
	await timeline(page, post());
	await page.goto('/feed');
	const card = page.locator('.member-post').first();
	await expect(card).toBeVisible();
	await expect(card).toHaveCSS('cursor', 'pointer');
	const idle = await card.evaluate((n) => getComputedStyle(n).backgroundColor);
	await card.hover({ position: { x: 4, y: 4 } });
	await expect
		.poll(() => card.evaluate((n) => getComputedStyle(n).backgroundColor))
		.not.toBe(idle);
	await page.getByText('Weekend making.', { exact: true }).click();
	await expect(page).toHaveURL(/\/posts\//);
	const detail = page.locator('.member-post').first();
	await page.mouse.move(0, 0);
	const detailIdle = await detail.evaluate(
		(n) => getComputedStyle(n).backgroundColor
	);
	await detail.hover({ position: { x: 4, y: 4 } });
	await expect(detail).toHaveCSS('background-color', detailIdle);
	await expect(detail).not.toHaveCSS('cursor', 'pointer');
	await page.locator('.comment-compose').click();
	await expect(page).toHaveURL(/\/compose\/reply/);
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await expect(page).toHaveURL(/\/posts\//);
	await page.getByRole('link', { name: 'Back', exact: true }).click();
	await expect(page).toHaveURL(/\/feed$/);
});

test('likes rows keep avatar and identity centred, contiguous and highlighted on hover', async ({
	page
}) => {
	await timeline(page, post('first', { likeCount: 2 }));
	await page.route('**/api/appview/v1/posts/**/likes*', (route) =>
		route.fulfill({
			json: {
				items: [
					maker,
					{
						...maker,
						did: 'did:plc:carol',
						handle: 'carol.test',
						displayName: 'Carol Maker'
					}
				]
			}
		})
	);
	await page.goto('/posts/did%3Aplc%3Abob/first/likes');
	const rows = page.locator('.thread-person');
	await expect(rows).toHaveCount(2);
	const metrics = await rows.evaluateAll((nodes) =>
		nodes.map((n) => {
			const row = n.getBoundingClientRect(),
				avatar = n.querySelector('.avatar')!.getBoundingClientRect();
			return {
				top: row.top,
				bottom: row.bottom,
				height: row.height,
				center: row.y + row.height / 2,
				avatarCenter: avatar.y + avatar.height / 2
			};
		})
	);
	expect(metrics[0].height).toBe(56);
	expect(metrics[1].top).toBe(metrics[0].bottom);
	expect(metrics[0].center).toBe(metrics[0].avatarCenter);
	const idle = await rows
		.first()
		.evaluate((n) => getComputedStyle(n).backgroundColor);
	await rows.first().hover();
	await expect
		.poll(() =>
			rows.first().evaluate((n) => getComputedStyle(n).backgroundColor)
		)
		.not.toBe(idle);
});

test('profile drag lifts the complete row, supports cancel and commits only on profile save', async ({
	page
}) => {
	const writes = await profileEditor(page);
	const links = page.locator('.profile-link:not(.reorder-ghost)');
	const first = links.first(),
		second = links.nth(1);
	await first.evaluate((node) =>
		node.scrollIntoView({ block: 'center', behavior: 'instant' })
	);
	const handle = await first.locator('.drag-handle').boundingBox(),
		target = await second.boundingBox();
	await page.mouse.move(
		handle!.x + handle!.width / 2,
		handle!.y + handle!.height / 2
	);
	await page.mouse.down();
	await page.mouse.move(
		target!.x + target!.width / 2,
		target!.y + target!.height / 2 + 10,
		{ steps: 8 }
	);
	const ghost = page.locator('.reorder-ghost');
	await expect(ghost).toContainText('Alpha');
	await expect(ghost.locator('.link-card')).toBeVisible();
	await page.screenshot({
		path: test.info().outputPath('profile-row-lift.png')
	});
	await page.keyboard.press('Escape');
	await page.mouse.up();
	await expect(ghost).toHaveCount(0);
	await expect(links.locator('.link-card')).toHaveText(['Alpha', 'Beta']);
	await first.evaluate((node) =>
		node.scrollIntoView({ block: 'center', behavior: 'instant' })
	);
	const h = await first.locator('.drag-handle').boundingBox(),
		t = await second.boundingBox();
	await page.mouse.move(h!.x + h!.width / 2, h!.y + h!.height / 2);
	await page.mouse.down();
	await page.mouse.move(t!.x + t!.width / 2, t!.y + t!.height / 2 + 10, {
		steps: 8
	});
	await page.mouse.up();
	await expect(links.locator('.link-card')).toHaveText(['Beta', 'Alpha']);
	const crafts = page.getByRole('group', { name: 'Crafts', exact: true });
	await crafts.getByRole('checkbox', { name: 'Sewing', exact: true }).check();
	const sewing = crafts.getByRole('group', { name: 'Sewing', exact: true });
	await sewing.scrollIntoViewIfNeeded();
	const sh = await sewing.locator('.drag-handle').boundingBox();
	const knitting = await crafts
		.getByRole('group', { name: 'Knitting', exact: true })
		.boundingBox();
	await page.mouse.move(sh!.x + sh!.width / 2, sh!.y + sh!.height / 2);
	await page.mouse.down();
	await page.mouse.move(
		sh!.x + sh!.width / 2,
		knitting!.y + knitting!.height / 2 - 10,
		{ steps: 8 }
	);
	await expect(page.locator('.reorder-ghost')).toContainText('Sewing');
	await page.mouse.up();
	await expect(crafts.locator('.craft-name').first()).toHaveText('Sewing');
	expect(writes).toHaveLength(0);
	await page.getByRole('button', { name: 'Save profile', exact: true }).click();
	await expect.poll(() => writes.length).toBe(1);
	expect(
		(writes[0].body.links as { rkey: string }[]).map((x) => x.rkey)
	).toEqual(['beta', 'alpha']);
});

test('desktop sponsored menu has no overflow and toast stays inside the content column', async ({
	page
}) => {
	await page.goto('/compose/project');
	const trigger = page.getByRole('button', {
		name: 'Sponsored options',
		exact: true
	});
	await trigger.click();
	const dialog = page.getByRole('dialog', { name: 'Sponsored', exact: true });
	await expect(dialog).toBeVisible();
	const overflow = await dialog.evaluate((n) => ({
		x: n.scrollWidth - n.clientWidth,
		y: n.scrollHeight - n.clientHeight
	}));
	expect(overflow).toEqual({ x: 0, y: 0 });
	await page.screenshot({
		path: test.info().outputPath('sponsored-desktop-menu.png')
	});
	await page.keyboard.press('Escape');
	await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
	await page.goto('/feed');
	await page.getByLabel('More post actions', { exact: true }).first().click();
	await page.getByRole('button', { name: 'Copy link', exact: true }).click();
	const toast = page.locator('.snackbar');
	await expect(toast).toBeVisible();
	const main = await page.locator('main').boundingBox(),
		bounds = await toast.boundingBox();
	expect(bounds!.x).toBeGreaterThanOrEqual(main!.x);
	expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(main!.x + main!.width);
});

test('draft app bars share the standard app header and profile focus uses ink rather than cobalt', async ({
	page
}) => {
	await page.goto('/feed');
	const header = page.locator('.workspace-header,.workspace-heading').first();
	const standard = await header.evaluate((n) => ({
		height: n.getBoundingClientRect().height,
		border: getComputedStyle(n).borderBottom,
		background: getComputedStyle(n).backgroundColor
	}));
	for (const path of ['/drafts', '/scheduled']) {
		await page.goto(path);
		await expect(page.locator('.draft-appbar')).toBeVisible();
		expect(
			await page.locator('.draft-appbar').evaluate((n) => ({
				height: n.getBoundingClientRect().height,
				border: getComputedStyle(n).borderBottom,
				background: getComputedStyle(n).backgroundColor
			}))
		).toEqual(standard);
	}
	await profileEditor(page);
	const input = page.getByRole('textbox', {
		name: 'Display name',
		exact: true
	});
	await input.focus();
	const ink = await page.evaluate(() =>
		getComputedStyle(document.documentElement).getPropertyValue('--ink').trim()
	);
	await expect
		.poll(() => input.evaluate((n) => getComputedStyle(n).outlineColor))
		.toBe(
			await page.evaluate((value) => {
				const n = document.createElement('span');
				n.style.color = value;
				document.body.append(n);
				const c = getComputedStyle(n).color;
				n.remove();
				return c;
			}, ink)
		);
	await page.screenshot({
		path: test.info().outputPath('profile-field-focus.png')
	});
});

for (const width of [390, 1280]) {
	test(`feed comment action opens the loaded comments section at ${width}px`, async ({
		page
	}) => {
		await page.setViewportSize({ width, height: 844 });
		await timeline(
			page,
			post('first', {
				text: 'A long story about making things.\n\n'.repeat(100)
			})
		);
		let release!: () => void;
		const pending = new Promise<void>((resolve) => {
			release = resolve;
		});
		await page.route('**/api/appview/v1/posts/**/comments*', async (route) => {
			await pending;
			await route.fulfill({ json: { comments: { items: [], cursor: null } } });
		});
		await page.goto('/feed');
		const comment = page
			.getByRole('link', { name: 'Comments', exact: true })
			.first();
		await expect(comment).toHaveAttribute('href', /\?show-comments=true$/);
		try {
			await comment.click();
			await expect(page).toHaveURL(/show-comments=true/);
			await expect(page.locator('.skeleton').first()).toBeVisible();
		} finally {
			release();
		}
		const comments = page.getByRole('region', {
			name: 'Comments',
			exact: true
		});
		await expect(comments).toBeVisible();
		await expect
			.poll(async () => (await comments.boundingBox())!.y)
			.toBeGreaterThanOrEqual(56);
		await expect
			.poll(async () => (await comments.boundingBox())!.y)
			.toBeLessThan(160);
		expect(await page.evaluate(() => scrollY)).toBeGreaterThan(1000);
		await expect(
			page
				.locator('.post-actions')
				.getByRole('link', { name: 'Comments', exact: true })
		).toHaveAttribute('href', /\/compose\/reply\?/);
		await expect(
			page.getByRole('link', { name: 'Add a comment', exact: true })
		).toBeVisible();
		// Opening the post body retains the ordinary top-of-post destination.
		await page.getByRole('link', { name: 'Back', exact: true }).click();
		await page.locator('.post-card .prose').first().click();
		await expect(page).not.toHaveURL(/show-comments/);
		await expect(page.locator('.post-detail')).toBeVisible();
		await expect.poll(() => page.evaluate(() => scrollY)).toBe(0);
	});
}

for (const target of ['/feed', '/posts/did%3Aplc%3Abob/first']) {
	test(`comment icon hover matches the other post actions on ${target}`, async ({
		page
	}) => {
		await timeline(page, post());
		await page.goto(target);
		const card = page.locator('.member-post').first();
		const comment = card.getByRole('link', { name: 'Comments', exact: true });
		const like = card.getByRole('button', { name: 'Like', exact: true });
		await expect(comment).toBeVisible();
		const cardIdle = await card.evaluate(
			(n) => getComputedStyle(n).backgroundColor
		);
		await like.hover();
		await like.evaluate(async (node) => {
			await Promise.all(
				node.getAnimations().map((animation) => animation.finished)
			);
		});
		const otherAction = await like.evaluate((n) => {
			const s = getComputedStyle(n);
			return {
				background: s.backgroundColor,
				radius: s.borderRadius,
				color: s.color
			};
		});
		const before = await comment.boundingBox();
		await comment.hover();
		await expect(comment).toHaveCSS('background-color', otherAction.background);
		await expect(comment).toHaveCSS('border-radius', otherAction.radius);
		await expect(comment).toHaveCSS('color', otherAction.color);
		await expect(comment).toHaveCSS('text-decoration-line', 'none');
		await expect(comment).toHaveCSS('box-shadow', 'none');
		await expect(comment).toHaveCSS('transform', 'none');
		await expect(card).toHaveCSS('background-color', cardIdle);
		expect(await comment.boundingBox()).toEqual(before);
	});
}

for (const colorScheme of ['light', 'dark'] as const) {
	test(`full-width Comment button retains its primary background and text on hover in ${colorScheme}`, async ({
		page
	}) => {
		await page.emulateMedia({ colorScheme });
		await timeline(page, post());
		await page.goto('/posts/did%3Aplc%3Abob/first');
		const button = page.getByRole('link', {
			name: 'Add a comment',
			exact: true
		});
		await expect(button).toBeVisible();
		const textColor = await button.evaluate((n) => getComputedStyle(n).color);
		const expectedBackground = await button.evaluate((n) => {
			const sample = document.createElement('span');
			sample.style.background = 'var(--primary-hover)';
			n.append(sample);
			const expected = getComputedStyle(sample).backgroundColor;
			sample.remove();
			return expected;
		});
		await button.hover();
		await expect(button).toHaveCSS('background-color', expectedBackground);
		await expect(button).toHaveCSS('color', textColor);
		await expect(button.locator('svg')).toHaveCSS('color', textColor);
		await expect(button).toHaveCSS('transform', 'none');
		await button.click();
		await expect(page).toHaveURL(/\/compose\/reply/);
	});
}

test('YouTube embeds require consent, retain timestamps, remember permission and handle playback errors', async ({
	page
}) => {
	let youtubeRequests = 0;
	await page.route('https://www.youtube.com/iframe_api', (route) => {
		youtubeRequests++;
		return route.fulfill({
			contentType: 'application/javascript',
			body: `window.YT = { Player: function(frame, options) { window.youtubeFailure = options.events.onError; this.destroy = function() {}; } }; window.onYouTubeIframeAPIReady();`
		});
	});
	await page.route('https://www.youtube-nocookie.com/**', (route) => {
		youtubeRequests++;
		return route.fulfill({
			contentType: 'text/html',
			body: '<html><body>Fixture player</body></html>'
		});
	});
	await page.route(
		'https://fixture.craftsky.test/youtube-thumbnail.svg',
		(route) =>
			route.fulfill({
				contentType: 'image/svg+xml',
				body: '<svg xmlns="http://www.w3.org/2000/svg" width="120" height="240"><rect width="120" height="240" fill="blue"/></svg>'
			})
	);
	await timeline(
		page,
		post('youtube', {
			external: {
				uri: 'https://m.youtube.com/shorts/dQw4w9WgXcQ?t=1m30s',
				title: 'Knitting tutorial',
				description: 'Learn a stitch',
				thumb: { url: 'https://fixture.craftsky.test/youtube-thumbnail.svg' }
			}
		})
	);
	await page.goto('/feed');
	const play = page.getByRole('button', {
		name: 'Play YouTube video: Knitting tutorial',
		exact: true
	});
	await expect(play).toBeVisible();
	for (const width of [390, 1280]) {
		await page.setViewportSize({ width, height: 900 });
		const media = play.locator('.preview-media');
		await expect(media).toHaveCSS('aspect-ratio', '16 / 9');
		// Read both boxes in one layout snapshot: the responsive shell can resize
		// between separate Playwright calls after a viewport change.
		const { bounds, image } = await media.evaluate((element) => {
			const bounds = element.getBoundingClientRect();
			const image = element.querySelector('img')!.getBoundingClientRect();
			return {
				bounds: { width: bounds.width, height: bounds.height },
				image: { width: image.width, height: image.height }
			};
		});
		expect(bounds!.width / bounds!.height).toBeCloseTo(16 / 9, 2);
		expect(image!.width).toBeCloseTo(bounds!.width, 1);
		expect(image!.height).toBeCloseTo(bounds!.height, 1);
		await expect(media.locator('img')).toHaveCSS('object-fit', 'cover');
	}
	await expect(play.locator('.preview-copy strong')).toHaveText(
		'Knitting tutorial'
	);
	await expect(play.locator('.description')).toHaveText('Learn a stitch');
	await expect(play.locator('.host')).toHaveText('m.youtube.com');
	expect(youtubeRequests).toBe(0);
	await play.click();
	await expect(
		page.getByRole('dialog', { name: 'Play video from YouTube?' })
	).toBeVisible();
	expect(youtubeRequests).toBe(0);
	await page.keyboard.press('Escape');
	await expect(play).toBeFocused();
	await play.click();
	await page.getByRole('button', { name: 'Allow once', exact: true }).click();
	const frame = page.getByTitle('YouTube video player: Knitting tutorial');
	await expect(frame).toHaveAttribute(
		'src',
		/youtube-nocookie.com\/embed\/dQw4w9WgXcQ\?.*start=90/
	);
	await expect(frame).toHaveCSS('aspect-ratio', '9 / 16');
	await expect(frame).toHaveAttribute(
		'referrerpolicy',
		'strict-origin-when-cross-origin'
	);
	await expect(page).toHaveURL(/\/feed$/);
	await page.reload();
	await play.click();
	await page
		.getByRole('button', { name: 'Always allow YouTube', exact: true })
		.click();
	await expect(frame).toBeVisible();
	await page.reload();
	await play.click();
	await expect(frame).toBeVisible();
	await expect(
		page.getByRole('dialog', { name: 'Play video from YouTube?' })
	).not.toBeVisible();
	await page.waitForFunction(() => 'youtubeFailure' in window);
	await page.evaluate(() =>
		(window as typeof window & { youtubeFailure: () => void }).youtubeFailure()
	);
	await expect(
		page
			.getByRole('status')
			.filter({ hasText: 'This video can’t be played here.' })
	).toBeVisible();
	await expect(frame).toHaveCount(0);
	await page
		.getByRole('link', { name: 'Open in YouTube', exact: true })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Open link?', exact: true })
	).toBeVisible();
});

test('blocked YouTube API retains an external fallback', async ({ page }) => {
	await page.route('https://www.youtube.com/iframe_api', (route) =>
		route.abort()
	);
	await page.route('https://www.youtube-nocookie.com/**', (route) =>
		route.fulfill({ contentType: 'text/html', body: 'Fixture player' })
	);
	await timeline(
		page,
		post('youtube-blocked', {
			external: {
				uri: 'https://youtu.be/dQw4w9WgXcQ',
				title: 'Landscape tutorial',
				description: ''
			}
		})
	);
	await page.goto('/feed');
	await page
		.getByRole('button', {
			name: 'Play YouTube video: Landscape tutorial',
			exact: true
		})
		.click();
	await page
		.getByRole('dialog')
		.getByText('Playing this video connects to YouTube.', { exact: false })
		.click();
	await expect(page).toHaveURL(/\/feed$/);
	await page.getByRole('button', { name: 'Allow once', exact: true }).click();
	await expect(
		page
			.getByRole('status')
			.filter({ hasText: 'This video can’t be played here.' })
	).toBeVisible();
	await expect(
		page.getByRole('link', { name: 'Open in YouTube', exact: true })
	).toHaveAttribute('href', 'https://youtu.be/dQw4w9WgXcQ');
});

for (const example of [
	{
		kind: 'posts',
		path: '/feed',
		endpoint: /\/api\/appview\/v1\/feed\/timeline(?:\?|$)/,
		avatar: '.skeleton-avatar'
	},
	{
		kind: 'people',
		path: '/profiles/did%3Aplc%3Abob/mutual-followers',
		endpoint: /\/api\/appview\/v1\/profiles\/[^/]+\/mutual-followers(?:\?|$)/,
		avatar: '.skeleton-avatar.person'
	},
	{
		kind: 'profile',
		path: '/profiles/did%3Aplc%3Abob',
		endpoint: /\/api\/appview\/v1\/profiles\/(?:@|%40)?did[^/?]+(?:\?|$)/,
		avatar: '.profile .skeleton-avatar'
	},
	{
		kind: 'activity',
		path: '/notifications',
		endpoint: /\/api\/appview\/v1\/notifications(?:\?|$)/,
		avatar: '.skeleton-avatar'
	},
	{
		kind: 'management',
		path: '/saved',
		endpoint: /\/api\/appview\/v1\/saved-posts(?:\?|$)/,
		avatar: '.skeleton-avatar.thumbnail'
	},
	{
		kind: 'comments',
		path: '/profiles/did%3Aplc%3Abob?tab=comments',
		endpoint: /\/api\/appview\/v1\/profiles\/[^/]+\/comments(?:\?|$)/,
		avatar: '.skeleton-avatar.comment'
	}
]) {
	test(`${example.kind} skeleton avatars do not inherit profile rings`, async ({
		page
	}) => {
		let release!: () => void;
		const pending = new Promise<void>((resolve) => (release = resolve));
		await page.route(example.endpoint, async (route) => {
			const response = await route.fetch();
			await pending;
			await route.fulfill({ response });
		});
		await page.goto(example.path);
		try {
			await page.addStyleTag({
				content: ':root { --profile-accent: #1535d6; }'
			});
			const skeleton = page
				.locator('.skeletons')
				.filter({ has: page.locator(example.avatar) });
			await expect(skeleton).toBeVisible();
			await expect(skeleton.locator('.avatar')).toHaveCount(0);
			const avatar = skeleton.locator(example.avatar).first();
			await expect(avatar).not.toHaveCSS(
				'border-top-color',
				'rgb(21, 53, 214)'
			);
			await expect(avatar).toHaveCSS('box-shadow', 'none');
			await expect(avatar).toHaveCSS('outline-style', 'none');
			if (example.kind !== 'profile')
				await expect(avatar).toHaveCSS('border-top-width', '0px');
		} finally {
			release();
		}
		await expect(
			page.locator('.skeletons').filter({ has: page.locator(example.avatar) })
		).toHaveCount(0);
	});
}
