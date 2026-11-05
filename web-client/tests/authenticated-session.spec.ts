import { test, expect, type BrowserContext, type Page } from '@playwright/test';
const base = 'http://127.0.0.1:19846',
	fixture = 'http://127.0.0.1:19847';
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
async function configure(value: unknown) {
	const response = await fetch(fixture + '/__test/control', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(value)
	});
	expect(response.ok).toBe(true);
}
async function authState() {
	return (await (await fetch(fixture + '/__test/auth-state')).json()) as {
		events: {
			stage: string;
			name: string;
			meetsMinimumAge?: boolean;
			policyVersion?: string;
		}[];
		deletions: { jobId: string; name: string; accepted: boolean }[];
	};
}
async function seed(context: BrowserContext, both = false) {
	await context.addCookies(
		[alice, ...(both ? [bob] : [])].map((account) => ({
			name: 'craftsky_account_' + account.id,
			value: Buffer.from(JSON.stringify(account)).toString('base64url'),
			url: base,
			httpOnly: true,
			sameSite: 'Lax' as const
		}))
	);
	await context.addCookies([
		{
			name: 'craftsky_active',
			value: alice.id,
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
}
async function signIn(page: Page, handle = 'alice.test') {
	await page.goto('/sign-in');
	await page.getByLabel('Your Atmosphere Handle').fill(handle);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
}
async function accounts(page: Page) {
	return await page.evaluate(async () => {
		const response = await fetch('/api/session');
		return (await response.json()) as {
			accounts: { id: string; did: string; handle: string }[];
			active: string | null;
		};
	});
}
test.beforeEach(async ({ page }) => {
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await fetch(fixture + '/__test/reset');
});

test('browser sign-in crosses the fake provider and persists HttpOnly pending before confirmation', async ({
	page,
	context
}) => {
	const pendingResponse = page.waitForResponse(
		(response) => new URL(response.url()).pathname === '/auth/browser/complete'
	);
	const callbackPaths: string[] = [];
	page.on('request', (request) => {
		const url = new URL(request.url());
		if (url.pathname.includes('complete')) callbackPaths.push(url.pathname);
	});

	await signIn(page);
	await expect(page).toHaveURL(/\/feed$/);
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	const response = await pendingResponse;
	const setCookies = (await response.headersArray())
		.filter((header) => header.name.toLowerCase() === 'set-cookie')
		.map((header) => header.value);
	const pendingCookie = setCookies.find((value) =>
		value.startsWith('craftsky_pending=')
	);
	expect(pendingCookie).toContain('HttpOnly');
	expect(pendingCookie).toContain('SameSite=Lax');
	expect(
		setCookies.some((value) => value.startsWith('craftsky_account_'))
	).toBe(false);
	expect(callbackPaths).toContain('/auth/browser/complete');
	expect(callbackPaths).not.toContain('/auth/complete');
	const cookies = await context.cookies(),
		credential = cookies.find((cookie) =>
			cookie.name.startsWith('craftsky_account_')
		)!;
	expect(credential.httpOnly).toBe(true);
	expect(credential.sameSite).toBe('Lax');
	expect(cookies.some((cookie) => cookie.name === 'craftsky_pending')).toBe(
		false
	);
	const privateValue = JSON.parse(
		Buffer.from(credential.value, 'base64url').toString()
	) as { token: string };
	expect(await page.content()).not.toContain(privateValue.token);
	expect(await page.evaluate(() => document.cookie)).not.toContain('craftsky_');
	expect((await authState()).events.map((event) => event.stage)).toEqual([
		'start',
		'provider',
		'exchange',
		'confirm'
	]);
});

test('a temporary confirmation outage retains pending state and retries without repeating provider authorization', async ({
	page,
	context
}) => {
	await configure({ confirmFailures: 1 });
	await signIn(page);
	await expect(page).toHaveURL(/\/sign-in\?error=confirmation_pending$/);
	expect(
		(await context.cookies()).some(
			(cookie) => cookie.name === 'craftsky_pending'
		)
	).toBe(true);
	expect(
		(await context.cookies()).filter((cookie) =>
			cookie.name.startsWith('craftsky_account_')
		)
	).toHaveLength(0);
	await page.getByRole('link', { name: 'Retry confirmation' }).click();
	await expect(page).toHaveURL(/\/feed$/);
	const stages = (await authState()).events.map((event) => event.stage);
	expect(stages.filter((stage) => stage === 'confirm')).toHaveLength(2);
	expect(stages.filter((stage) => stage === 'exchange')).toHaveLength(1);
	expect(stages.filter((stage) => stage === 'provider')).toHaveLength(1);
});

test('two browser accounts switch independently and isolate API401 while preserving cookies on503', async ({
	page,
	context
}) => {
	await signIn(page);
	await expect(page).toHaveURL(/\/feed$/);
	await signIn(page, 'bob.test');
	await expect(page).toHaveURL(/\/feed$/);
	let state = await accounts(page);
	expect(state.accounts.map((account) => account.did).sort()).toEqual([
		alice.did,
		bob.did
	]);
	expect(
		state.accounts.find((account) => account.id === state.active)?.did
	).toBe(bob.did);
	await page.getByRole('button', { name: /^Switch accounts,/ }).click();
	await page
		.locator('.account-menu .person-row')
		.filter({ hasText: '@alice.test' })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Switch accounts?', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', {
			name: 'Switch accounts, @alice.test',
			exact: true
		})
	).toBeVisible();
	state = await accounts(page);
	const bobId = state.accounts.find((account) => account.did === bob.did)!.id,
		aliceId = state.accounts.find((account) => account.did === alice.did)!.id;
	await configure({
		owners: { bob: { apiStatus: 503, apiPath: '/v1/profiles/me' } }
	});
	let status = await page.evaluate(
		async (id) =>
			(
				await fetch('/api/appview/v1/profiles/me', {
					headers: { 'X-Craftsky-Account-Id': id }
				})
			).status,
		bobId
	);
	expect(status).toBe(503);
	expect((await accounts(page)).accounts).toHaveLength(2);
	await configure({
		owners: { bob: { apiStatus: 401, apiPath: '/v1/profiles/me' } }
	});
	status = await page.evaluate(
		async (id) =>
			(
				await fetch('/api/appview/v1/profiles/me', {
					headers: { 'X-Craftsky-Account-Id': id }
				})
			).status,
		bobId
	);
	expect(status).toBe(401);
	state = await accounts(page);
	expect(state.accounts.map((account) => account.did)).toEqual([alice.did]);
	expect(state.active).toBe(aliceId);
	expect(
		(await context.cookies()).some(
			(cookie) => cookie.name === 'craftsky_account_' + aliceId
		)
	).toBe(true);
});

for (const status of [503, 401]) {
	test(`startup ${status} preserves an accessible switch to another retained account`, async ({
		page,
		context
	}) => {
		await page.setViewportSize({ width: 402, height: 874 });
		await seed(context, true);
		await configure({
			owners: {
				alice: { apiStatus: status, apiPath: '/v1/account/eligibility' }
			}
		});
		await page.goto('/feed');
		await expect(
			page.getByRole('heading', {
				name:
					status === 401 ? 'Your session has expired' : 'Account unavailable',
				exact: true
			})
		).toBeVisible();
		await expect(
			page.getByText('Loading your CraftSky…', { exact: true })
		).toHaveCount(0);
		await page
			.getByRole('button', { name: 'Switch account', exact: true })
			.click();
		const chooser = page.getByRole('dialog', { name: 'Accounts', exact: true });
		await expect(chooser).toBeVisible();
		await chooser.getByRole('button', { name: /@bob\.test/ }).click();
		await expect(
			page.getByRole('dialog', { name: 'Switch accounts?', exact: true })
		).toHaveCount(0);
		await expect(
			page.getByRole('heading', { name: 'Home', exact: true })
		).toBeVisible();
		const state = await accounts(page);
		expect(
			state.accounts.find((account) => account.id === state.active)?.did
		).toBe(bob.did);
		expect(state.accounts.some((account) => account.did === alice.did)).toBe(
			status === 503
		);
	});
}

test('onboarding requires the16-year age and policy declaration independently of diagnostics configuration', async ({
	page,
	context
}) => {
	await seed(context);
	await configure({ owners: { alice: { completed: false } } });
	await page.goto('/feed');
	await expect(
		page.getByRole('heading', { name: 'Make it yours', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'What do you make?', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Our community guidelines' })
	).toBeVisible();
	const declaration = page.getByRole('checkbox', { name: /16 years old/ }),
		join = page.getByRole('button', {
			name: 'Finish',
			exact: true
		});
	await expect(declaration).toBeVisible();
	await expect(join).toBeDisabled();
	const diagnostics = page.getByRole('checkbox', {
		name: /Enable error reporting/
	});
	if (await diagnostics.count()) {
		await diagnostics.check();
		await expect(join).toBeDisabled();
		await diagnostics.uncheck();
	}
	await declaration.check();
	await expect(join).toBeEnabled();
	await join.click();
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	const completion = (await authState()).events.find(
		(event) => event.stage === 'onboarding'
	);
	expect(completion).toMatchObject({
		name: 'alice',
		meetsMinimumAge: true,
		policyVersion: 'v1'
	});
	expect(
		await page.evaluate(() =>
			localStorage.getItem('craftsky.diagnostics-consent')
		)
	).not.toBe('true');
});

test('restricted accounts retain settings and complete owner-bound reauthentication and empty202 deletion', async ({
	page,
	context
}) => {
	await seed(context, true);
	await configure({ owners: { alice: { eligibility: 'restricted' } } });
	await page.goto('/feed');
	await expect(
		page.getByRole('heading', { name: 'Your account is restricted' })
	).toBeVisible();
	await page.goto('/settings');
	await expect(
		page.getByRole('heading', { name: 'Settings', exact: true })
	).toBeVisible();
	await page.goto('/settings/account');
	await page
		.getByRole('button', { name: 'Delete account', exact: true })
		.click();
	const deletionDialog = page.getByRole('dialog');
	await expect(deletionDialog).toBeVisible();
	await expect(
		deletionDialog.getByRole('heading', {
			name: 'Delete CraftSky account?',
			exact: true
		})
	).toBeVisible();
	await page.getByRole('button', { name: 'Continue', exact: true }).click();
	await expect(page).toHaveURL(/\/settings\/account\?deletion=ready$/);
	const identifier = page.getByRole('textbox', {
			name: 'Account identifier',
			exact: true
		}),
		confirm = page.getByRole('button', {
			name: 'Permanently delete CraftSky account',
			exact: true
		});
	await expect(identifier).toBeVisible();
	await identifier.fill(bob.did);
	await expect(confirm).toBeDisabled();
	await identifier.fill(alice.did);
	await expect(confirm).toBeEnabled();
	const pending = (await context.cookies()).find(
		(cookie) => cookie.name === 'craftsky_deletion_pending'
	)!;
	expect(pending.httpOnly).toBe(true);
	const proof = JSON.parse(
		Buffer.from(pending.value, 'base64url').toString()
	) as { jobId: string; proof: string };
	expect(await page.content()).not.toContain(proof.proof);
	const wrong = await page.evaluate(
		async (input) =>
			(
				await fetch('/api/auth/deletion/confirm', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
						'X-Craftsky-Account-Id': input.accountId
					},
					body: JSON.stringify({
						jobId: input.jobId,
						confirmationDid: 'did:plc:alice'
					})
				})
			).status,
		{ accountId: bob.id, jobId: proof.jobId }
	);
	expect(wrong).toBe(400);
	expect((await authState()).deletions[0].accepted).toBe(false);
	await confirm.click();
	await expect(page).toHaveURL(/\/sign-in$/);
	expect((await accounts(page)).accounts.map((account) => account.did)).toEqual(
		[bob.did]
	);
	expect(
		(await context.cookies()).some(
			(cookie) => cookie.name === 'craftsky_deletion_pending'
		)
	).toBe(false);
	expect((await authState()).deletions[0]).toMatchObject({
		name: 'alice',
		accepted: true
	});
});
