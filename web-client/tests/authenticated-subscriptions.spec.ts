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
const checkout =
	'https://checkout.stripe.com/c/pay/cs_test_fixture#opaque%2Fcapability';
const appview = '/api/appview/v1';

test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
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
		await page.route(`**${appview}/billing/web-checkout`, (route) => {
			requests.push(route.request().postDataJSON());
			return route.fulfill({ json: { url: checkout } });
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
		await expect(complimentary).toContainText('Complimentary access ends');
		await expect(complimentary).toContainText('no recurring charge');
		await expect(
			page.getByText('Paid billing starts immediately.', { exact: false })
		).toBeVisible();
		await expect(
			page.getByText(
				'Unused complimentary time is not added to your paid subscription.',
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
		await expect(page).toHaveURL(checkout);
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
			name: /^(Choose|Subscribe to) (Plus|Business)$/
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
		page.getByRole('button', { name: 'Choose Plus', exact: true })
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
	test(`opens identified Stripe checkout for ${tier} without accepting payment as access`, async ({
		page
	}) => {
		await billingFixture(page);
		const requests: { body: unknown; account: string | undefined }[] = [];
		await page.route(`**${appview}/billing/web-checkout`, async (route) => {
			requests.push({
				body: route.request().postDataJSON(),
				account: route.request().headers()['x-craftsky-account-id']
			});
			await route.fulfill({ json: { url: checkout } });
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
				name: `Choose ${tier === 'plus' ? 'Plus' : 'Business'}`,
				exact: true
			})
			.click();
		await expect(page).toHaveURL(checkout);
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
			page.getByRole('button', { name: /^Choose (Plus|Business)$/ })
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
		page.getByRole('button', { name: /^Choose (Plus|Business)$/ })
	).toHaveCount(0);
	await billingFixture(page);
	await page.route(`**${appview}/billing/web-checkout`, (route) =>
		route.fulfill({
			status: 409,
			json: { error: 'subscription_already_exists', message: privateId }
		})
	);
	await page.reload();
	await page.getByRole('button', { name: 'Choose Plus', exact: true }).click();
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
		await page
			.getByRole('button', { name: 'Refresh status', exact: true })
			.click();
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
				page.getByRole('dialog', { name: 'Scheduled posts with CraftSky Plus' })
			).toBeVisible();
		expect(await page.evaluate(() => performance.timeOrigin)).toBe(
			documentStarted
		);
	});
