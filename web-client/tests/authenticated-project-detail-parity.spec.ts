import { test, expect, type Page, type Request } from '@playwright/test';
import type { Draft } from '../src/lib/features/drafts/store.ts';
import type { JsonObject } from '../src/lib/api/types.ts';

const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const draftId = '3f9c35e5-270e-420d-94e4-7380780b6c56';
test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('**/api/service-status', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await context.addCookies([
		{
			name: `craftsky_account_${alice.id}`,
			value: Buffer.from(JSON.stringify(alice)).toString('base64url'),
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		},
		{
			name: 'craftsky_active',
			value: alice.id,
			url: 'http://127.0.0.1:19846',
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	// Draft tests guard against accidental publication. The publish-and-edit test removes this guard.
	await page.route('**/api/appview/v1/posts', async (route) => {
		if (route.request().method() === 'POST')
			throw new Error('Project detail tests must not publish');
		await route.continue();
	});
});
async function seed(page: Page, common: JsonObject, details?: JsonObject) {
	await page.goto('/drafts');
	await expect(
		page.getByRole('heading', { level: 1, name: 'Drafts', exact: true })
	).toBeVisible();
	const draft: Draft = {
		id: draftId,
		owner: alice.did,
		kind: 'project',
		text: 'Project details reference',
		langs: ['en'],
		sponsored: false,
		images: [],
		updatedAt: '2026-11-03T09:00:00Z',
		project: {
			common: {
				craftType: 'social.craftsky.feed.defs#sewing',
				title: 'Native project fields',
				...common
			},
			...(details ? { details } : {})
		}
	};
	await page.evaluate(
		(draft: unknown) =>
			new Promise<void>((resolve, reject) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onupgradeneeded = () => {
					const store = request.result.createObjectStore('drafts', {
						keyPath: ['owner', 'id']
					});
					store.createIndex('owner', 'owner');
				};
				request.onerror = () => reject(request.error);
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readwrite');
					tx.objectStore('drafts').put(draft);
					tx.oncomplete = () => {
						db.close();
						resolve();
					};
					tx.onerror = () => {
						db.close();
						reject(tx.error);
					};
				};
			}),
		draft as unknown
	);
	await page.goto(`/compose/project?draft=${draftId}`);
	await expect(
		page.getByRole('textbox', { name: 'Project title', exact: true })
	).toHaveValue('Native project fields');
}
async function storedProject(page: Page) {
	return page.evaluate(
		({ owner, id }) =>
			new Promise<JsonObject>((resolve, reject) => {
				const request = indexedDB.open('craftsky-private-drafts', 1);
				request.onerror = () => reject(request.error);
				request.onsuccess = () => {
					const db = request.result,
						tx = db.transaction('drafts', 'readonly'),
						read = tx.objectStore('drafts').get([owner, id]);
					read.onsuccess = () => resolve(read.result.project);
					read.onerror = () => reject(read.error);
					tx.oncomplete = () => db.close();
				};
			}),
		{ owner: alice.did, id: draftId }
	);
}

test('project text limits preserve complete emoji and defer truncation until IME commits', async ({
	page
}) => {
	await seed(page, {});
	const title = page.getByRole('textbox', {
		name: 'Project title',
		exact: true
	});
	await title.fill('🧶'.repeat(150));
	await expect(title).toHaveValue('🧶'.repeat(125));
	await expect(
		page.getByRole('textbox', { name: 'Duration', exact: true })
	).toHaveCount(0);
	await title.evaluate((input: HTMLInputElement) => {
		input.dispatchEvent(
			new CompositionEvent('compositionstart', { bubbles: true })
		);
		input.value = '界'.repeat(180);
		input.dispatchEvent(
			new InputEvent('input', { bubbles: true, isComposing: true })
		);
	});
	await expect(title).toHaveValue('界'.repeat(180));
	await title.dispatchEvent('compositionend');
	await expect(title).toHaveValue('界'.repeat(166));
});
async function save(page: Page) {
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(page.getByText('Draft saved', { exact: true })).toBeVisible();
}

