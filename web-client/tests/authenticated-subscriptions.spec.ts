import { mockRevenueCat } from './revenuecat-fixture';
import { expect, test, type Page } from '@playwright/test';
import type {
	BillingPromotionalGrant,
	BillingSubscription
} from '../src/lib/api/types.ts';

const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const privateId = '20000000-0000-4000-8000-000000000001';
const appview = '/api/appview/v1';
const paywall = {
	apiKey: 'strp_sb_fixture',
	appUserId: privateId,
	offering: 'plus'
};

test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await mockRevenueCat(page);
	await page.route(`**${appview}/billing/web-paywall`, (route) =>
		route.fulfill({
			json: { ...paywall, offering: route.request().postDataJSON().tier }
		})
	);
	await page.route(`**${appview}/billing/web-checkout/cancel`, (route) =>
		route.fulfill({ json: { status: 'canceled' } })
	);
	await page.route('https://status.craftsky.social/app.json', (route) =>
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
});

async function billingFixture(
	page: Page,
	subscriptions: BillingSubscription[] = [],
	available = true,
	promotion: {
		grants: BillingPromotionalGrant[];
		effectiveTier: 'free' | 'plus' | 'business';
	} = { grants: [], effectiveTier: 'free' }
) {
	await page.route(`**${appview}/billing/web-prices`, (route) =>
		route.fulfill({
			json: {
				items: [
					{
						tier: 'plus',
						amount: 249,
						currency: 'gbp',
						taxBehavior: 'inclusive'
					},
					{
						tier: 'business',
						amount: 899,
						currency: 'gbp',
						taxBehavior: 'inclusive'
					}
				]
			}
		})
	);
	await page.route(`**${appview}/subscriptions/access`, (route) =>
		route.fulfill({
			json: {
				did: alice.did,
				effectiveTier: promotion.effectiveTier,
				givesAccess: promotion.effectiveTier !== 'free'
			}
		})
	);
	await page.route(`**${appview}/billing/account`, (route) =>
		route.fulfill({
			json: {
				billingAccountId: 'account',
				revenueCatAppUserId: privateId,
				requestedGeneration: 1,
				reconciledGeneration: 1,
				reconciliationStale: false,
				reconciledAt: new Date().toISOString(),
				webCheckoutAvailable: available,
				promotionalGrants: promotion.grants,
				subscriptions,
				licenses: []
			}
		})
	);
}

const grant: BillingPromotionalGrant = {
	id: '50000000-0000-4000-8000-000000000001',
	tier: 'plus',
	givesAccess: true,
	expiresAt: '2032-05-20T12:00:00Z'
};

for (const tier of ['plus', 'business'] as const)
	test(`complimentary Plus permits choosing ${tier} before expiry`, async ({
		page
	}) => {
		await billingFixture(page, [], true, {
			grants: [grant],
			effectiveTier: 'plus'
		});
		const requests: unknown[] = [];
		await page.route(`**${appview}/billing/web-paywall`, (route) => {
			requests.push(route.request().postDataJSON());
			return route.fulfill({
				json: { ...paywall, offering: route.request().postDataJSON().tier }
			});
		});
		await page.route('https://checkout.stripe.com/**', (route) =>
			route.fulfill({
				contentType: 'text/html',
				body: '<h1>Review subscription</h1>'
			})
		);
		await page.goto('/settings/subscriptions');
		await expect(
			page.getByRole('heading', { name: 'Current access: Plus' })
		).toBeVisible();
		const complimentary = page.getByRole('region', {
			name: 'Complimentary CraftSky Plus'
		});
		await expect(complimentary).toContainText('Plus until');
		await expect(complimentary).toContainText('no recurring charge');
		await expect(
			page.getByText('Subscribing starts paid billing immediately.', {
				exact: false
			})
		).toBeVisible();
		await expect(
			page.getByText(
				'Remaining complimentary time is not added to your paid subscription.',
				{ exact: false }
			)
		).toBeVisible();
		await expect(
			page.getByRole('region', { name: 'Current subscription' })
		).toHaveCount(0);
		await expect(
			page.getByRole('button', { name: 'Manage in Stripe', exact: true })
		).toHaveCount(0);
		await expect(
			page.getByRole('button', { name: 'Subscribe to Plus', exact: true })
		).toBeEnabled();
		await expect(
			page.getByRole('button', { name: 'Subscribe to Business', exact: true })
		).toBeEnabled();
		await page
			.getByRole('button', {
				name: `Subscribe to ${tier === 'plus' ? 'Plus' : 'Business'}`,
				exact: true
			})
			.click();
		await expect(
			page.getByRole('dialog', { name: /CraftSky (Plus|Business)$/ })
		).toBeVisible();
		await expect(
			page.getByRole('heading', {
				name: /^RevenueCat (plus|business) paywall$/
			})
		).toBeVisible();
		await expect(page).toHaveURL(/\/settings\/subscriptions(?:\?.*)?$/);
		expect(requests).toEqual([{ tier }]);
	});

