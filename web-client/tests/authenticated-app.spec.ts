import { test, expect, type BrowserContext, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
const base = 'http://127.0.0.1:19846';
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const bob = {
	id: 'c6dc7d6a-4bff-45f0-87ee-d42634646546',
	did: 'did:plc:bob',
	handle: 'bob.test',
	token: Buffer.alloc(32, 2).toString('base64url')
};
async function login(context: BrowserContext, who = alice) {
	await context.addCookies(
		[alice, bob].map((account) => ({
			name: `craftsky_account_${account.id}`,
			value: Buffer.from(JSON.stringify(account)).toString('base64url'),
			url: base,
			httpOnly: true,
			sameSite: 'Lax' as const
		}))
	);
	await context.addCookies([
		{
			name: 'craftsky_active',
			value: who.id,
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
}
async function compose(page: Page) {
	await page.goto('/compose');
	await expect(
		page.getByRole('heading', { name: 'New post', exact: true })
	).toBeVisible();
}
async function calls() {
	return (await (
		await fetch('http://127.0.0.1:19847/__test/requests')
	).json()) as {
		method: string;
		path: string;
		body: Record<string, unknown>;
		owner: string;
		key?: string;
		ifMatch?: string;
		bytes?: number;
		contentType?: string;
	}[];
}
async function imageFixture(page: Page) {
	const encoded = await page.evaluate(() => {
		const canvas = document.createElement('canvas');
		canvas.width = 8;
		canvas.height = 10;
		const context = canvas.getContext('2d')!;
		context.fillStyle = '#1535d6';
		context.fillRect(0, 0, 8, 10);
		return canvas.toDataURL('image/png').split(',')[1];
	});
	return Buffer.from(encoded, 'base64');
}
test.beforeEach(async ({ context, page }) => {
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await fetch('http://127.0.0.1:19847/__test/reset');
	await login(context);
});

test('publishes through AppView, returns to the feed and edits with the observed CID', async ({
	page
}) => {
	await compose(page);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('My new handknitted scarf #knitting');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL('/feed');
	await expect(
		page.getByText('My new handknitted scarf #knitting', { exact: true })
	).toBeVisible();
	const create = (await calls()).find(
		(call) => call.method === 'POST' && call.path === '/v1/posts'
	);
	expect(create?.key).toMatch(/^[0-9a-f-]{36}$/);
	expect(create?.body).toMatchObject({ langs: ['en'], sponsored: false });
	await page.goto('/compose/edit?did=did%3Aplc%3Aalice&rkey=created-1');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('My edited scarf');
	await page.getByRole('button', { name: 'Save changes' }).click();
	await expect(page).toHaveURL(/\/posts\//);
	await expect(
		page.getByText('My edited scarf', { exact: true })
	).toBeVisible();
	const edit = (await calls()).find((call) => call.method === 'PATCH');
	expect(edit?.ifMatch).toBe('cid-created-1');
	expect(edit?.body).toMatchObject({
		changes: { text: 'My edited scarf', facets: null }
	});
});
test('an ambiguous write freezes the editor and retries the exact saved key and payload', async ({
	page
}) => {
	await compose(page);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Ambiguous but created once');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Waiting for confirmation' })
	).toBeVisible();
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toBeDisabled();
	await page
		.getByRole('button', { name: 'Check the same action', exact: true })
		.click();
	await expect(page).toHaveURL('/feed');
	const attempts = (await calls()).filter(
		(call) => call.method === 'POST' && call.path === '/v1/posts'
	);
	expect(attempts).toHaveLength(2);
	expect(attempts[0].key).toBe(attempts[1].key);
	expect(attempts[0].body).toEqual(attempts[1].body);
});
test('inline mention and hashtag pickers support keyboard selection and publish resolved facets', async ({
	page
}) => {
	const scopes: string[] = [];
	await page.route('**/api/appview/v1/facets/mentions?*', (route) => {
		scopes.push(route.request().headers()['x-craftsky-account-id']);
		return route.fulfill({
			json: {
				items: [
					{
						did: alice.did,
						handle: alice.handle,
						displayName: 'Alice Maker',
						avatar: 'https://fixture.craftsky.test/avatar.png'
					}
				]
			}
		});
	});
	await page.route('**/api/appview/v1/facets/hashtags?*', (route) =>
		route.fulfill({ json: { items: [{ tag: 'knitting', postCount: 12 }] } })
	);
	await compose(page);
	const editor = page.getByRole('textbox', {
		name: 'What would you like to share?',
		exact: true
	});
	await editor.fill('Thanks @ali');
	await expect(
		page.getByRole('listbox', { name: 'Account suggestions' })
	).toBeVisible();
	await expect(page.getByRole('option', { name: /Alice Maker/ })).toBeVisible();
	await expect(
		page.getByRole('option', { name: /Alice Maker/ }).locator('.avatar img')
	).toHaveAttribute('src', 'https://fixture.craftsky.test/avatar.png');
	await editor.press('ArrowDown');
	await editor.press('Enter');
	await expect(editor).toHaveValue('Thanks @alice.test ');
	await editor.press('End');
	await editor.pressSequentially('#kni');
	await expect(
		page.getByRole('listbox', { name: 'Hashtag suggestions' })
	).toBeVisible();
	await expect(
		page.getByRole('option', { name: '#knitting 12 posts', exact: true })
	).toBeVisible();
	await editor.press('Enter');
	await expect(editor).toHaveValue('Thanks @alice.test #knitting ');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL('/feed');
	const created = (await calls()).find(
		(call) => call.path === '/v1/posts' && call.method === 'POST'
	);
	expect(created?.body.facets).toEqual(
		expect.arrayContaining([
			expect.objectContaining({
				features: [{ $type: 'app.bsky.richtext.facet#mention', did: alice.did }]
			}),
			expect.objectContaining({
				features: [{ $type: 'app.bsky.richtext.facet#tag', tag: 'knitting' }]
			})
		])
	);
	expect(scopes.every((id) => id === alice.id)).toBe(true);
});
test('post reporting submits the supported spam reason through the real form', async ({
	page,
	context
}) => {
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	await page.getByLabel('More post actions').click();
	await expect(
		page.getByRole('button', { name: 'Report post', exact: true })
	).toHaveCount(0);
	await login(context, bob);
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	await page.getByLabel('More post actions').click();
	await page.getByRole('button', { name: 'Report post', exact: true }).click();
	await page
		.getByRole('radio', {
			name: 'Spam, misleading or off-topic content',
			exact: true
		})
		.click();
	await page.getByRole('radio', { name: 'Spam', exact: true }).check();
	await page
		.getByLabel('Details', { exact: true })
		.fill('Repeated unwanted promotions');
	await page
		.getByRole('button', { name: 'Submit report', exact: true })
		.click();
	await expect
		.poll(
			async () =>
				(await calls()).filter(
					(call) => call.method === 'POST' && call.path.endsWith('/reports')
				).length
		)
		.toBe(1);
	const report = (await calls()).find(
		(call) => call.method === 'POST' && call.path.endsWith('/reports')
	);
	expect(report?.owner).toBe(bob.did);
	expect(report?.path).toBe('/v1/posts/did:plc:alice/first/reports');
	expect(report?.body).toEqual({
		reasonType: 'spam',
		details: 'Repeated unwanted promotions'
	});
});
test('accepted publication survives a local draft cleanup failure', async ({
	page
}) => {
	await page.addInitScript(() => {
		const remove = IDBObjectStore.prototype.delete;
		IDBObjectStore.prototype.delete = function (key) {
			if (this.name === 'drafts')
				throw new DOMException('Fixture storage failure', 'QuotaExceededError');
			return remove.call(this, key);
		};
	});
	await compose(page);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Published despite a local storage cleanup failure');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL('/feed');
	await expect(
		page.getByText('Published despite a local storage cleanup failure', {
			exact: true
		})
	).toBeVisible();
	expect(
		(await calls()).filter(
			(call) => call.method === 'POST' && call.path === '/v1/posts'
		)
	).toHaveLength(1);
});
test('home interactions and profile navigation use the authenticated account', async ({
	page
}) => {
	await page.goto('/feed');
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	const like = page.getByRole('button', { name: /Like$/ }).first();
	await like.click();
	await expect(like).toHaveAttribute('aria-pressed', 'true');
	await like.click();
	await expect(like).toHaveAttribute('aria-pressed', 'false');
	await expect(page.locator('.post-card .date').first()).toHaveAttribute(
		'href',
		'/posts/did%3Aplc%3Aalice/first'
	);
	await expect(page.locator('.post-card .author a').first()).toHaveAttribute(
		'href',
		'/profiles/did%3Aplc%3Aalice'
	);
	await page.locator('.post-card .date').first().click();
	await expect(page).toHaveURL(/\/posts\/did%3Aplc%3Aalice\/first/);
	await page.locator('.post-card .author a').first().click();
	await expect(page).toHaveURL(/\/profiles\/did%3Aplc%3Aalice/);
	await expect(
		page.getByRole('heading', { name: 'Alice Maker', exact: true })
	).toBeVisible();
	await page
		.getByRole('navigation', { name: 'Profile content' })
		.getByRole('link', { name: 'Posts', exact: true })
		.click();
	await expect(
		page.getByText('A little knitting for the weekend.', { exact: true })
	).toBeVisible();
	const likes = (await calls()).filter((call) => call.path.endsWith('/likes'));
	expect(likes.map((call) => call.method)).toEqual(['POST', 'DELETE']);
	expect(likes.every((call) => call.owner === 'did:plc:alice')).toBe(true);
});
test('compact quote previews derive their route from AT URI and show safe selected images and sponsorship', async ({
	page
}) => {
	let unsafe = false;
	const image = await imageFixture(page);
	await page.route('https://cdn.example/**', (route) =>
		route.fulfill({ contentType: 'image/png', body: image })
	);
	await page.route(
		/\/api\/appview\/v1\/posts\/[^/]+\/first$/,
		async (route) => {
			const response = await route.fetch();
			const post = await response.json();
			return route.fulfill({
				response,
				json: {
					...post,
					external: {
						uri: 'https://example.com/pattern',
						title: 'Pattern website',
						description: 'A useful pattern',
						thumb: {
							url: unsafe
								? 'javascript:alert(1)'
								: 'https://cdn.example/external.png',
							cid: 'thumb',
							mime: 'image/png',
							size: image.length
						}
					},
					quoteView: {
						state: 'visible',
						revealable: false,
						post: {
							uri: 'at://did:plc:bob/social.craftsky.feed.post/quoted',
							cid: 'quoted-cid',
							text: 'Quoted project story',
							author: {
								did: bob.did,
								handle: bob.handle,
								displayName: 'Bob Maker'
							},
							createdAt: '2026-11-01T12:00:00Z',
							sponsored: true,
							project: {
								common: {
									craftType: 'social.craftsky.feed.defs#sewing',
									title: 'Quoted linen dress'
								}
							},
							images: [
								{
									thumb: unsafe
										? 'javascript:alert(1)'
										: 'https://cdn.example/quote-selected.png',
									fullsize: unsafe
										? 'javascript:alert(1)'
										: 'https://cdn.example/quote-selected.png',
									alt: 'Selected quote image'
								}
							]
						}
					}
				}
			});
		}
	);
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	const quote = page.locator('.quote-card');
	await expect(quote.locator('.summary-content')).toHaveAttribute(
		'href',
		'/posts/did%3Aplc%3Abob/quoted'
	);
	await expect(
		quote.getByRole('heading', { name: 'Quoted linen dress' })
	).toBeVisible();
	await expect(
		quote.getByText('Sponsored / commercial relationship', { exact: true })
	).toBeVisible();
	await expect(
		quote.getByRole('img', { name: 'Selected quote image' })
	).toHaveAttribute('src', 'https://cdn.example/quote-selected.png');
	await expect(page.locator('.external-card img')).toHaveAttribute(
		'src',
		'https://cdn.example/external.png'
	);
	expect(await page.locator('img[src^="javascript:"]').count()).toBe(0);
	unsafe = true;
	await page.reload();
	await expect(
		page.getByText('Quoted project story', { exact: true })
	).toBeVisible();
	await expect(page.locator('.quote-card img')).toHaveCount(0);
	await expect(page.locator('.external-card img')).toHaveCount(0);
});
test('project pattern and material annotations link to their makers and hashtags', async ({
	page
}) => {
	const facet = (
		text: string,
		token: string,
		feature: Record<string, string>
	) => {
		const start = text.indexOf(token);
		return [
			{
				index: {
					byteStart: Buffer.byteLength(text.slice(0, start)),
					byteEnd: Buffer.byteLength(text.slice(0, start + token.length))
				},
				features: [feature]
			}
		];
	};
	const material = '🧶 Linen from @bob.test';
	await page.route(
		/\/api\/appview\/v1\/posts\/[^/]+\/first$/,
		async (route) => {
			const response = await route.fetch();
			const post = await response.json();
			return route.fulfill({
				response,
				json: {
					...post,
					project: {
						common: {
							craftType: 'social.craftsky.feed.defs#sewing',
							title: 'Faceted linen dress',
							pattern: {
								name: '#LinenDress',
								nameFacets: facet('#LinenDress', '#LinenDress', {
									$type: 'app.bsky.richtext.facet#tag',
									tag: 'LinenDress'
								}),
								designer: '@bob.test',
								designerFacets: facet('@bob.test', '@bob.test', {
									$type: 'app.bsky.richtext.facet#mention',
									did: bob.did
								}),
								publisher: '@alice.test',
								publisherFacets: facet('@alice.test', '@alice.test', {
									$type: 'app.bsky.richtext.facet#mention',
									did: alice.did
								}),
								url: 'javascript:alert(1)'
							},
							materials: [
								{
									text: material,
									facets: facet(material, '@bob.test', {
										$type: 'app.bsky.richtext.facet#mention',
										did: bob.did
									})
								}
							]
						}
					}
				}
			});
		}
	);
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	const details = page.locator('.post-card');
	await expect(
		details.getByRole('link', { name: '#LinenDress', exact: true })
	).toHaveAttribute('href', /\/search\/tags/);
	await expect(
		details.getByRole('link', { name: '@bob.test', exact: true })
	).toHaveCount(2);
	for (const link of await details
		.getByRole('link', { name: '@bob.test', exact: true })
		.all())
		await expect(link).toHaveAttribute('href', '/profiles/did%3Aplc%3Abob');
	await expect(
		details.getByRole('link', { name: '@alice.test', exact: true })
	).toHaveAttribute('href', '/profiles/did%3Aplc%3Aalice');
	await expect(details.locator('a[href^="javascript:"]')).toHaveCount(0);
});
test('a conflicting published edit keeps unsaved changes in the editor without automatically creating a draft', async ({
	page
}) => {
	await page.goto('/compose/edit?did=did%3Aplc%3Aalice&rkey=first');
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Conflict changes');
	await page.getByRole('button', { name: 'Save changes' }).click();
	await expect(page.getByRole('alert')).toContainText(
		'changed while you were editing'
	);
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toBeEnabled();
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toHaveValue('Conflict changes');
	await page.goto('/drafts');
	await expect(page.getByText('Conflict changes', { exact: true })).toHaveCount(
		0
	);
	const edit = (await calls()).find(
		(call) =>
			call.method === 'PATCH' && call.path === '/v1/posts/did:plc:alice/first'
	);
	expect(edit?.ifMatch).toBe('cid-first');
	expect(edit?.body).toMatchObject({
		changes: { text: 'Conflict changes', facets: null }
	});
});
test('editing an existing reply retains its structural context and excludes invalid video and sponsorship controls', async ({
	page
}) => {
	await page.route(
		/\/api\/appview\/v1\/posts\/[^/]+\/first\/edit$/,
		async (route) => {
			const response = await route.fetch();
			const seed = await response.json();
			const parent = {
				uri: 'at://did:plc:bob/social.craftsky.feed.post/parent',
				cid: 'parent-cid'
			};
			return route.fulfill({
				response,
				json: {
					...seed,
					post: { ...seed.post, reply: { root: parent, parent } }
				}
			});
		}
	);
	await page.goto('/compose/edit?did=did%3Aplc%3Aalice&rkey=first');
	await expect(
		page.getByRole('textbox', { name: 'Write your comment', exact: true })
	).toBeVisible();
	await expect(page.locator('input[accept="video/mp4"]')).toHaveCount(0);
	for (const input of await page.locator('input[type="file"]').all())
		await expect(input).not.toHaveAttribute('accept', /video\//);
	await expect(
		page.getByRole('checkbox', { name: 'Sponsored content', exact: true })
	).toHaveCount(0);
});
test('restores private image bytes from IndexedDB after reload and isolates another account', async ({
	page,
	context
}) => {
	await compose(page);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Alice private quilt draft');
	const png = await imageFixture(page);
	await page
		.locator('input[type=file][multiple]')
		.setInputFiles({ name: 'quilt.png', mimeType: 'image/png', buffer: png });
	await page.getByLabel('Image description').fill('A blue quilt');
	await page.getByRole('button', { name: 'Edit photo', exact: true }).click();
	await expect(page.locator('.save-draft')).toBeDisabled();
	for (let i = 0; i < 5; i++)
		await page
			.getByRole('button', { name: 'Crop bottom right corner' })
			.press('Shift+ArrowLeft');
	await page.getByRole('button', { name: 'Rotate photo' }).click();
	await page.getByRole('button', { name: 'Apply edit', exact: true }).click();
	await expect(
		page.getByRole('img', { name: 'A blue quilt' })
	).toHaveJSProperty('naturalWidth', 10);
	await page.getByRole('button', { name: 'Back to post', exact: true }).click();
	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page.getByText('Draft saved', { exact: true })).toBeVisible();
	await page.goto('/drafts');
	await page.reload();
	await page.getByRole('link', { name: 'Continue editing' }).click();
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toHaveValue('Alice private quilt draft');
	await page.getByRole('button', { name: /Photos.*Reorder photos/ }).click();
	await expect(page.getByLabel('Image description')).toHaveValue(
		'A blue quilt'
	);
	await expect(
		page.getByRole('img', { name: 'A blue quilt' })
	).toHaveJSProperty('naturalWidth', 10);
	await expect(
		page.getByRole('img', { name: 'A blue quilt' })
	).toHaveJSProperty('naturalHeight', 4);
	await login(context, bob);
	await page.goto('/drafts');
	await expect(
		page.getByText('Alice private quilt draft', { exact: true })
	).toHaveCount(0);
	await expect(
		page
			.locator('.draft-workspace')
			.getByRole('heading', { name: 'Drafts', level: 2, exact: true })
	).toBeVisible();
});
test('schedules private media, restores the schedule and requests publication now', async ({
	page
}) => {
	await compose(page);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Tomorrow’s knitting');
	await page.locator('input[type=file][multiple]').setInputFiles({
		name: 'knitting.png',
		mimeType: 'image/png',
		buffer: await imageFixture(page)
	});
	await page.getByLabel('Image description').fill('Knitting on blue needles');
	await page.getByRole('button', { name: 'Back to post', exact: true }).click();
	await page.getByRole('button', { name: 'When', exact: true }).click();
	await page
		.getByRole('menuitemradio', { name: 'Schedule for later', exact: true })
		.click();
	const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
	const local = new Date(
		tomorrow.getTime() - tomorrow.getTimezoneOffset() * 60000
	)
		.toISOString()
		.slice(0, 16);
	await page.getByLabel('Publish date and time').fill(local);
	await page.getByRole('button', { name: 'Set time', exact: true }).click();
	await page
		.getByRole('button', { name: 'Save schedule', exact: true })
		.click();
	await expect(page).toHaveURL('/scheduled');
	const schedule = page
		.getByRole('listitem')
		.filter({ has: page.getByText('Tomorrow’s knitting', { exact: true }) });
	await expect(schedule).toBeVisible();
	await schedule
		.getByRole('link', { name: 'Edit scheduled post', exact: true })
		.click();
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toHaveValue('Tomorrow’s knitting');
	await page.getByRole('button', { name: 'Publish now', exact: true }).click();
	await expect(page).toHaveURL('/scheduled');
	await expect(
		page.getByText('Tomorrow’s knitting', { exact: true })
	).toHaveCount(0);
	const requests = await calls();
	expect(
		requests.some(
			(call) =>
				call.path.startsWith('/v1/scheduled-post-media/') &&
				call.method === 'PUT' &&
				!!call.bytes
		)
	).toBe(true);
	expect(requests.some((call) => call.path === '/v1/blobs/images')).toBe(false);
	const scheduled = requests.find(
		(call) => call.path === '/v1/scheduled-posts' && call.method === 'POST'
	);
	expect(scheduled?.body).toMatchObject({
		payload: { kind: 'standard', text: 'Tomorrow’s knitting', sponsored: false }
	});
	expect(
		requests.some((call) => call.path === '/v1/posts' && call.method === 'POST')
	).toBe(false);
});
test('supports mobile navigation and project fields without horizontal overflow', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/feed');
	await page.getByRole('button', { name: 'Create post', exact: true }).click();
	const chooser = page.getByRole('dialog', {
		name: 'What would you like to share?',
		exact: true
	});
	await expect(chooser).toBeVisible();
	await chooser.locator('a[href="/compose/project"]').click();
	await expect(
		page.getByRole('heading', { name: 'Project post', exact: true })
	).toBeVisible();
	await page
		.getByRole('combobox', { name: 'Primary craft required', exact: true })
		.click();
	await page.getByRole('option', { name: 'Sewing', exact: true }).click();
	await page.getByLabel('Project title', { exact: true }).fill('Linen dress');
	await page
		.getByLabel('Project description')
		.fill('A summer dress made from linen');
	await page
		.getByRole('button', { name: 'Sewing details', exact: true })
		.click();
	await page.getByLabel('Size made').fill('12');
	await page.getByLabel('Fit notes').fill('Shortened the sleeves');
	await page
		.getByRole('region', { name: 'Sewing details', exact: true })
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	await expect(
		page.getByRole('button', { name: 'Sewing details', exact: true })
	).toBeFocused();
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
	const accessibility = await new AxeBuilder({ page })
		.include('.composer')
		.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
		.analyze();
	expect(accessibility.violations).toEqual([]);
	await page.evaluate(() => window.scrollTo(0, 0));
	await page.screenshot({
		path: test.info().outputPath('mobile-project-editor.png'),
		fullPage: false
	});
	await page.getByRole('button', { name: 'Clear Status', exact: true }).click();
	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page.getByText('Draft saved', { exact: true })).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'Open navigation' }).click();
	await page
		.getByRole('navigation', { name: 'App navigation' })
		.getByRole('link', { name: 'Drafts', exact: true })
		.click();
	await expect(
		page.getByRole('heading', { name: 'Drafts', level: 1, exact: true })
	).toBeVisible();
	const project = page
		.getByRole('listitem')
		.filter({ has: page.getByText('Linen dress', { exact: true }) });
	await project
		.getByRole('link', { name: 'Continue editing', exact: true })
		.click();
	await expect(
		page.getByRole('combobox', { name: 'Status', exact: true })
	).toHaveValue('');
	await expect(
		page.getByRole('textbox', { name: 'Project description', exact: true })
	).toHaveValue('A summer dress made from linen');
	await page
		.getByRole('button', { name: 'Sewing details', exact: true })
		.click();
	await expect(
		page.getByRole('textbox', { name: 'Size made', exact: true })
	).toHaveValue('12');
	await expect(
		page.getByRole('textbox', { name: 'Fit notes', exact: true })
	).toHaveValue('Shortened the sleeves');
	await page
		.getByRole('region', { name: 'Sewing details', exact: true })
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	expect(
		(await calls()).filter(
			(call) => call.method === 'POST' && call.path === '/v1/posts'
		)
	).toHaveLength(0);
});