test('project composer orders the main fields and opens craft and pattern pages', async ({
	page
}) => {
	await seed(page, { craftType: 'social.craftsky.feed.defs#spinning' });
	const craft = page.getByRole('button', {
		name: 'Spinning details',
		exact: true
	});
	const pattern = page.getByRole('button', {
		name: 'Pattern details',
		exact: true
	});
	const back = page.getByRole('button', {
		name: 'Back to project',
		exact: true
	});
	const additional = page.getByRole('searchbox', {
		name: 'Additional crafts',
		exact: true
	});
	const name = page.getByRole('textbox', { name: 'Pattern name', exact: true });
	await expect(additional).toHaveCount(0);
	await expect(pattern).toHaveCount(0);
	await expect(name).toHaveCount(0);
	await craft.click();
	await expect(
		page.getByRole('heading', { name: 'Spinning details', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'What are you making?' })
	).toHaveCount(0);
	await additional.fill('weaving');
	await page.getByRole('checkbox', { name: 'Weaving', exact: true }).click();
	await expect(pattern).toHaveCount(0);
	await back.click();
	await expect(craft).toBeFocused();
	await expect(pattern).toBeVisible();
	await expect(craft).toHaveCSS('border-bottom-width', '0px');
	await expect(pattern).toHaveCSS('border-bottom-width', '0px');
	const [craftRow, patternRow] = await Promise.all([
		craft.boundingBox(),
		pattern.boundingBox()
	]);
	expect(patternRow!.y - (craftRow!.y + craftRow!.height)).toBeCloseTo(4, 1);

	const description = page.getByRole('textbox', {
		name: 'Project description'
	});
	const mainFields = [
		page.getByRole('heading', { name: /^Media/ }),
		page.getByRole('combobox', { name: 'Primary craft required', exact: true }),
		craft,
		pattern,
		page.getByRole('combobox', { name: 'Status required', exact: true }),
		page.getByRole('textbox', { name: 'Project title', exact: true }),
		description
	];
	const positions = await Promise.all(
		mainFields.map((field) =>
			field.evaluate((element) => element.getBoundingClientRect().top)
		)
	);
	expect(positions).toEqual([...positions].sort((a, b) => a - b));
	expect(new Set(positions).size).toBe(mainFields.length);
	await expect(description).toHaveAttribute(
		'placeholder',
		'Tell everyone about your project.\n\nYou can @mention people, use #hashtags or add https://links.com.'
	);
	await expect(
		page.getByRole('heading', { name: 'Add more details' })
	).toHaveCount(0);
	await pattern.click();
	await expect(
		page.getByRole('textbox', { name: 'Designer', exact: true })
	).toHaveAttribute('placeholder', 'Add or @mention pattern designer');
	await expect(
		page.getByRole('textbox', { name: 'Publisher', exact: true })
	).toHaveAttribute('placeholder', 'Add or @mention pattern publisher');
	await name.fill('Handwoven scarf');
	await back.click();
	await craft.click();
	await page
		.getByRole('button', { name: 'Remove Weaving from Additional crafts' })
		.click();
	await back.click();
	await pattern.click();
	await expect(name).toHaveValue('Handwoven scarf');
	await back.click();
	await save(page);
	expect((await storedProject(page)).common).toMatchObject({
		additionalCrafts: [],
		patterns: [{ name: 'Handwoven scarf' }]
	});
});

test('a saved dyeing pattern stays editable until its last detail is cleared', async ({
	page
}) => {
	await seed(page, {
		craftType: 'social.craftsky.feed.defs#dyeing',
		additionalCrafts: [{ craftType: 'social.craftsky.feed.defs#spinning' }],
		pattern: { url: 'https://example.com/dyeing' }
	});
	const details = page.getByRole('button', { name: 'Pattern details' });
	await details.click();
	const link = page.getByRole('textbox', { name: 'Link', exact: true });
	await expect(link).toHaveValue('https://example.com/dyeing');
	await link.fill('');
	await expect(link).toBeFocused();
	await page.getByRole('button', { name: 'Back to project' }).click();
	await expect(details).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Dyeing details' })
	).toBeFocused();
	await save(page);
	await page.goto(`/compose/project?draft=${draftId}`);
	await expect(
		page.getByRole('button', { name: 'Dyeing details' })
	).toBeVisible();
	await expect(details).toHaveCount(0);
});

for (const [craft, label] of [
	['weaving', 'Pattern or draft'],
	['felting', 'Pattern or template']
]) {
	test(`${craft} labels every pattern row as ${label.toLowerCase()}`, async ({
		page
	}) => {
		await seed(page, { craftType: `social.craftsky.feed.defs#${craft}` });
		await page.getByRole('button', { name: label, exact: true }).click();
		await expect(
			page.getByRole('heading', { name: label, exact: true })
		).toBeVisible();
		const names = page.getByRole('textbox', {
			name: `${label} name`,
			exact: true
		});
		await expect(names).toHaveCount(1);
		await expect(names.first()).toHaveAttribute('placeholder', '#PatternName');
		await expect(names.first()).toHaveValue('');
		await page.getByRole('button', { name: 'Add another pattern' }).click();
		await expect(names).toHaveCount(2);
		await expect(names.nth(1)).toHaveAttribute('placeholder', '#PatternName');
		await expect(names.nth(1)).toHaveValue('');
		await names.nth(1).fill('#Reference');
		await page.getByRole('button', { name: 'Back to project' }).click();
		await save(page);
		expect((await storedProject(page)).common).toMatchObject({
			patterns: expect.arrayContaining([
				expect.objectContaining({ name: '#Reference' })
			])
		});
	});
}

