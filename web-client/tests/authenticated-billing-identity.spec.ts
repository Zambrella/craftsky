import { expect, test } from '@playwright/test';

const base = 'http://127.0.0.1:19846';
const appview = '/api/appview/v1';
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
const billingState = {
	billingAccountId: 'account',
	revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
	requestedGeneration: 1,
	reconciledGeneration: 0,
	reconciliationStale: true,
	webCheckoutAvailable: false,
	subscriptions: [],
	licenses: []
};

test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('**/api/service-status', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await context.addCookies([
		...[alice, bob].map((account) => ({
			name: `craftsky_account_${account.id}`,
			value: Buffer.from(JSON.stringify(account)).toString('base64url'),
			url: base,
			httpOnly: true,
			sameSite: 'Lax' as const
		})),
		{
			name: 'craftsky_active',
			value: alice.id,
			url: base,
			httpOnly: true,
			sameSite: 'Lax' as const
		}
	]);
});

test('registers without visiting billing and refreshes access without blocking the app', async ({
	page
}) => {
	let release!: () => void;
	const registration = new Promise<void>((resolve) => (release = resolve));
	let registered = false,
		attempts = 0,
		refreshed = 0;
	await page.route(`**${appview}/billing/account`, async (route) => {
		expect(route.request().method()).toBe('PUT');
		expect(route.request().headers()['x-craftsky-account-id']).toBe(alice.id);
		attempts++;
		await registration;
		registered = true;
		await route.fulfill({ json: billingState });
	});
	await page.route(`**${appview}/subscriptions/access`, (route) => {
		if (registered) refreshed++;
		return route.fulfill({
			json: {
				did: alice.did,
				effectiveTier: registered ? 'plus' : 'free',
				givesAccess: registered
			}
		});
	});
	try {
		await page.goto('/compose');
		await expect(
			page.getByRole('heading', { name: 'New post', exact: true })
		).toBeVisible();
		await expect.poll(() => attempts).toBe(1);
		expect(registered).toBe(false);
	} finally {
		release();
	}
	await expect.poll(() => refreshed).toBe(1);
	await page.getByRole('button', { name: 'When', exact: true }).click();
	await expect(
		page.getByRole('menuitemradio', { name: 'Schedule for later', exact: true })
	).toBeVisible();
	expect(attempts).toBe(1);
});

test('registration failure leaves the signed-in app usable', async ({
	page
}) => {
	let attempts = 0;
	await page.route(`**${appview}/billing/account`, (route) => {
		attempts++;
		return route.fulfill({
			status: 503,
			json: { error: 'provider_unavailable', message: 'fixture failure' }
		});
	});
	await page.goto('/compose');
	await expect(
		page.getByRole('heading', { name: 'New post', exact: true })
	).toBeVisible();
	await expect.poll(() => attempts).toBe(1);
	await expect(
		page.getByText('Account unavailable', { exact: true })
	).toHaveCount(0);
});

test('restored sessions and account switching register the selected account', async ({
	page
}) => {
	const accounts: string[] = [];
	await page.route(`**${appview}/billing/account`, (route) => {
		accounts.push(route.request().headers()['x-craftsky-account-id']);
		return route.fulfill({ json: billingState });
	});
	await page.goto('/feed');
	await expect.poll(() => accounts).toEqual([alice.id]);
	await page.reload();
	await expect.poll(() => accounts).toEqual([alice.id, alice.id]);
	await page.getByRole('button', { name: /^Switch accounts,/ }).click();
	await page
		.locator('.account-menu .person-row')
		.filter({ hasText: '@bob.test' })
		.click();
	await expect(
		page.getByRole('button', {
			name: 'Switch accounts, @bob.test',
			exact: true
		})
	).toBeVisible();
	await expect.poll(() => accounts).toEqual([alice.id, alice.id, bob.id]);
});

for (const gate of ['onboarding', 'eligibility'])
	test(`does not register before ${gate} permits account use`, async ({
		page
	}) => {
		let attempts = 0;
		await page.route(`**${appview}/billing/account`, (route) => {
			attempts++;
			return route.fulfill({ json: billingState });
		});
		await page.route(
			`**${appview}/${gate === 'onboarding' ? 'onboarding/status' : 'account/eligibility'}`,
			(route) =>
				route.fulfill({
					json:
						gate === 'onboarding'
							? { completed: false, requiredPolicyVersion: 'v1' }
							: { state: 'restricted', appealable: true }
				})
		);
		await page.goto('/feed');
		await expect(
			gate === 'onboarding'
				? page.getByRole('heading', { name: 'Set up your CraftSky profile' })
				: page.getByRole('heading', { name: 'Your account is restricted' })
		).toBeVisible();
		expect(attempts).toBe(0);
		if (gate === 'onboarding') {
			await page.getByRole('button', { name: 'Next', exact: true }).click();
			await page.getByRole('button', { name: 'Next', exact: true }).click();
			await page
				.getByRole('checkbox', {
					name: 'I confirm that I am at least 16 years old and accept the current CraftSky policies.'
				})
				.check();
			await page.getByRole('button', { name: 'Finish', exact: true }).click();
			await expect.poll(() => attempts).toBe(1);
		}
	});

test('respects the subscriptions release flag without blocking account readiness', async ({
	page
}) => {
	let attempts = 0;
	await page.route(`**${appview}/billing/account`, (route) => {
		attempts++;
		return route.fulfill({ json: billingState });
	});
	await page.goto('/compose');
	await expect(
		page.getByRole('heading', { name: 'New post', exact: true })
	).toBeVisible();
	await page.waitForLoadState('networkidle');
	expect(attempts).toBe(
		process.env.WEB_CLIENT_SUBSCRIPTIONS_ENABLED === 'false' ? 0 : 1
	);
});