test('project detail pages fill the compact viewport and keep changes on Back, Escape and draft reopening', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/compose/project');
	await page
		.getByRole('combobox', { name: 'Primary craft required', exact: true })
		.click();
	await page.getByRole('option', { name: 'Sewing', exact: true }).click();
	await page
		.getByRole('textbox', { name: 'Project title', exact: true })
		.fill('Three detail pages');
	await page
		.getByRole('textbox', { name: 'Project description', exact: true })
		.fill('A summer project with details worth keeping.');

	const open = async (title: string) => {
		await page.getByRole('button', { name: title, exact: true }).click();
		const detail = page.getByRole('region', { name: title, exact: true });
		await expect(detail).toBeVisible();
		await expect(page.locator('.project-detail-page')).toHaveCount(1);
		return detail;
	};
	const back = async (title: string) => {
		await page
			.getByRole('region', { name: title, exact: true })
			.getByRole('button', { name: 'Back to project', exact: true })
			.click();
		await expect(
			page.getByRole('region', { name: title, exact: true })
		).toBeHidden();
		await expect(
			page.getByRole('button', { name: title, exact: true })
		).toBeFocused();
		await expect(
			page.getByRole('button', { name: title, exact: true })
		).toBeInViewport();
	};

	const pattern = await open('Pattern details');
	const bounds = await pattern.boundingBox();
	expect(bounds).not.toBeNull();
	expect(bounds!.x).toBeCloseTo(0, 0);
	expect(bounds!.y).toBeCloseTo(0, 0);
	expect(bounds!.width).toBeCloseTo(390, 0);
	expect(bounds!.height).toBeCloseTo(844, 0);
	await expect(pattern.getByRole('button', { name: /Save/ })).toHaveCount(0);
	await pattern
		.getByRole('textbox', { name: 'Designer', exact: true })
		.fill('@bob.test');
	await pattern
		.getByRole('textbox', { name: 'Link', exact: true })
		.fill('https://example.com/pattern');
	await pattern
		.getByRole('textbox', { name: 'Link', exact: true })
		.press('Enter');
	await expect(pattern).toBeVisible();
	await pattern
		.getByRole('switch', { name: 'Self-drafted pattern', exact: true })
		.press('Enter');
	await expect(page.locator('.composer-error')).toHaveCount(0);
	await back('Pattern details');
	await page.goForward();
	await expect(pattern).toBeVisible();
	await expect(
		pattern.getByRole('textbox', { name: 'Designer', exact: true })
	).toHaveValue('@bob.test');
	await page.goBack();
	await expect(pattern).toBeHidden();

	const materials = await open('Materials and style');
	await materials
		.getByRole('textbox', { name: 'Materials', exact: true })
		.fill('Merino wool');
	await materials.getByRole('button', { name: 'Add', exact: true }).click();
	await materials
		.getByRole('textbox', { name: 'Materials', exact: true })
		.fill('Brass buttons');
	await materials.getByRole('button', { name: 'Add', exact: true }).click();
	await back('Materials and style');
	const craft = await open('Sewing details');
	await craft
		.getByRole('textbox', { name: 'Size made', exact: true })
		.fill('12');
	await craft
		.getByRole('textbox', { name: 'Fit notes', exact: true })
		.fill('Shortened sleeves');
	await back('Sewing details');

	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page.getByText('Draft saved', { exact: true })).toBeVisible();
	await page.goto('/drafts');
	const row = page
		.getByRole('listitem')
		.filter({ has: page.getByText('Three detail pages', { exact: true }) });
	await row
		.getByRole('link', { name: 'Continue editing', exact: true })
		.click();
	const restoredPattern = await open('Pattern details');
	await expect(
		restoredPattern.getByRole('textbox', { name: 'Designer', exact: true })
	).toHaveValue('@bob.test');
	await expect(
		restoredPattern.getByRole('textbox', { name: 'Link', exact: true })
	).toHaveValue('https://example.com/pattern');
	await page.keyboard.press('Escape');
	await expect(restoredPattern).toBeHidden();
	await expect(
		page.getByRole('button', { name: 'Pattern details', exact: true })
	).toBeFocused();
	const restoredMaterials = await open('Materials and style');
	await expect(
		restoredMaterials.getByText('Merino wool', { exact: true })
	).toBeVisible();
	await expect(
		restoredMaterials.getByText('Brass buttons', { exact: true })
	).toBeVisible();
	await expect(
		restoredMaterials.getByRole('textbox', { name: 'Materials', exact: true })
	).toHaveValue('');
	await back('Materials and style');
	const restoredCraft = await open('Sewing details');
	await expect(
		restoredCraft.getByRole('textbox', { name: 'Size made', exact: true })
	).toHaveValue('12');
	await expect(
		restoredCraft.getByRole('textbox', { name: 'Fit notes', exact: true })
	).toHaveValue('Shortened sleeves');
	await back('Sewing details');
	expect(
		(await calls()).filter(
			(call) => call.method === 'POST' && call.path === '/v1/posts'
		)
	).toHaveLength(0);
});