test('an older draft needs a status to publish but can be saved before choosing one', async ({
	page
}) => {
	await seed(page, {});
	const status = page.getByRole('combobox', {
		name: 'Status required',
		exact: true
	});
	await expect(status).toHaveValue('');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(status).toHaveAttribute('aria-invalid', 'true');
	await expect(
		page.getByText('Choose a project status.', { exact: true })
	).toBeVisible();
	await page
		.getByRole('textbox', { name: 'Project title', exact: true })
		.fill('Edited draft');
	await save(page);
	expect((await storedProject(page)).common).not.toHaveProperty('status');
	await page.goto(`/compose/project?draft=${draftId}`);
	await status.click();
	await page
		.getByRole('option', { name: 'Work in progress', exact: true })
		.click();
	await save(page);
	expect((await storedProject(page)).common).toMatchObject({
		status: 'social.craftsky.feed.defs#wip'
	});
});

test('opens a legacy pattern and persists independent entries up to the five-pattern limit', async ({
	page
}) => {
	await seed(page, { pattern: { name: 'First pattern', designer: 'Ada' } });
	await page
		.getByRole('button', { name: 'Pattern details', exact: true })
		.click();
	const panel = page.getByRole('region', {
		name: 'Pattern details',
		exact: true
	});
	const designer = panel.getByRole('textbox', {
		name: 'Designer',
		exact: true
	});
	const patternName = panel.getByRole('textbox', {
		name: 'Pattern name',
		exact: true
	});
	const addPattern = panel.getByRole('button', {
		name: 'Add another pattern',
		exact: true
	});
	const removePatterns = panel.getByRole('button', {
		name: /^Remove pattern \d+$/
	});
	await expect(designer).toHaveValue('Ada');
	await expect(patternName).toHaveValue('First pattern');
	await expect(patternName).toHaveAttribute('placeholder', '#PatternName');
	await expect(removePatterns).toHaveCount(0);
	await expect(addPattern).toHaveCSS('border-top-width', '0px');
	await expect(addPattern).toHaveCSS('box-shadow', 'none');
	await expect(addPattern.locator('svg')).toHaveCSS('fill', 'rgb(21, 53, 214)');
	await addPattern.click();
	await expect(removePatterns).toHaveCount(2);
	await expect(patternName.nth(1)).toHaveAttribute(
		'placeholder',
		'#PatternName'
	);
	await expect(patternName.nth(1)).toHaveValue('');
	for (const index of [1, 2]) {
		const entry = panel.getByRole('group', {
			name: `Pattern ${index}`,
			exact: true
		});
		const header = entry.locator('.pattern-heading');
		const heading = header.getByRole('heading', {
			name: `Pattern ${index}`,
			exact: true
		});
		await expect(heading).toHaveCSS('font-size', '20px');
		await expect(header).toHaveCSS(
			'border-top-width',
			index === 1 ? '0px' : '1px'
		);
		const remove = header.getByRole('button', {
			name: `Remove pattern ${index}`,
			exact: true
		});
		await expect(remove).toHaveCSS('color', 'rgb(184, 32, 22)');
		await expect(remove).toHaveCSS('border-top-width', '0px');
		await expect(remove).toHaveCSS('box-shadow', 'none');
		await expect(remove.locator('svg')).toHaveCSS('fill', 'rgb(184, 32, 22)');
		const titleBox = (await heading.boundingBox())!;
		const removeBox = (await remove.boundingBox())!;
		expect(removeBox.x).toBeGreaterThan(titleBox.x);
		expect(removeBox.y + removeBox.height / 2).toBeCloseTo(
			titleBox.y + titleBox.height / 2,
			0
		);
	}
	await removePatterns.last().click();
	await expect(removePatterns).toHaveCount(0);
	await expect(designer).toHaveValue('Ada');
	for (let index = 2; index <= 5; index++) {
		await addPattern.click();
		const entry = panel.getByRole('group', {
			name: `Pattern ${index}`,
			exact: true
		});
		await entry
			.getByRole('textbox', { name: 'Pattern name', exact: true })
			.fill(`Pattern name ${index}`);
		await entry
			.getByRole('textbox', { name: 'Designer', exact: true })
			.fill(`Designer ${index}`);
	}
	await expect(addPattern).toHaveCount(0);
	await panel
		.getByRole('button', { name: 'Remove pattern 2', exact: true })
		.click();
	await panel.getByRole('button', { name: 'Back to project' }).click();
	await save(page);
	const common = (await storedProject(page)).common as JsonObject;
	expect(common.pattern).toBeUndefined();
	expect(common.patterns).toEqual([
		{ name: 'First pattern', designer: 'Ada' },
		{ name: 'Pattern name 3', designer: 'Designer 3' },
		{ name: 'Pattern name 4', designer: 'Designer 4' },
		{ name: 'Pattern name 5', designer: 'Designer 5' }
	]);
});