test('paid billing stays authoritative when complimentary access overlaps', async ({
	page
}) => {
	await billingFixture(
		page,
		[
			{
				id: 'paid-business',
				productId: 'business-monthly',
				tier: 'business',
				store: 'stripe',
				status: 'active',
				givesAccess: true,
				pendingPayment: false,
				anomaly: 'none'
			}
		],
		true,
		{ grants: [grant], effectiveTier: 'business' }
	);
	await page.goto('/settings/subscriptions');
	await expect(
		page.getByRole('heading', { name: 'Current access: Business' })
	).toBeVisible();
	await expect(
		page.getByRole('region', { name: 'Complimentary CraftSky Plus' })
	).toBeVisible();
	await expect(
		page.getByRole('region', { name: 'Current subscription' })
	).toContainText('CraftSky Business');
	await expect(
		page.getByRole('button', { name: 'Manage in Stripe', exact: true })
	).toBeVisible();
	await expect(
		page.getByRole('button', {
			name: /^(View (Plus|Business) details|Subscribe to (Plus|Business))$/
		})
	).toHaveCount(0);
});

test('ended complimentary access returns to Free without a billing obligation', async ({
	page
}) => {
	await billingFixture(page, [], true, {
		grants: [
			{ ...grant, givesAccess: false, expiresAt: '2020-05-20T12:00:00Z' }
		],
		effectiveTier: 'free'
	});
	await page.goto('/settings/subscriptions');
	await expect(
		page.getByRole('heading', { name: 'Current access: Free' })
	).toBeVisible();
	await expect(
		page.getByRole('region', { name: 'Complimentary CraftSky Plus' })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'View Plus details', exact: true })
	).toBeEnabled();
	await expect(
		page.getByRole('button', { name: 'Manage in Stripe', exact: true })
	).toHaveCount(0);
});

test('complimentary Business displays separately and keeps paid choices available', async ({
	page
}) => {
	await billingFixture(page, [], true, {
		grants: [{ ...grant, tier: 'business', expiresAt: null }],
		effectiveTier: 'business'
	});
	await page.goto('/settings/subscriptions');
	await expect(
		page.getByRole('heading', { name: 'Current access: Business' })
	).toBeVisible();
	const complimentary = page.getByRole('region', {
		name: 'Complimentary CraftSky Business'
	});
	await expect(complimentary).toBeVisible();
	await expect(complimentary).not.toContainText('Not available');
	await expect(complimentary).toContainText('no expiry date');
	await expect(
		page.getByRole('button', { name: 'Subscribe to Plus', exact: true })
	).toBeEnabled();
	await expect(
		page.getByRole('button', { name: 'Subscribe to Business', exact: true })
	).toBeEnabled();
});

for (const tier of ['plus', 'business'])
	test(`opens the identified RevenueCat paywall for ${tier} without accepting payment as access`, async ({
		page
	}) => {
		await billingFixture(page);
		const requests: { body: unknown; account: string | undefined }[] = [];
		await page.route(`**${appview}/billing/web-paywall`, async (route) => {
			requests.push({
				body: route.request().postDataJSON(),
				account: route.request().headers()['x-craftsky-account-id']
			});
			await route.fulfill({
				json: { ...paywall, offering: route.request().postDataJSON().tier }
			});
		});
		await page.route('https://checkout.stripe.com/**', (route) =>
			route.fulfill({
				contentType: 'text/html',
				body: '<h1>Review payment with Stripe</h1>'
			})
		);
		await page.goto('/settings/subscriptions?checkout=success');
		await expect(
			page.getByRole('heading', { name: 'Current access: Free' })
		).toBeVisible();
		await page
			.getByRole('button', {
				name: `View ${tier === 'plus' ? 'Plus' : 'Business'} details`,
				exact: true
			})
			.click();
		await expect(
			page.getByRole('dialog', { name: /CraftSky (Plus|Business)$/ })
		).toBeVisible();
		await expect(
			page.getByRole('heading', {
				name: /^RevenueCat (plus|business) paywall$/
			})
		).toBeVisible();
		await expect(page).toHaveURL(/\/settings\/subscriptions(?:\?.*)?$/);
		expect(requests).toEqual([{ body: { tier }, account: alice.id }]);
	});

