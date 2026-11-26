import { mockRevenueCat } from './revenuecat-fixture';
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
	await mockRevenueCat(page);
	await page.route('**/api/appview/v1/billing/account', (route) =>
		route.fulfill({
			json: {
				billingAccountId: 'fixture',
				revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
				requestedGeneration: 1,
				reconciledGeneration: 1,
				reconciledAt: new Date().toISOString(),
				reconciliationStale: false,
				webCheckoutAvailable: true,
				webCheckoutPending: false,
				subscriptions: [],
				licenses: [],
				promotionalGrants: []
			}
		})
	);
	await page.route('**/api/appview/v1/billing/web-paywall', (route) =>
		route.fulfill({
			json: {
				apiKey: 'strp_sb_fixture',
				appUserId: '20000000-0000-4000-8000-000000000001',
				offering: route.request().postDataJSON().tier
			}
		})
	);

	await page.route('**/api/service-status', (route) =>
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
		name: 'New folder',
		exact: true
	});
	await expect(create).toBeVisible();
	await expect(create.locator('[data-plus-badge]')).toBeVisible();
	await create.click();
	const modal = page.getByRole('dialog', {
		name: 'CraftSky Plus',
		exact: true
	});
	await expect(
		modal.getByRole('heading', { name: 'RevenueCat plus paywall' })
	).toBeVisible();
	await expect(page).toHaveURL(/\/saved$/);
	await page.keyboard.press('Escape');
	await expect(modal).not.toBeVisible();
	await expect(create).toBeFocused();

	expect(writes).toBe(0);
});

test('free settings feature opens the Plus paywall without navigating', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'free');
	await page.goto('/settings');
	for (const name of ['Customisation', 'Growth']) {
		const feature = page.getByRole('link', { name, exact: true });
		await expect(feature.locator('[data-plus-badge] svg')).toBeVisible();
		await expect(feature.locator('[data-plus-badge]')).not.toHaveText('+');
	}
	await page.getByRole('link', { name: 'Customisation', exact: true }).click();
	const modal = page.getByRole('dialog', {
		name: 'CraftSky Plus',
		exact: true
	});
	await expect(modal).toBeVisible();
	await expect(page).toHaveURL(/\/settings$/);
	await page.keyboard.press('Escape');
	await expect(modal).not.toBeVisible();
	await page.getByRole('link', { name: 'Growth', exact: true }).click();
	await expect(
		page.getByRole('dialog', { name: 'CraftSky Plus', exact: true })
	).toBeVisible();
});

for (const section of ['customisation', 'growth'] as const)
	test(`purchasing from locked ${section} unlocks its current page without navigation`, async ({
		page
	}) => {
		test.skip(!enabled);
		await access(page, 'free');
		let tier: 'free' | 'plus' = 'free';
		await page.route('**/api/appview/v1/subscriptions/access', (route) =>
			route.fulfill({
				json: {
					did: account.did,
					effectiveTier: tier,
					givesAccess: tier !== 'free'
				}
			})
		);
		await page.goto(`/settings/${section}`);
		await page
			.getByRole('button', { name: 'Learn about subscriptions', exact: true })
			.click();
		const paywall = page.getByRole('dialog', {
			name: 'CraftSky Plus',
			exact: true
		});
		await paywall
			.getByRole('button', { name: 'Continue to checkout', exact: true })
			.click();
		tier = 'plus';
		await page
			.getByRole('button', { name: 'Complete fixture payment', exact: true })
			.click();
		await expect(paywall).not.toBeVisible();
		await expect(page).toHaveURL(new RegExp(`/settings/${section}$`));
		await expect(
			page.getByRole('button', {
				name: 'Learn about subscriptions',
				exact: true
			})
		).toHaveCount(0);
		if (section === 'customisation')
			await expect(page.locator('.customisation-form')).toBeVisible();
		else
			await expect(
				page.getByRole('heading', { name: 'Trend', exact: true })
			).toBeVisible();
	});