test('publishes five ordered patterns and edits one without changing its siblings', async ({
	page
}) => {
	await page.unroute('**/api/appview/v1/posts');
	await seed(page, { status: 'social.craftsky.feed.defs#finished' });
	const names = ['#FirstPattern', 'Second', 'Third', 'Fourth', 'Fifth'];
	const panel = page.getByRole('region', {
		name: 'Pattern details',
		exact: true
	});
	const nameInput = 'Pattern name';
	const patternName = (position: number) =>
		panel
			.getByRole('group', { name: `Pattern ${position}`, exact: true })
			.getByRole('textbox', { name: nameInput, exact: true });
	const details = page.locator('.project-detail-metadata dd');
	const isWrite = (method: string, path: string) => (request: Request) =>
		request.method() === method &&
		decodeURIComponent(new URL(request.url()).pathname) === path;
	await page
		.getByRole('button', { name: 'Pattern details', exact: true })
		.click();
	await panel
		.getByRole('textbox', { name: nameInput, exact: true })
		.fill(names[0]);
	await panel
		.getByRole('textbox', { name: 'Designer', exact: true })
		.fill('Ada');
	await panel
		.getByLabel('Link', { exact: true })
		.fill('https://example.com/first-pattern');
	for (const [index, name] of names.slice(1).entries()) {
		await panel
			.getByRole('button', { name: 'Add another pattern', exact: true })
			.click();
		await patternName(index + 2).fill(name);
	}
	await expect(
		panel.getByRole('button', { name: 'Add another pattern', exact: true })
	).toHaveCount(0);
	await panel.getByRole('button', { name: 'Back to project' }).click();
	const create = page.waitForRequest(isWrite('POST', '/api/appview/v1/posts'));
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect(page).toHaveURL('/drafts');
	const savedPatterns = (await create).postDataJSON().project.common.patterns;
	expect(savedPatterns?.map((item: { name: string }) => item.name)).toEqual(
		names
	);
	expect(savedPatterns?.[0]).toMatchObject({
		name: names[0],
		designer: 'Ada',
		url: 'https://example.com/first-pattern',
		nameFacets: [
			{
				index: { byteStart: 0, byteEnd: 13 },
				features: [
					{ $type: 'app.bsky.richtext.facet#tag', tag: 'FirstPattern' }
				]
			}
		]
	});
	const postPath = '/posts/did%3Aplc%3Aalice/created-1';
	await page.goto(postPath);
	await expect(details).toHaveText([
		/#FirstPattern\s+by\s+Ada\s+Pattern link/,
		...names.slice(1)
	]);
	await expect(
		details.first().getByRole('link', { name: names[0], exact: true })
	).toHaveAttribute('href', /\/search\/tags/);
	await expect(
		details.first().getByRole('link', { name: 'Pattern link', exact: true })
	).toHaveAttribute('href', 'https://example.com/first-pattern');
	await page.goto('/compose/edit?did=did%3Aplc%3Aalice&rkey=created-1');
	await page
		.getByRole('button', { name: 'Pattern details', exact: true })
		.click();
	await expect(patternName(1)).toHaveValue(names[0]);
	await expect(patternName(2)).toHaveValue(names[1]);
	await patternName(1).fill('Temporary pattern name');
	await patternName(1).fill(names[0]);
	await patternName(3).fill('Revised third');
	await panel
		.getByRole('button', { name: 'Remove pattern 2', exact: true })
		.click();
	await panel.getByRole('button', { name: 'Back to project' }).click();
	const edit = page.waitForRequest(
		isWrite('PATCH', '/api/appview/v1/posts/did:plc:alice/created-1')
	);
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(page).toHaveURL(postPath);
	expect((await edit).postDataJSON().changes.project.common.patterns).toEqual([
		{ existingIndex: 0 },
		{ existingIndex: 2, name: 'Revised third' },
		{ existingIndex: 3 },
		{ existingIndex: 4 }
	]);
	await page.reload();
	await expect(details).toHaveText([
		/#FirstPattern\s+by\s+Ada\s+Pattern link/,
		'Revised third',
		'Fourth',
		'Fifth'
	]);
	await expect(
		details.first().getByRole('link', { name: names[0], exact: true })
	).toHaveAttribute('href', /\/search\/tags/);
});