test('hides purchases while Stripe payment or a canceled native subscription is unresolved', async ({
	page
}) => {
	for (const subscription of [
		{
			store: 'stripe',
			status: 'pending',
			givesAccess: false,
			pendingPayment: true
		},
		{
			store: 'app_store',
			status: 'cancelled',
			givesAccess: true,
			pendingPayment: false
		}
	]) {
		await billingFixture(page, [
			{
				id: 'subscription',
				productId: 'plus',
				anomaly: 'none',
				...subscription
			}
		]);
		await page.goto('/settings/subscriptions');
		await expect(
			page.getByRole('region', { name: 'Current subscription' })
		).toBeVisible();
		await expect(
			page.getByRole('button', { name: /^View (Plus|Business) details$/ })
		).toHaveCount(0);
		await expect(
			page.getByRole('button', { name: /Assign|Unassign/ })
		).toHaveCount(0);
	}
});

test('unconfigured checkout and provider errors never navigate away or expose private customer IDs', async ({
	page
}) => {
	await billingFixture(page, [], false);
	await page.goto('/settings/subscriptions');
	await expect(
		page.getByText('Web subscriptions are not available yet.')
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: /^View (Plus|Business) details$/ })
	).toHaveCount(0);
	await billingFixture(page);
	await page.route(`**${appview}/billing/web-paywall`, (route) =>
		route.fulfill({
			status: 409,
			json: { error: 'subscription_already_exists', message: privateId }
		})
	);
	await page.reload();
	await page
		.getByRole('button', { name: 'View Plus details', exact: true })
		.click();
	await expect(page.getByRole('alert')).toContainText(
		'This account already has a subscription'
	);
	await expect(page).toHaveURL(/\/settings\/subscriptions$/);
	await expect(page.locator('body')).not.toContainText(privateId);
});

test('opens an account-specific Stripe portal without accepting a customer identifier', async ({
	page
}) => {
	await billingFixture(page, [
		{
			id: 'subscription',
			productId: 'plus',
			store: 'stripe',
			status: 'active',
			givesAccess: true,
			pendingPayment: false,
			anomaly: 'none'
		}
	]);
	const portal =
		'https://billing.stripe.com/p/session/test_fixture#portal%2Fcapability';
	const requests: { body: string | null; account: string | undefined }[] = [];
	await page.route(`**${appview}/billing/web-management`, async (route) => {
		requests.push({
			body: route.request().postData(),
			account: route.request().headers()['x-craftsky-account-id']
		});
		await route.fulfill({ json: { url: portal } });
	});
	await page.route('https://billing.stripe.com/**', (route) =>
		route.fulfill({
			contentType: 'text/html',
			body: '<h1>Manage this subscription</h1>'
		})
	);
	await page.goto('/settings/subscriptions');
	await expect(page.getByText('Billed through Stripe.')).toBeVisible();
	await page
		.getByRole('button', { name: 'Manage in Stripe', exact: true })
		.click();
	await expect(page).toHaveURL(portal);
	expect(requests).toEqual([{ body: null, account: alice.id }]);
});

test('shows a safe receipt fallback when account-specific Stripe management is unavailable', async ({
	page
}) => {
	await billingFixture(page, [
		{
			id: 'subscription',
			productId: 'plus',
			store: 'stripe',
			status: 'active',
			givesAccess: true,
			pendingPayment: false,
			anomaly: 'none'
		}
	]);
	await page.route(`**${appview}/billing/web-management`, (route) =>
		route.fulfill({
			status: 503,
			json: { error: 'web_management_unavailable', message: privateId }
		})
	);
	await page.goto('/settings/subscriptions');
	await page
		.getByRole('button', { name: 'Manage in Stripe', exact: true })
		.click();
	await expect(page.getByRole('alert')).toContainText(
		'Use the management link in your Stripe receipt.'
	);
	await expect(page).toHaveURL(/\/settings\/subscriptions$/);
	await expect(page.locator('body')).not.toContainText(privateId);
});

