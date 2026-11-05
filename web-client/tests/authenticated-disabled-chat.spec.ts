import { test, expect } from '@playwright/test';
import type { ChatStatus } from '../src/lib/api/types';

const base = 'http://127.0.0.1:19846';
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const disabled: ChatStatus = {
	generation: 1,
	enabled: false,
	pushEnabled: false,
	state: 'disabled',
	capabilities: {
		canRead: false,
		canSend: false,
		canReport: false,
		canSetPolicy: false
	}
};
const disclosures = [
	'This is your shared Bluesky inbox, showing conversations with CraftSky members, including those started in other apps. Read, accept, mute and leave actions also affect those apps.',
	'Messages are stored by Bluesky, outside public PDS records. They are not end-to-end encrypted.',
	'Message reports go to Bluesky only. CraftSky moderators do not receive these reports.'
];

test.beforeEach(async ({ page, context }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await context.addCookies([
		{
			name: 'craftsky_account_' + alice.id,
			value: Buffer.from(JSON.stringify(alice)).toString('base64url'),
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		},
		{
			name: 'craftsky_active',
			value: alice.id,
			url: base,
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
});

test('disabled messaging matches the native settings gate without inbox actions or implicit writes', async ({
	page
}) => {
	const scopes: string[] = [],
		writes: string[] = [],
		privateReads: string[] = [];
	page.on('request', (request) => {
		const path = new URL(request.url()).pathname;
		if (path.startsWith('/api/appview/v1/chat/') && request.method() !== 'GET')
			writes.push(path);
		if (path.startsWith('/api/appview/v1/chat/conversations'))
			privateReads.push(path);
	});
	await page.route('**/api/appview/v1/chat/status', (route) => {
		scopes.push(route.request().headers()['x-craftsky-account-id']);
		return route.fulfill({ json: disabled });
	});
	await page.goto('/messages');
	const workspace = page.getByRole('region', {
		name: 'Direct messages',
		exact: true
	});
	await expect(
		workspace.getByRole('heading', {
			name: 'Direct messages',
			level: 1,
			exact: true
		})
	).toBeVisible();
	await expect(workspace.locator('.settings-body > p')).toHaveText(disclosures);
	await expect(
		workspace.getByRole('checkbox', { name: /^Enable messaging\b/ })
	).not.toBeChecked();
	await expect(
		workspace.getByRole('navigation', {
			name: 'Messaging actions',
			exact: true
		})
	).toHaveCount(0);
	await expect(
		workspace.getByRole('link', { name: 'Requests', exact: true })
	).toHaveCount(0);
	await expect(
		workspace.getByRole('link', {
			name: 'Direct message settings',
			exact: true
		})
	).toHaveCount(0);
	await expect(
		workspace.getByRole('button', { name: 'Refresh messages', exact: true })
	).toHaveCount(0);
	await expect(
		workspace.getByRole('link', { name: 'New message', exact: true })
	).toHaveCount(0);
	expect(scopes.length).toBeGreaterThan(0);
	expect(scopes.every((id) => id === alice.id)).toBe(true);
	expect(writes).toEqual([]);
	expect(privateReads).toEqual([]);
});

test('authorization-needed messaging retains its existing permission control in the native settings gate', async ({
	page
}) => {
	await page.route('**/api/appview/v1/chat/status', (route) =>
		route.fulfill({
			json: {
				...disabled,
				enabled: true,
				state: 'authorizationNeeded'
			} satisfies ChatStatus
		})
	);
	await page.goto('/messages');
	const workspace = page.getByRole('region', {
		name: 'Direct messages',
		exact: true
	});
	await expect(
		workspace.getByRole('heading', {
			name: 'Direct messages',
			level: 1,
			exact: true
		})
	).toBeVisible();
	await expect(
		workspace.getByRole('checkbox', { name: /^Enable messaging\b/ })
	).toBeChecked();
	await expect(
		workspace
			.locator('.permission-panel')
			.getByRole('button', { name: 'Review permissions', exact: true })
	).toBeVisible();
	await expect(
		workspace.getByRole('navigation', {
			name: 'Messaging actions',
			exact: true
		})
	).toHaveCount(0);
});

test('a restricted nonready gate retains refresh and can recover into the disabled settings gate', async ({
	page
}) => {
	let restricted = true,
		requests = 0;
	await page.route('**/api/appview/v1/chat/status', (route) => {
		requests++;
		return route.fulfill({
			json: restricted
				? ({
						...disabled,
						enabled: true,
						state: 'restricted'
					} satisfies ChatStatus)
				: disabled
		});
	});
	await page.goto('/messages');
	const workspace = page.getByRole('region', {
		name: 'Direct messages',
		exact: true
	});
	await expect(
		workspace.getByRole('heading', { name: 'Messages', level: 1, exact: true })
	).toBeVisible();
	await expect(
		workspace.getByText(
			'Messaging is restricted by Bluesky. Other CraftSky features are still available.',
			{ exact: true }
		)
	).toBeVisible();
	await expect(
		workspace.getByRole('navigation', {
			name: 'Messaging actions',
			exact: true
		})
	).toHaveCount(0);
	const previous = requests;
	restricted = false;
	await workspace.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(
		workspace.getByRole('heading', {
			name: 'Direct messages',
			level: 1,
			exact: true
		})
	).toBeVisible();
	expect(requests).toBeGreaterThan(previous);
	await expect(
		workspace.getByRole('checkbox', { name: /^Enable messaging\b/ })
	).not.toBeChecked();
});