test('counts the unchanged caption facets before budgeting a later pattern tag', async ({
	page,
	request
}) => {
	const tags = Array.from({ length: 9 }, (_, index) => `t${index}`);
	const text = `${tags.map((tag) => `#${tag}`).join(' ')} #credit  `;
	const facets = tags.map((tag, index) => ({
		index: { byteStart: index * 4, byteEnd: index * 4 + 3 },
		features: [{ $type: 'app.bsky.richtext.facet#tag', tag }]
	}));
	const create = await request.post('http://127.0.0.1:19847/v1/posts', {
		headers: {
			authorization: `Bearer ${alice.token}`,
			'idempotency-key': crypto.randomUUID()
		},
		data: {
			text,
			facets: [
				...facets,
				{
					index: {
						byteStart: text.indexOf('#credit'),
						byteEnd: text.trimEnd().length
					},
					features: [
						{
							$type: 'app.bsky.richtext.facet#link',
							uri: 'https://example.com/credit'
						}
					]
				}
			],
			langs: ['en'],
			project: {
				common: {
					craftType: 'social.craftsky.feed.defs#sewing',
					patterns: [{ name: 'First pattern' }, { name: 'Other' }]
				}
			}
		}
	});
	expect(create.status()).toBe(201);
	const { rkey } = await create.json();
	await page.goto(`/compose/edit?did=did%3Aplc%3Aalice&rkey=${rkey}`);
	await page
		.getByRole('button', { name: 'Pattern details', exact: true })
		.click();
	const panel = page.getByRole('region', {
		name: 'Pattern details',
		exact: true
	});
	await panel
		.getByRole('group', { name: 'Pattern 2', exact: true })
		.getByRole('textbox', { name: 'Pattern name', exact: true })
		.fill('Other #lace');
	await panel.getByRole('button', { name: 'Back to project' }).click();
	const status = page.getByRole('combobox', {
		name: 'Status required',
		exact: true
	});
	await expect(status).toHaveValue('');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(status).toHaveAttribute('aria-invalid', 'true');
	await status.click();
	await page.getByRole('option', { name: 'Finished', exact: true }).click();
	await expect(status).not.toHaveAttribute('aria-invalid');
	const edit = page.waitForRequest(
		(request) =>
			request.method() === 'PATCH' && request.url().endsWith('/' + rkey)
	);
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	const changes = (await edit).postDataJSON().changes;
	expect(changes.text).toBeUndefined();
	expect(changes.facets).toBeUndefined();
	expect(changes.project.common.status).toBe(
		'social.craftsky.feed.defs#finished'
	);
	expect(changes.project.common.patterns).toEqual([
		{ existingIndex: 0 },
		{
			existingIndex: 1,
			name: 'Other #lace',
			nameFacets: [
				{
					index: { byteStart: 6, byteEnd: 11 },
					features: [{ $type: 'app.bsky.richtext.facet#tag', tag: 'lace' }]
				}
			]
		}
	]);
	await expect(
		page.getByRole('link', { name: '#credit', exact: true })
	).toHaveAttribute('href', 'https://example.com/credit');
	await expect(
		page.getByRole('link', { name: '#lace', exact: true })
	).toHaveAttribute('href', /\/search\/tags/);
});

test('saves embroidery method and style as independent choices', async ({
	page
}) => {
	await seed(page, { craftType: 'social.craftsky.feed.defs#embroidery' });
	await page
		.getByRole('button', { name: 'Embroidery details', exact: true })
		.click();
	const style = page.getByRole('combobox', { name: 'Style', exact: true });
	await style.click();
	await page.getByRole('option', { name: 'Needlepoint', exact: true }).click();
	await page.getByRole('combobox', { name: 'Method', exact: true }).click();
	await page.getByRole('option', { name: 'Hand', exact: true }).click();
	await expect(style).toHaveValue('Needlepoint');
	await page.getByRole('button', { name: 'Back to project' }).click();
	await save(page);
	expect((await storedProject(page)).details).toEqual({
		$type: 'social.craftsky.project.embroidery#details',
		method: 'social.craftsky.project.embroidery.defs#hand',
		style: 'social.craftsky.project.embroidery.defs#needlepoint'
	});
});