test('foreground access changes unlock business details and hide them again after confirmed access loss', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'free');
	let tier: 'free' | 'business' = 'free';
	await page.route('**/api/appview/v1/subscriptions/access', (route) =>
		route.fulfill({
			json: {
				did: account.did,
				effectiveTier: tier,
				givesAccess: tier !== 'free'
			}
		})
	);
	await page.goto('/settings/business');
	await expect(
		page.getByRole('searchbox', { name: 'Business types', exact: true })
	).toHaveCount(0);
	tier = 'business';
	await page.evaluate(() =>
		document.dispatchEvent(new Event('visibilitychange'))
	);
	await expect(
		page.getByRole('searchbox', { name: 'Business types', exact: true })
	).toBeVisible();
	tier = 'free';
	await page.evaluate(() =>
		document.dispatchEvent(new Event('visibilitychange'))
	);
	await expect(
		page.getByRole('searchbox', { name: 'Business types', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('link', { name: 'View subscription access', exact: true })
	).toBeVisible();
});

for (const updated of ['free', 'plus'] as const)
	test(`period boundary refresh respects server-confirmed ${updated} access`, async ({
		page
	}) => {
		test.skip(!enabled);
		await page.clock.install({ time: new Date('2026-11-16T10:00:00Z') });
		await access(page, 'plus');
		let tier: 'free' | 'plus' = 'plus',
			reads = 0;
		await page.route('**/api/appview/v1/subscriptions/access', (route) => {
			reads++;
			return route.fulfill({
				json: {
					did: account.did,
					effectiveTier: tier,
					givesAccess: tier !== 'free',
					accessEndsAt: '2026-11-16T10:00:20Z'
				}
			});
		});
		await page.goto('/compose');
		const when = page.getByRole('button', { name: 'When', exact: true });
		await expect(when.locator('[data-plus-badge]')).toHaveCount(0);
		const initialReads = reads;
		tier = updated;
		await page.clock.fastForward(30_000);
		await expect.poll(() => reads).toBeGreaterThan(initialReads);
		if (updated === 'free')
			await expect(when.locator('[data-plus-badge]')).toBeVisible();
		else await expect(when.locator('[data-plus-badge]')).toHaveCount(0);
	});

for (const composer of ['post', 'project'] as const)
	test(`free ${composer} scheduling keeps a small Plus overlay and opens Plus`, async ({
		page
	}) => {
		test.skip(!enabled);
		await access(page, 'free');
		await page.goto(composer === 'post' ? '/compose' : '/compose/project');
		const when = page.getByRole('button', { name: 'When', exact: true });
		const mark = when.locator('[data-plus-badge]');
		await expect(mark.locator('svg')).toBeVisible();
		// Catch global pill-badge and composer SVG styles leaking into the overlay.
		const geometry = await mark.evaluate((element) => {
			const bounds = element.getBoundingClientRect();
			const svg = element.querySelector('svg')!.getBoundingClientRect();
			const button = element.closest('button')!.getBoundingClientRect();
			const clock = element
				.parentElement!.querySelector('svg')!
				.getBoundingClientRect();
			const css = getComputedStyle(element);
			return {
				width: bounds.width,
				height: bounds.height,
				svgWidth: svg.width,
				svgHeight: svg.height,
				border: css.borderWidth,
				padding: css.padding,
				margin: css.margin,
				background: css.backgroundColor,
				inside:
					bounds.left >= button.left &&
					bounds.right <= button.right &&
					bounds.top >= button.top &&
					bounds.bottom <= button.bottom,
				clockVisible:
					bounds.left >= clock.left + clock.width / 2 &&
					bounds.top >= clock.top + clock.height / 2
			};
		});
		expect(geometry).toEqual({
			width: 12,
			height: 12,
			svgWidth: 12,
			svgHeight: 12,
			border: '0px',
			padding: '0px',
			margin: '0px',
			background: 'rgba(0, 0, 0, 0)',
			inside: true,
			clockVisible: true
		});
		await when.click();
		const paywall = page.getByRole('dialog', {
			name: 'CraftSky Plus',
			exact: true
		});
		await expect(paywall).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(paywall).not.toBeVisible();
		await expect(when).toBeFocused();
	});