test('indexed edits replace stale detail and cached feed content without a refresh', async ({
	page
}) => {
	await compose(page);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('Before indexing');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL('/feed');
	const card = page
		.locator('.member-post')
		.filter({ has: page.getByText('Before indexing', { exact: true }) })
		.first();
	await card.locator('summary[aria-label="More post actions"]').click();
	await card.getByRole('link', { name: 'Edit', exact: true }).click();
	await expect(
		page.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
	).toHaveValue('Before indexing');
	let indexed = false;
	let reads = 0;
	const stale = await page.evaluate(async () =>
		(await fetch('/api/appview/v1/posts/did%3Aplc%3Aalice/created-1')).json()
	);
	await page.route(
		'**/api/appview/v1/posts/did%3Aplc%3Aalice/created-1',
		async (route) => {
			if (route.request().method() !== 'GET') return route.continue();
			reads++;
			if (!indexed) return route.fulfill({ json: stale });
			return route.continue();
		}
	);
	await page
		.getByRole('textbox', {
			name: 'What would you like to share?',
			exact: true
		})
		.fill('After indexing');
	await page.getByRole('button', { name: 'Save changes' }).click();
	await expect(page).toHaveURL(/\/posts\//);
	await expect(
		page.getByText('Before indexing', { exact: true })
	).toBeVisible();
	await expect.poll(() => reads).toBeGreaterThanOrEqual(2);
	indexed = true;
	await expect(page.getByText('After indexing', { exact: true })).toBeVisible();
	// Simulate a stale cached response on return, even though the edit was observed.
	indexed = false;
	await page
		.getByRole('link', { name: 'Home', exact: true })
		.filter({ visible: true })
		.first()
		.click();
	await expect(page).toHaveURL('/feed');
	await expect(page.getByText('After indexing', { exact: true })).toBeVisible();
	await expect(page.getByText('Before indexing', { exact: true })).toHaveCount(
		0
	);
});

test('root opens the mobile feed route and navigation uses canonical paths', async ({
	page
}) => {
	await page.goto('/');
	await expect(page).toHaveURL('/feed');
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	const navigation = page.getByRole('navigation', { name: 'App navigation' });
	for (const [name, path] of [
		['Home', '/feed'],
		['Notifications', '/notifications'],
		['Messages', '/messages'],
		['Profile', '/profile']
	])
		await expect(
			navigation.getByRole('link', { name: new RegExp('^' + name) })
		).toHaveAttribute('href', path);
	await page.goto('/projects?craft=knitting');
	await expect(page).toHaveURL('/feed?craft=knitting');
});