for (const sample of [
	{
		craft: 'Crochet',
		id: 'crochet',
		existing: 'Tunisian',
		existingID: 'social.craftsky.project.crochet.defs#tunisianCrochet',
		addition: 'Standard',
		query: 'Standard crochet',
		additionID: 'social.craftsky.project.crochet.defs#standardCrochet',
		legacy: {
			techniques: [{ technique: 'TunisianCrochet' }]
		}
	},
	{
		craft: 'Felting',
		id: 'felting',
		existing: 'Needle',
		existingID: 'social.craftsky.project.felting#needle',
		addition: 'Wet',
		query: 'Wet felting',
		additionID: 'social.craftsky.project.felting#wet',
		legacy: { method: 'needle' }
	}
]) {
	test(`${sample.craft} methods reopen together and clearing keeps the older field`, async ({
		page
	}) => {
		await seed(
			page,
			{ craftType: `social.craftsky.feed.defs#${sample.id}` },
			{
				$type: `social.craftsky.project.${sample.id}#details`,
				...sample.legacy
			}
		);
		const open = () =>
			page
				.getByRole('button', { name: `${sample.craft} details`, exact: true })
				.click();
		const remove = (choice: string) =>
			page.getByRole('button', {
				name: `Remove ${choice} from ${sample.craft} methods`,
				exact: true
			});
		const finish = async () => {
			await page.getByRole('button', { name: 'Back to project' }).click();
			await save(page);
		};
		await open();
		await expect(remove(sample.existing)).toBeVisible();
		await page
			.getByRole('searchbox', { name: `${sample.craft} methods` })
			.fill(sample.query);
		await page
			.getByRole('checkbox', { name: sample.addition, exact: true })
			.click();
		await finish();
		expect((await storedProject(page)).details).toMatchObject({
			...sample.legacy,
			methods: [sample.existingID, sample.additionID]
		});
		await page.goto(`/compose/project?draft=${draftId}`);
		await open();
		await remove(sample.existing).click();
		await remove(sample.addition).click();
		await finish();
		expect((await storedProject(page)).details).toMatchObject({
			...sample.legacy,
			methods: []
		});
		await page.goto(`/compose/project?draft=${draftId}`);
		await open();
		await expect(remove(sample.existing)).toHaveCount(0);
	});
}

test('saves a woven item and its loom separately', async ({ page }) => {
	await seed(page, { craftType: 'social.craftsky.feed.defs#weaving' });
	await page
		.getByRole('button', { name: 'Weaving details', exact: true })
		.click();
	for (const [label, choice] of [
		['Project type', 'Towel'],
		['Loom type', 'Rigid heddle']
	]) {
		await page.getByRole('combobox', { name: label, exact: true }).click();
		await page.getByRole('option', { name: choice, exact: true }).click();
	}
	await page.getByRole('button', { name: 'Back to project' }).click();
	await save(page);
	expect((await storedProject(page)).details).toMatchObject({
		$type: 'social.craftsky.project.weaving#details',
		projectType: 'social.craftsky.project.defs#homeGoods',
		projectSubtype: 'social.craftsky.project.weaving#towel',
		loomType: 'social.craftsky.project.weaving#rigidHeddle'
	});
});

test('searchable project selections retain unknown values and existing craft metadata while excluding the new primary craft', async ({
	page
}) => {
	const knitting = 'social.craftsky.feed.defs#knitting',
		crochet = 'social.craftsky.feed.defs#crochet';
	const retained = {
		craftType: knitting,
		existingIndex: 3,
		details: { futureMetadata: 'kept' }
	};
	await seed(page, {
		additionalCrafts: [
			retained,
			{ craftType: 'social.craftsky.feed.defs#future-craft', existingIndex: 8 }
		],
		colors: ['future-colour'],
		designTags: ['social.craftsky.project.defs#future-design']
	});
	await page
		.getByRole('button', { name: 'Sewing details', exact: true })
		.click();
	const additional = page.getByRole('searchbox', {
		name: 'Additional crafts',
		exact: true
	});
	const type = page.getByRole('combobox', {
		name: 'Project type',
		exact: true
	});
	expect((await additional.boundingBox())!.y).toBeGreaterThan(
		(await type.boundingBox())!.y
	);
	await additional.fill('Crochet');
	await page.getByRole('checkbox', { name: 'Crochet', exact: true }).click();
	await page
		.getByRole('button', { name: 'Back to project', exact: true })
		.click();
	await expect(
		page.getByRole('button', { name: 'Materials and style', exact: true })
	).toHaveCount(0);
	await save(page);
	await expect
		.poll(() => storedProject(page))
		.toMatchObject({
			common: {
				additionalCrafts: [
					retained,
					{
						craftType: 'social.craftsky.feed.defs#future-craft',
						existingIndex: 8
					},
					{ craftType: crochet }
				],
				colors: ['future-colour'],
				designTags: ['social.craftsky.project.defs#future-design']
			}
		});
	await page.goto(`/compose/project?draft=${draftId}`);
	await expect(
		page.getByRole('textbox', { name: 'Project title', exact: true })
	).toHaveValue('Native project fields');
	await page
		.getByRole('combobox', { name: 'Primary craft', exact: false })
		.click();
	await page.getByRole('option', { name: 'Knitting', exact: true }).click();
	await save(page);
	await expect
		.poll(() => storedProject(page))
		.toMatchObject({
			common: {
				craftType: knitting,
				additionalCrafts: [
					{
						craftType: 'social.craftsky.feed.defs#future-craft',
						existingIndex: 8
					},
					{ craftType: crochet }
				]
			}
		});
});

