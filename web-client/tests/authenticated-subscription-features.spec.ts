import { expect, test, type Page } from '@playwright/test';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const enabled = process.env.WEB_CLIENT_SUBSCRIPTIONS_ENABLED !== 'false';

test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await context.addCookies([
		{
			name: `craftsky_account_${account.id}`,
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

async function access(
	page: Page,
	tier: 'free' | 'plus' | 'business' | 'unknown'
) {
	await page.route('**/api/appview/v1/subscriptions/access', (route) =>
		tier === 'unknown'
			? route.fulfill({
					status: 503,
					json: { error: 'temporarily_unavailable', message: 'Unavailable' }
				})
			: route.fulfill({
					json: {
						did: account.did,
						effectiveTier: tier,
						givesAccess: tier !== 'free'
					}
				})
	);
	await page.route('**/api/appview/v1/saved-post-folders', (route) =>
		route.request().method() === 'GET'
			? route.fulfill(
					tier === 'free' || tier === 'unknown'
						? {
								status: 403,
								json: { error: 'plus_required', message: 'Plus required' }
							}
						: { json: { items: [], cursor: null } }
				)
			: route.fulfill({
					status: 201,
					json: { id: 'new-folder', name: 'Knitting', savedCount: 0 }
				})
	);
	await page.route('**/api/appview/v1/saved-posts?*', (route) =>
		route.fulfill({ json: { items: [], cursor: null } })
	);
}

test('free saved posts keeps the badged folder action and opens a cancellable upgrade modal', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'free');
	let writes = 0;
	page.on('request', (request) => {
		if (
			request.url().endsWith('/saved-post-folders') &&
			request.method() === 'POST'
		)
			writes++;
	});
	await page.goto('/saved');
	const create = page.getByRole('button', {
		name: 'Create folder',
		exact: true
	});
	await expect(create).toBeVisible();
	await expect(create.locator('[data-plus-badge]')).toBeVisible();
	await create.click();
	const modal = page.getByRole('dialog', {
		name: 'Folders requires Plus',
		exact: true
	});
	await expect(modal).toContainText(
		'Subscribe to Plus or Business to use Folders.'
	);
	await expect(page).toHaveURL(/\/saved$/);
	await page.keyboard.press('Escape');
	await expect(modal).not.toBeVisible();
	await expect(create).toBeFocused();
	await create.click();
	await modal
		.getByRole('button', { name: 'Learn about subscriptions' })
		.click();
	await expect(page).toHaveURL(/\/settings\/subscriptions$/);
	expect(writes).toBe(0);
});

test('free settings feature opens the shared modal before navigating', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'free');
	await page.goto('/settings');
	await page.getByRole('link', { name: 'Customisation', exact: true }).click();
	const modal = page.getByRole('dialog', {
		name: 'Customisation requires Plus',
		exact: true
	});
	await expect(modal).toBeVisible();
	await expect(page).toHaveURL(/\/settings$/);
	await modal.getByRole('button', { name: 'Cancel', exact: true }).click();
	await expect(modal).not.toBeVisible();
	await page.getByRole('link', { name: 'Growth', exact: true }).click();
	await expect(
		page.getByRole('dialog', { name: 'Growth requires Plus', exact: true })
	).toBeVisible();
});

for (const tier of ['plus', 'business'] as const)
	test(`${tier} opens folder creation without a Plus badge or upgrade prompt`, async ({
		page
	}) => {
		test.skip(!enabled);
		await access(page, tier);
		await page.goto('/saved');
		const create = page.getByRole('button', {
			name: 'Create folder',
			exact: true
		});
		await expect(create).toBeVisible();
		await expect(create.locator('[data-plus-badge]')).toHaveCount(0);
		await create.click();
		await expect(
			page.getByRole('dialog', { name: 'Create folder', exact: true })
		).toBeVisible();
		await expect(
			page.getByRole('dialog', { name: 'Folders requires Plus' })
		).toHaveCount(0);
		const dialog = page.getByRole('dialog', {
			name: 'Create folder',
			exact: true
		});
		await dialog.getByRole('textbox', { name: 'Folder name' }).fill('Knitting');
		const write = page.waitForRequest(
			(request) =>
				request.url().endsWith('/saved-post-folders') &&
				request.method() === 'POST'
		);
		await dialog
			.getByRole('button', { name: 'Create folder', exact: true })
			.click();
		expect((await write).postDataJSON()).toEqual({ name: 'Knitting' });
		await expect(dialog).not.toBeVisible();
	});

test('unknown access prompts to retry rather than advertising an upgrade', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'unknown');
	await page.goto('/saved');
	await page
		.getByRole('button', { name: 'Create folder', exact: true })
		.click();
	const modal = page.getByRole('dialog', {
		name: 'Subscription access unavailable',
		exact: true
	});
	await expect(modal).toBeVisible();
	await expect(
		modal.getByRole('button', { name: 'Learn about subscriptions' })
	).toHaveCount(0);
});

test('disabled release flag keeps the folder action and shows coming soon', async ({
	page
}) => {
	test.skip(enabled);
	await access(page, 'free');
	await page.goto('/saved');
	await page
		.getByRole('button', { name: 'Folders is coming soon', exact: true })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Folders is coming soon', exact: true })
	).toBeVisible();
});

test('free post actions prompt for Pin and New folder without losing the save dialog', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'free');
	await page.goto('/posts/did%3Aplc%3Aalice/first');
	await page.getByLabel('More post actions', { exact: true }).first().click();
	const pin = page.getByRole('button', { name: 'Pin', exact: true }).first();
	await expect(pin.locator('[data-plus-badge]')).toBeVisible();
	await pin.click();
	const pinPrompt = page.getByRole('dialog', {
		name: 'Pin requires Plus',
		exact: true
	});
	await expect(pinPrompt).toBeVisible();
	await pinPrompt.getByRole('button', { name: 'Cancel', exact: true }).click();
	await page.getByRole('button', { name: 'Save', exact: true }).first().click();
	const save = page.getByRole('dialog', { name: 'Save post', exact: true });
	await expect(save).toBeVisible();
	const folder = save.getByRole('button', { name: 'New folder', exact: true });
	await folder.click();
	const prompt = page.getByRole('dialog', {
		name: 'Folders requires Plus',
		exact: true
	});
	await expect(prompt).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(prompt).not.toBeVisible();
	await expect(save).toBeVisible();
	await expect(folder).toBeFocused();
});

test('free saved post offers a badged Move to folder action', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'free');
	const response = await page.request.get(
		'/api/appview/v1/posts/did%3Aplc%3Aalice/first'
	);
	const post = await response.json();
	await page.route('**/api/appview/v1/saved-posts?*', (route) =>
		route.fulfill({
			json: {
				items: [
					{
						post: { ...post, viewerHasSaved: true },
						savedAt: '2026-11-01T12:00:00Z'
					}
				],
				cursor: null
			}
		})
	);
	await page.goto('/saved');
	await page.getByLabel('Saved post actions', { exact: true }).first().click();
	const move = page
		.getByRole('button', { name: 'Move to folder', exact: true })
		.first();
	await expect(move.locator('[data-plus-badge]')).toBeVisible();
	await move.click();
	await expect(
		page.getByRole('dialog', { name: 'Folders requires Plus', exact: true })
	).toBeVisible();
});