for (const [initial, updated] of [
	['free', 'plus'],
	['plus', 'free']
] as const)
	test(`reconciled ${initial} to ${updated} access reaches scheduling without reloading`, async ({
		page
	}) => {
		await billingFixture(page);
		let tier: 'free' | 'plus' = initial;
		await page.route(`**${appview}/subscriptions/access`, (route) =>
			route.fulfill({
				json: {
					did: alice.did,
					effectiveTier: tier,
					givesAccess: tier !== 'free'
				}
			})
		);
		await page.goto('/settings/subscriptions');
		await expect(
			page.getByRole('heading', {
				name: `Current access: ${initial === 'plus' ? 'Plus' : 'Free'}`
			})
		).toBeVisible();
		const documentStarted = await page.evaluate(() => performance.timeOrigin);
		tier = updated;
		await page.evaluate(() =>
			window.dispatchEvent(
				new PageTransitionEvent('pageshow', { persisted: true })
			)
		);
		await expect(
			page.getByRole('heading', {
				name: `Current access: ${updated === 'plus' ? 'Plus' : 'Free'}`
			})
		).toBeVisible();
		await page
			.getByRole('button', { name: 'Create post', exact: true })
			.click();
		await page.getByRole('link', { name: /^Regular post/ }).click();
		await expect(
			page.getByRole('heading', { name: 'New post', exact: true })
		).toBeVisible();
		await page.getByRole('button', { name: 'When', exact: true }).click();
		if (updated === 'plus')
			await expect(
				page.getByRole('menuitemradio', {
					name: 'Schedule for later',
					exact: true
				})
			).toBeVisible();
		else
			await expect(
				page.getByRole('dialog', { name: 'CraftSky Plus' })
			).toBeVisible();
		expect(await page.evaluate(() => performance.timeOrigin)).toBe(
			documentStarted
		);
	});

test('monthly plan cards match native copy and show actual web catalog prices', async ({
	page
}) => {
	await billingFixture(page);
	await page.goto('/settings/subscriptions');
	await expect(
		page.getByRole('button', { name: 'Refresh status', exact: true })
	).toHaveCount(0);
	await expect(page.getByText(/^Last checked /)).toHaveCount(0);
	await expect(
		page.getByText(
			'Your subscription belongs to this CraftSky account. Business includes everything in Plus.',
			{ exact: true }
		)
	).toBeVisible();
	for (const [tier, price] of [
		['Plus', '£2.49 / month'],
		['Business', '£8.99 / month']
	]) {
		const card = page.getByRole('region', {
			name: `CraftSky ${tier}`,
			exact: true
		});
		await expect(card.getByText(price, { exact: true })).toBeVisible();
		await expect(
			card.getByRole('button', { name: `View ${tier} details` })
		).toHaveText('View details');
	}
	await expect(page.getByText(/Tax\s+included/)).toBeVisible();
	await page.setViewportSize({ width: 390, height: 844 });
	await expect(page.getByText('£8.99 / month', { exact: true })).toBeVisible();
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
});

test('catalog failures do not hide access or invent a price and retry on page restoration', async ({
	page
}) => {
	await billingFixture(page);
	let available = false;
	await page.route(`**${appview}/billing/web-prices`, (route) =>
		available
			? route.fulfill({
					json: {
						items: [
							{
								tier: 'plus',
								amount: 379,
								currency: 'gbp',
								taxBehavior: 'inclusive'
							}
						]
					}
				})
			: route.fulfill({
					status: 503,
					json: { error: 'web_prices_unavailable', message: 'Unavailable' }
				})
	);
	await page.goto('/settings/subscriptions');
	await expect(
		page.getByRole('heading', { name: 'Current access: Free' })
	).toBeVisible();
	await expect(
		page.getByText('Price unavailable', { exact: true })
	).toHaveCount(2);
	await expect(page.getByText('£2.49 / month', { exact: true })).toHaveCount(0);
	available = true;
	await page.evaluate(() =>
		window.dispatchEvent(
			new PageTransitionEvent('pageshow', { persisted: true })
		)
	);
	await expect(page.getByText('£3.79 / month', { exact: true })).toBeVisible();
});