test('primary craft choices follow the same preference and catalogue order as Home', async ({
	page
}) => {
	await page.route('**/api/appview/v1/profiles/me', async (route) => {
		const response = await route.fetch();
		await route.fulfill({
			json: { ...(await response.json()), crafts: ['weaving', 'sewing'] }
		});
	});
	await page.goto('/feed');
	const crafts = page
		.getByRole('navigation', { name: 'Home feeds' })
		.locator('a[href*="?craft="]');
	await expect(crafts.first()).toHaveText('Weaving');
	const homeOrder = (await crafts.allTextContents()).map((name) => name.trim());
	expect(homeOrder).toHaveLength(11);
	expect(homeOrder.slice(0, 2)).toEqual(['Weaving', 'Sewing']);
	await seed(page, {});
	await page.getByRole('combobox', { name: 'Primary craft required' }).click();
	const choices = page
		.getByRole('listbox', { name: 'Primary craft options' })
		.getByRole('option');
	await expect(choices).toHaveCount(homeOrder.length);
	expect(
		await choices.evaluateAll((options) =>
			options.map((option) => option.getAttribute('aria-label'))
		)
	).toEqual(homeOrder);
});

test('composer project types show nested options and match their aliases without selecting the parent', async ({
	page
}) => {
	await seed(page, {});
	const craft = page.getByRole('button', {
		name: 'Sewing details',
		exact: true
	});
	await expect(craft).toHaveAccessibleDescription(
		'Add other crafts and describe what you made'
	);
	await craft.click();
	const details = page.getByRole('region', {
		name: 'Sewing details',
		exact: true
	});
	await expect(details.getByRole('heading')).toHaveText('Sewing details');
	const type = details.getByRole('combobox', {
		name: 'Project type',
		exact: true
	});
	await expect(
		details.getByRole('combobox', { name: 'Project subtype', exact: true })
	).toHaveCount(0);
	await type.click();
	const typeChoices = page.getByRole('listbox', {
		name: 'Project type options'
	});
	const options = typeChoices.getByRole('option');
	await expect(options.first()).toHaveAccessibleName('Accessory');
	await expect(options.last()).toHaveAccessibleName('Other');
	await expect(options.first()).toHaveCSS('margin-block-start', '0px');
	await expect(
		typeChoices.getByRole('option', { name: 'Garment', exact: true })
	).toHaveCSS('margin-block-start', '6px');
	const costume = typeChoices.getByRole('option', {
		name: 'Costume',
		exact: true
	});
	await expect(costume).toHaveCSS('font-weight', '700');
	await type.fill('costume');
	await expect(costume).toHaveCSS('font-weight', '700');
	await type.fill('home decor');
	const homeGoods = page.getByRole('option', {
		name: 'Home goods',
		exact: true
	});
	await expect(homeGoods.locator('small')).toHaveText('Home decor');
	await type.fill('garment');
	await expect(options.first()).toHaveAccessibleName('Garment');
	const [parent, ...subtypeNames] = await options.evaluateAll((rows) =>
		rows.map((option) => option.getAttribute('aria-label') ?? '')
	);
	expect(parent).toBe('Garment');
	expect(subtypeNames.length).toBeGreaterThan(5);
	expect(subtypeNames).toEqual([...subtypeNames].sort());
	await type.fill('pants');
	const trousers = page.getByRole('option', { name: 'Trousers', exact: true });
	await expect(trousers.locator('small')).toHaveText('Pants');
	await type.fill('overalls');
	const dungarees = page.getByRole('option', {
		name: 'Dungarees',
		exact: true
	});
	await expect(options).toHaveCount(2);
	await expect(options.first()).toHaveAccessibleName('Garment');
	await expect(dungarees.locator('small')).toHaveText('Overalls');
	await expect(dungarees).toHaveAccessibleDescription(
		'Garment · Dungarees Overalls'
	);
	const labelOffsets = await options.evaluateAll((rows) =>
		rows.map((row) => parseFloat(getComputedStyle(row).paddingInlineStart))
	);
	expect(labelOffsets[1]).toBeGreaterThan(labelOffsets[0]);
	await expect(options.first()).toHaveCSS('font-weight', '700');
	await expect(dungarees).toHaveCSS('font-weight', '400');
	await expect(dungarees).toHaveCSS('margin-block-start', '0px');
	await type.press('Enter');
	await expect(type).toHaveValue('Garment · Dungarees');
	const back = page.getByRole('button', { name: 'Back to project' });
	await back.click();
	await expect(craft).toHaveAccessibleDescription('Garment · Dungarees');
	await craft.click();
	await type.fill('garment');
	await options.first().click();
	await expect(type).toHaveValue('Garment');
	await back.click();
	await save(page);
	const { details: savedDetails } = await storedProject(page);
	expect(savedDetails).toMatchObject({
		projectType: 'social.craftsky.project.defs#garment'
	});
	expect((savedDetails as JsonObject).projectSubtype).toBeUndefined();
});

