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
	await page.goto('/app/home');
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
		'/app/posts/did%3Aplc%3Abob/dinosaur'
	);
	await expect(
		quote.getByRole('link', { name: 'View profile of Bob Maker', exact: true })
	).toHaveAttribute('href', '/app/profile/did%3Aplc%3Abob');
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
	await expect(page).toHaveURL(/\/app\/profile\/did%3Aplc%3Abob$/);
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
	await page.goto('/app/home');
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
	await page.goto('/app/home');
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Save post', exact: true });
	await expect(dialog).toBeVisible();
	await dialog.screenshot({
		path: test.info().outputPath('native-save-dialog.png')
	});
	await expect(
		dialog.getByRole('combobox', { name: 'Folder', exact: true })
	).toHaveValue('');
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
	await page.goto('/app/home');
	await page
		.getByRole('button', { name: 'Show image 2 of 2', exact: true })
		.click();
	await page.getByLabel('Share post', { exact: true }).click();
	const share = page.locator('.post-share[open]');
	await expect(share.locator('.menu-items')).toHaveCSS('position', 'fixed');
	await expect(
		share.getByRole('button', { name: 'Repost', exact: true })
	).toBeVisible();
	await share
		.getByRole('checkbox', { name: 'Quote image 2 of 2', exact: true })
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
	await expect(
		share.getByRole('checkbox', { name: 'Quote image 2 of 2', exact: true })
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
	await page.goto('/app/home');
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
	await page.goto('/app/posts/did%3Aplc%3Abob/thread');
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
	).toHaveAttribute(
		'href',
		'/app/compose/reply?did=did%3Aplc%3Abob&rkey=nested'
	);
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
	await page.goto('/app/home');
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
	await page.goto('/app/activity');
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
	await page.goto('/app/profile/edit');
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
	await Promise.all([
		page.waitForEvent('dialog').then(async (dialog) => {
			expect(dialog.message()).toBe('Discard your unsaved profile changes?');
			await dialog.dismiss();
		}),
		close.click()
	]);
	await expect(page).toHaveURL(/\/app\/profile\/edit$/);
	await expect(
		page.getByRole('textbox', { name: 'Display name', exact: true })
	).toHaveValue('Unsaved maker name');
	expect(writes).toHaveLength(0);
	await Promise.all([
		page.waitForEvent('dialog').then(async (dialog) => {
			expect(dialog.message()).toBe('Discard your unsaved profile changes?');
			await dialog.accept();
		}),
		close.click()
	]);
	await expect(page).toHaveURL(/\/app\/profile\/did%3Aplc%3Aalice$/);
	expect(writes).toHaveLength(0);
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
	await page.goto('/app/profile/edit');
	await page
		.getByRole('textbox', { name: 'Display name', exact: true })
		.fill('Saved maker identity');
	await page.getByRole('button', { name: 'Save profile', exact: true }).click();
	await expect(page).toHaveURL(/\/app\/profile\/did%3Aplc%3Aalice$/);
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
	const identity = page.locator('.account-trigger .shell-identity strong');
	await expect(identity).toHaveText('Saved maker identity');
	await expect(page.locator('.account-trigger img')).toHaveAttribute(
		'src',
		'https://fixture.craftsky.test/saved-avatar.svg'
	);
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