for (const tier of ['Plus', 'Business']) {
	test(`entire ${tier} card opens its checkout exactly once from the heading area`, async ({
		page
	}) => {
		await billingFixture(page);
		const selected: string[] = [];
		await page.route(`**${appview}/billing/web-paywall`, (route) => {
			selected.push(route.request().postDataJSON().tier);
			return route.fulfill({
				json: { ...paywall, offering: route.request().postDataJSON().tier }
			});
		});
		await page.route('https://checkout.stripe.com/**', (route) =>
			route.fulfill({
				contentType: 'text/html',
				body: '<h1>Review payment</h1>'
			})
		);
		await page.goto('/settings/subscriptions');
		const card = page.getByRole('region', {
			name: `CraftSky ${tier}`,
			exact: true
		});
		await expect(
			card.getByRole('button', { name: `View ${tier} details` })
		).toBeVisible();
		await card.click({ position: { x: 28, y: 28 } });
		await expect(
			page.getByRole('dialog', { name: /CraftSky (Plus|Business)$/ })
		).toBeVisible();
		await expect(
			page.getByRole('heading', {
				name: /^RevenueCat (plus|business) paywall$/
			})
		).toBeVisible();
		await expect(page).toHaveURL(/\/settings\/subscriptions(?:\?.*)?$/);
		expect(selected).toEqual([tier.toLowerCase()]);
	});
}

test('plan card action supports keyboard activation and a disabled catalog stays inert', async ({
	page
}) => {
	await billingFixture(page, [], false);
	let writes = 0;
	page.on('request', (request) => {
		if (request.url().endsWith('/billing/web-paywall')) writes++;
	});
	await page.goto('/settings/subscriptions');
	const card = page.getByRole('region', { name: 'CraftSky Plus', exact: true });
	await expect(card).toBeVisible();
	await card.click({ position: { x: 28, y: 28 } });
	expect(writes).toBe(0);
	await billingFixture(page);
	await page.route(`**${appview}/billing/web-paywall`, (route) =>
		route.fulfill({
			json: { ...paywall, offering: route.request().postDataJSON().tier }
		})
	);
	await page.route('https://checkout.stripe.com/**', (route) =>
		route.fulfill({ contentType: 'text/html', body: '<h1>Review payment</h1>' })
	);
	await page.reload();
	const action = card.getByRole('button', { name: 'View Plus details' });
	await action.focus();
	await action.press('Enter');
	await expect(
		page.getByRole('dialog', { name: /CraftSky (Plus|Business)$/ })
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: /^RevenueCat (plus|business) paywall$/ })
	).toBeVisible();
	await expect(page).toHaveURL(/\/settings\/subscriptions(?:\?.*)?$/);
	expect(writes).toBe(1);
});

test('paywall closes on Escape without direct-checkout cancellation and SDK success still reconciles access', async ({
	page
}) => {
	await billingFixture(page);
	const requests: unknown[] = [];
	await page.route(`**${appview}/billing/web-paywall`, (route) => {
		requests.push(route.request().postDataJSON());
		return route.fulfill({ json: paywall });
	});
	let cancellations = 0;
	await page.route(`**${appview}/billing/web-checkout/cancel`, (route) => {
		cancellations++;
		return route.fulfill({ json: { status: 'canceled' } });
	});
	await page.goto('/settings/subscriptions');
	const trigger = page.getByRole('button', { name: 'View Plus details' });
	await trigger.click();
	const dialog = page.getByRole('dialog', { name: 'CraftSky Plus' });
	await expect(
		dialog.getByRole('heading', { name: 'RevenueCat plus paywall' })
	).toBeVisible();
	await page.keyboard.press('Escape');
	await expect(dialog).toHaveCount(0);
	expect(cancellations).toBe(0);
	await expect(trigger).toBeFocused();
	await trigger.click();
	await dialog.getByRole('button', { name: 'Continue to checkout' }).click();
	const checkout = page.getByRole('dialog', { name: 'CraftSky Plus checkout' });
	await expect(
		checkout.getByRole('heading', { name: 'Stripe checkout' })
	).toBeVisible();
	await checkout
		.getByRole('button', { name: 'Complete fixture payment' })
		.click();
	await expect(dialog).toHaveCount(0);
	await expect(
		page.getByRole('heading', { name: 'Current access: Free' })
	).toBeVisible();
	expect(requests).toEqual([
		{ tier: 'plus' },
		{ tier: 'plus' },
		{ tier: 'plus' }
	]);
	expect(cancellations).toBe(0);
});