test('custom item names are editable only under Other and saved names under named types are retained', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	const legacy = { text: 'Cotton fabric' };
	await seed(
		page,
		{ craftType: 'social.craftsky.feed.defs#knitting', materials: [legacy] },
		{
			$type: 'social.craftsky.project.knitting#details',
			projectType: 'social.craftsky.project.defs#homeGoods',
			projectSubtype: 'Lampshade',
			needleSizesMm: ['4.0mm']
		}
	);
	const craft = page.getByRole('button', {
		name: 'Knitting details',
		exact: true
	});
	const open = () => craft.click();
	const back = () =>
		page.getByRole('button', { name: 'Back to project', exact: true }).click();
	const type = page.getByRole('combobox', {
		name: 'Project type',
		exact: true
	});
	const custom = page.getByRole('textbox', {
		name: 'What did you make?',
		exact: true
	});
	await expect(craft).toHaveAccessibleDescription('Home goods · Lampshade');
	await open();
	const details = page.getByRole('region', {
		name: 'Knitting details',
		exact: true
	});
	await expect(details.getByRole('heading')).toHaveText('Knitting details');
	const detailOrder = [
		type,
		page.getByRole('searchbox', { name: 'Knitting methods' }),
		page.getByRole('searchbox', { name: 'Additional crafts' })
	];
	const positions = await Promise.all(
		detailOrder.map((field) =>
			field.evaluate((element) => element.getBoundingClientRect().top)
		)
	);
	expect(positions).toEqual([...positions].sort((a, b) => a - b));
	await expect(type).toHaveValue('Home goods · Lampshade');
	await expect(custom).toHaveCount(0);
	await type.fill('cushion');
	const cushion = page.getByRole('option', {
		name: 'Cushion or cover',
		exact: true
	});
	await expect(cushion).toHaveAccessibleDescription(
		'Home goods · Cushion or cover Cushion cover · Pillow · Pillow cover'
	);
	await type.fill('Add your own');
	await expect(
		page.getByRole('option', { name: 'Add your own', exact: true })
	).toHaveCount(0);
	await type.press('Escape');
	await back();
	await page
		.getByRole('textbox', { name: 'Project title', exact: true })
		.fill('Revised project title');
	await save(page);
	expect((await storedProject(page)).details).toMatchObject({
		projectType: 'social.craftsky.project.defs#homeGoods',
		projectSubtype: 'Lampshade',
		needleSizesMm: ['4.0mm']
	});
	expect((await storedProject(page)).common).toMatchObject({
		materials: [legacy]
	});
	await page.goto(`/compose/project?draft=${draftId}`);
	await open();
	await expect(type).toHaveValue('Home goods · Lampshade');
	await type.fill('Costume');
	await page.getByRole('option', { name: 'Costume', exact: true }).click();
	await expect(custom).toHaveCount(0);
	await expect(type).toHaveValue('Costume');
	await type.fill('Other');
	await page.getByRole('option', { name: 'Other', exact: true }).click();
	await expect(custom).toHaveValue('');
	await expect(type).toHaveValue('Other');
	await custom.fill('  Textile sculpture  ');
	await back();
	await expect(craft).toHaveAccessibleDescription('Other · Textile sculpture');
	await save(page);
	expect((await storedProject(page)).details).toMatchObject({
		projectType: 'social.craftsky.project.defs#other',
		projectSubtype: 'Textile sculpture'
	});
	await page.goto(`/compose/project?draft=${draftId}`);
	await open();
	await expect(custom).toHaveValue('Textile sculpture');
	await type.fill('Home goods');
	await page.getByRole('option', { name: 'Home goods', exact: true }).click();
	await expect(custom).toHaveCount(0);
	await expect(type).toHaveValue('Home goods');
});
