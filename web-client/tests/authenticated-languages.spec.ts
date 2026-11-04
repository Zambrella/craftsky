import { test, expect, type BrowserContext } from '@playwright/test';
const base = 'http://127.0.0.1:19846',
	fixture = 'http://127.0.0.1:19847';
const accounts = [
	{
		id: '0c10030a-6208-43c2-8048-d3edcc609271',
		did: 'did:plc:alice',
		handle: 'alice.test',
		token: Buffer.alloc(32, 1).toString('base64url')
	},
	{
		id: 'c6dc7d6a-4bff-45f0-87ee-d42634646546',
		did: 'did:plc:bob',
		handle: 'bob.test',
		token: Buffer.alloc(32, 2).toString('base64url')
	}
];
async function seed(context: BrowserContext) {
	await context.addCookies(
		accounts.map((account) => ({
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
			value: accounts[0].id,
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
}
async function configure(owners: unknown) {
	const response = await fetch(fixture + '/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ owners })
	});
	expect(response.ok).toBe(true);
}
async function calls() {
	return (await (await fetch(fixture + '/__test/requests')).json()) as {
		path: string;
		method: string;
		owner: string;
		body?: unknown;
	}[];
}
test.use({ locale: 'fr-FR' });
test.beforeEach(async ({ page, context }) => {
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await fetch(fixture + '/__test/reset');
	await seed(context);
});

test('initializes missing language preferences before requesting Home and preserves them on reload', async ({
	page
}) => {
	await configure({ alice: { languages: null } });
	let release!: () => void;
	const pending = new Promise<void>((resolve) => {
		release = resolve;
	});
	let initializationStarted = false;
	await page.route('**/v1/languages/preferences/initialize', async (route) => {
		initializationStarted = true;
		await pending;
		await route.continue();
	});
	await page.goto('/app/home');
	try {
		await expect.poll(() => initializationStarted).toBe(true);
		await expect(
			page.getByText('Loading your CraftSky…', { exact: true })
		).toBeVisible();
		expect(
			(await calls()).some((call) =>
				['/v1/feed/timeline', '/v1/projects'].includes(call.path)
			)
		).toBe(false);
	} finally {
		release();
	}
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	await expect
		.poll(
			async () =>
				(await calls()).filter((call) => call.path === '/v1/feed/timeline')
					.length
		)
		.toBe(1);
	const requests = await calls();
	const initialized = requests.find((call) =>
		call.path.endsWith('/preferences/initialize')
	)!;
	expect(initialized.body).toEqual({
		primaryLanguage: 'fr',
		contentLanguages: ['fr', 'en']
	});
	expect(initialized.owner).toBe(accounts[0].did);
	expect(requests.indexOf(initialized)).toBeLessThan(
		requests.findIndex((call) => call.path === '/v1/feed/timeline')
	);
	await page.reload();
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	expect(
		(await calls()).filter((call) =>
			call.path.endsWith('/preferences/initialize')
		)
	).toHaveLength(1);
	await expect(
		page.getByText('language preferences lookup failed', { exact: true })
	).toHaveCount(0);
});

test('keeps existing saved language choices when browser languages differ', async ({
	page
}) => {
	await configure({
		alice: { languages: { primaryLanguage: 'cy', contentLanguages: [] } }
	});
	await page.goto('/app/settings/languages');
	await expect(
		page.getByRole('combobox', { name: 'Primary language', exact: true })
	).toHaveValue('Welsh');
	await expect(
		page.getByText(/If none are selected, all languages will be shown\./)
	).toBeVisible();
	await page
		.getByRole('searchbox', { name: 'Content languages', exact: true })
		.click();
	await page
		.getByRole('searchbox', { name: 'Content languages', exact: true })
		.fill('French');
	await expect(
		page.getByRole('checkbox', { name: 'French', exact: true })
	).not.toBeChecked();
	await expect(
		page.getByRole('combobox', { name: 'Primary language', exact: true })
	).toHaveValue('Welsh');
	expect(
		(await calls()).filter(
			(call) =>
				call.path.endsWith('/preferences/initialize') || call.method === 'PUT'
		)
	).toHaveLength(0);
});

test('keeps Home behind a retryable setup gate if language initialization fails', async ({
	page
}) => {
	await configure({
		alice: { languages: null, languageInitializeFailures: 1 }
	});
	await page.goto('/app/home');
	await expect(page.getByRole('alert')).toContainText(
		'CraftSky could not load your account'
	);
	expect(
		(await calls()).some((call) =>
			['/v1/feed/timeline', '/v1/projects'].includes(call.path)
		)
	).toBe(false);
	await page.getByRole('button', { name: 'Retry', exact: true }).click();
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	expect(
		(await calls()).filter((call) =>
			call.path.endsWith('/preferences/initialize')
		)
	).toHaveLength(2);
});

test('initializes each newly selected account through its own bound session', async ({
	page
}) => {
	await configure({ alice: { languages: null }, bob: { languages: null } });
	await page.goto('/app/home');
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	await page.getByRole('button', { name: /^Switch accounts,/ }).click();
	await page
		.locator('.account-menu .person-row')
		.filter({ hasText: '@bob.test' })
		.click();
	await expect(
		page.getByRole('dialog', { name: 'Switch accounts?', exact: true })
	).toHaveCount(0);
	await expect(
		page.getByRole('button', {
			name: 'Switch accounts, @bob.test',
			exact: true
		})
	).toBeVisible();
	await expect(
		page.getByRole('heading', { name: 'Home', exact: true })
	).toBeVisible();
	expect(
		(await calls())
			.filter((call) => call.path.endsWith('/preferences/initialize'))
			.map((call) => call.owner)
	).toEqual(accounts.map((account) => account.did));
});