for (const viewport of [
	{ width: 1280, height: 900 },
	{ width: 1050, height: 900 },
	{ width: 390, height: 844 }
])
	test(`paywall sits at the bottom at ${viewport.width}px and checkout cancellation restores it`, async ({
		page
	}) => {
		await page.setViewportSize(viewport);
		await billingFixture(page);
		await page.route(`**${appview}/billing/web-paywall`, (route) =>
			route.fulfill({ json: paywall })
		);
		await page.goto('/settings/subscriptions');
		await page.getByRole('button', { name: 'View Plus details' }).click();
		const sheet = page.getByRole('dialog', {
			name: 'CraftSky Plus',
			exact: true
		});
		await expect(
			sheet.getByRole('heading', { name: 'RevenueCat plus paywall' })
		).toBeVisible();
		await expect
			.poll(async () => {
				const box = await sheet.boundingBox();
				return box ? Math.abs(box.y + box.height - viewport.height) : Infinity;
			})
			.toBeLessThanOrEqual(1);
		expect((await sheet.boundingBox())!.width).toBeLessThanOrEqual(720);
		await sheet
			.getByRole('heading', { name: 'RevenueCat plus paywall' })
			.click();
		await expect(sheet).toBeVisible();
		// Dragging from content onto the backdrop is not an outside click.
		const bounds = (await sheet.boundingBox())!;
		await page.mouse.move(bounds.x + 20, bounds.y + 20);
		await page.mouse.down();
		await page.mouse.move(10, 10);
		await page.mouse.up();
		await expect(sheet).toBeVisible();
		await page.mouse.click(10, 10);
		await expect(sheet).toHaveCount(0);
		await expect(
			page.getByRole('button', { name: 'View Plus details', exact: true })
		).toBeFocused();
		await page
			.getByRole('button', { name: 'View Plus details', exact: true })
			.click();
		await expect(
			sheet.getByRole('heading', { name: 'RevenueCat plus paywall' })
		).toBeVisible();
		await sheet.getByRole('button', { name: 'Continue to checkout' }).click();
		const checkout = page.getByRole('dialog', {
			name: 'CraftSky Plus checkout',
			exact: true
		});
		await expect(
			checkout.getByRole('heading', { name: 'Stripe checkout' })
		).toBeVisible();
		await expect(
			page.getByRole('heading', { name: 'RevenueCat plus paywall' })
		).not.toBeVisible();
		// Force a vertical scrollbar like real Stripe Checkout, then check the
		// frame's available width rather than just the outer modal width.
		await checkout.locator('.checkout-host').evaluate((e) => {
			e.style.minHeight = '1600px';
		});
		const checkoutWidth = await checkout
			.locator('.checkout-host')
			.evaluate((e) => e.clientWidth);
		if (viewport.width >= 1050)
			expect(checkoutWidth).toBeGreaterThanOrEqual(991);
		else expect(checkoutWidth).toBeLessThan(991);
		expect(
			await page.evaluate(
				() => document.documentElement.scrollWidth <= innerWidth
			)
		).toBe(true);
		await checkout.getByRole('button', { name: 'Back to plans' }).click();
		await expect(
			sheet.getByRole('heading', { name: 'RevenueCat plus paywall' })
		).toBeVisible();
		await expect(sheet.getByRole('alert')).toHaveCount(0);
		await expect(
			sheet.getByRole('button', { name: 'Close checkout', exact: true })
		).toHaveCount(0);
		await sheet.getByRole('button', { name: 'Continue to checkout' }).click();
		await expect(
			checkout.getByRole('heading', { name: 'Stripe checkout' })
		).toBeVisible();
		await checkout
			.getByRole('button', { name: 'Close checkout', exact: true })
			.click();
		await expect(checkout).toHaveCount(0);
		await expect(
			page.getByRole('button', { name: 'View Plus details', exact: true })
		).toBeFocused();
		await expect(
			page.getByRole('heading', { name: 'Current access: Free' })
		).toBeVisible();
	});