for (const tier of ['plus', 'business'] as const)
	test(`${tier} opens folder creation without a Plus badge or upgrade prompt`, async ({
		page
	}) => {
		test.skip(!enabled);
		await access(page, tier);
		await page.goto('/saved');
		const create = page.getByRole('button', {
			name: 'New folder',
			exact: true
		});
		await expect(create).toBeVisible();
		await expect(create.locator('[data-plus-badge]')).toHaveCount(0);
		await create.click();
		await expect(
			page.getByRole('dialog', { name: 'New folder', exact: true })
		).toBeVisible();
		await expect(
			page.getByRole('dialog', { name: 'CraftSky Plus' })
		).toHaveCount(0);
		const dialog = page.getByRole('dialog', {
			name: 'New folder',
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
	await page.getByRole('button', { name: 'New folder', exact: true }).click();
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
		name: 'CraftSky Plus',
		exact: true
	});
	await expect(pinPrompt).toBeVisible();
	await page.keyboard.press('Escape');
	await page.getByRole('button', { name: 'Save', exact: true }).first().click();
	const save = page.getByRole('dialog', { name: 'Save post', exact: true });
	await expect(save).toBeVisible();
	const folder = save.getByRole('button', { name: 'New folder', exact: true });
	await folder.click();
	const prompt = page.getByRole('dialog', {
		name: 'CraftSky Plus',
		exact: true
	});
	await expect(prompt).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(prompt).not.toBeVisible();
	await expect(save).toBeVisible();
	await expect(folder).toBeFocused();
});

test('free saved post offers a badged Move action', async ({ page }) => {
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
	const move = page.getByRole('button', { name: 'Move', exact: true }).first();
	await expect(move.locator('[data-plus-badge]')).toBeVisible();
	await move.click();
	await expect(
		page.getByRole('dialog', { name: 'CraftSky Plus', exact: true })
	).toBeVisible();
});

test('a mismatched access response during feature preparation cannot open a paywall', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'free');
	await page.goto('/saved');
	const trigger = page.getByRole('button', {
		name: 'New folder',
		exact: true
	});
	await expect(trigger).toBeVisible();
	await page.route('**/api/appview/v1/subscriptions/access', (route) =>
		route.fulfill({
			json: { did: 'did:plc:other', effectiveTier: 'free', givesAccess: false }
		})
	);
	let configurations = 0;
	page.on('request', (request) => {
		if (request.url().endsWith('/billing/web-paywall')) configurations++;
	});
	await trigger.click();
	const dialog = page.getByRole('dialog', {
		name: 'Subscription details',
		exact: true
	});
	await expect(dialog.getByRole('alert')).toContainText(
		'Subscription access is unavailable'
	);
	expect(configurations).toBe(0);
	await dialog.getByRole('button', { name: 'Close', exact: true }).click();
	await expect(dialog).toHaveCount(0);
	await expect(trigger).toBeFocused();
});

test('freshly recognized paid or promotional access bypasses the feature paywall', async ({
	page
}) => {
	test.skip(!enabled);
	await access(page, 'free');
	await page.goto('/saved');
	const trigger = page.getByRole('button', {
		name: 'New folder',
		exact: true
	});
	await expect(trigger.locator('[data-plus-badge]')).toBeVisible();
	await page.route('**/api/appview/v1/subscriptions/access', (route) =>
		route.fulfill({
			json: { did: account.did, effectiveTier: 'plus', givesAccess: true }
		})
	);
	let configurations = 0;
	page.on('request', (request) => {
		if (request.url().endsWith('/billing/web-paywall')) configurations++;
	});
	await trigger.click();
	await expect(trigger.locator('[data-plus-badge]')).toHaveCount(0);
	expect(configurations).toBe(0);
	await expect(
		page.getByRole('dialog', { name: 'CraftSky Plus', exact: true })
	).toHaveCount(0);
	const folderDialog = page.getByRole('dialog', {
		name: 'New folder',
		exact: true
	});
	// The initial page load can discover the updated access before the first click.
	if (!(await folderDialog.isVisible())) await trigger.click();
	await expect(folderDialog).toBeVisible();
});
