import { test, expect, type Page } from '@playwright/test';
const base = 'http://127.0.0.1:19846';
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
const editor = (page: Page) =>
	page.getByRole('textbox', {
		name: 'What would you like to share?',
		exact: true
	});
async function localDrafts(
	page: Page
): Promise<{ text: string; pending?: { key: string } }[]> {
	return page.evaluate(async () => {
		if (
			!(await indexedDB.databases()).some(
				(db) => db.name === 'craftsky-private-drafts'
			)
		)
			return [];
		return new Promise((resolve, reject) => {
			const request = indexedDB.open('craftsky-private-drafts', 1);
			request.onsuccess = () => {
				const db = request.result;
				const tx = db.transaction('drafts', 'readonly');
				const read = tx.objectStore('drafts').getAll();
				read.onsuccess = () =>
					resolve(read.result.filter((row) => !row.deleted));
				tx.oncomplete = () => db.close();
				tx.onerror = () => reject(new Error('Fixture read failed'));
			};
		});
	});
}
test.beforeEach(async ({ context }) => {
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
	await context.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
});
test('typing and discarding never save; explicit saves survive reload and stale tabs cannot overwrite them', async ({
	page,
	context
}) => {
	await page.goto('/compose');
	await editor(page).fill('Not automatically saved');
	// Exercise the removed one-second autosave trigger, not just the immediate state.
	await page.waitForTimeout(1200);
	expect(await localDrafts(page)).toEqual([]);
	await page.getByRole('link', { name: 'Close editor', exact: true }).click();
	await page
		.getByRole('dialog')
		.getByRole('button', { name: 'Discard', exact: true })
		.click();
	await expect(page).toHaveURL(/\/feed$/);
	expect(await localDrafts(page)).toEqual([]);
	await page.goto('/compose');
	await editor(page).fill('Explicit saved draft');
	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page).toHaveURL(/\/feed$/);
	await page.goto('/drafts');
	await page.getByRole('link', { name: 'Continue editing' }).click();
	const other = await context.newPage();
	await other.goto(page.url());
	await expect(editor(other)).toHaveValue('Explicit saved draft');
	await editor(page).fill('First tab saved');
	await page.getByRole('button', { name: 'Save changes', exact: true }).click();
	await expect(page).toHaveURL(/\/drafts$/);
	await editor(other).fill('Second tab unsaved');
	await other
		.getByRole('button', { name: 'Save changes', exact: true })
		.click();
	await expect(other.getByRole('alert')).toContainText(
		'changed in another tab'
	);
	await expect(editor(other)).toHaveValue('Second tab unsaved');
	expect((await localDrafts(page)).map((row) => row.text)).toEqual([
		'First tab saved'
	]);
	await other.close();
});
test('chat typing is unsaved until Save draft is chosen', async ({ page }) => {
	await fetch('http://127.0.0.1:19847/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ visual: true })
	});
	await page.goto('/messages/conversations/visual-accepted');
	const text = page.getByRole('textbox', { name: 'Message', exact: true });
	await text.fill('PRIVATE_CHAT_DRAFT_CANARY');
	expect(await page.evaluate(() => JSON.stringify(localStorage))).not.toContain(
		'PRIVATE_CHAT_DRAFT_CANARY'
	);
	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(
		page.getByRole('button', { name: 'Save draft', exact: true })
	).toHaveCount(0);
	await page.reload();
	await expect(text).toHaveValue('PRIVATE_CHAT_DRAFT_CANARY');
	await text.fill('Unsaved replacement');
	await page.getByRole('link', { name: 'Back to messages' }).click();
	const prompt = page.getByRole('dialog', { name: 'Discard unsaved message?' });
	await expect(prompt).toBeVisible();
	await prompt.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(page).toHaveURL(/\/messages$/);
	await page.goto('/messages/conversations/visual-accepted');
	await expect(text).toHaveValue('PRIVATE_CHAT_DRAFT_CANARY');
});
test('stopping a pending publication retains the frozen request and retry reuses it', async ({
	page
}) => {
	const attempts: { key: string | undefined; body: string | null }[] = [];
	let release!: () => void;
	const held = new Promise<void>((resolve) => {
		release = resolve;
	});
	await page.route('**/api/appview/v1/posts', async (route) => {
		attempts.push({
			key: route.request().headers()['idempotency-key'],
			body: route.request().postData()
		});
		if (attempts.length === 1) await held;
		await route
			.fulfill({ status: 503, json: { error: 'unavailable' } })
			.catch(() => {});
	});
	await page.goto('/compose');
	await editor(page).fill('Frozen original');
	await page.getByRole('button', { name: 'Post', exact: true }).click();
	await expect.poll(() => attempts.length).toBe(1);
	await page.getByRole('button', { name: 'Stop waiting' }).click();
	await expect(
		page.getByRole('heading', { name: 'Waiting for confirmation' })
	).toBeVisible();
	const recovery = await localDrafts(page);
	expect(recovery[0].pending?.key).toBe(attempts[0].key);
	release();
	await page
		.getByRole('button', { name: 'Check the same action', exact: true })
		.click();
	await expect.poll(() => attempts.length).toBe(2);
	expect(attempts[1]).toEqual(attempts[0]);
});
test('page titles distinguish routes on direct loads and client navigation', async ({
	page
}) => {
	await page.goto('/feed');
	await expect(page).toHaveTitle('Feed · CraftSky');
	await page
		.getByRole('link', { name: 'Settings', exact: true })
		.first()
		.click();
	await expect(page).toHaveTitle('Settings · CraftSky');
	await page.goto('/settings/privacy');
	await expect(page).toHaveTitle('Privacy settings · CraftSky');
});
test('proxy rejects oversized JSON with a syntactically valid but invalid credential', async ({
	request
}) => {
	const result = await request.post(base + '/api/appview/v1/posts', {
		headers: {
			origin: base,
			cookie:
				'craftsky_account_' +
				alice.id +
				'=' +
				Buffer.from(
					JSON.stringify({
						...alice,
						token: Buffer.alloc(32, 9).toString('base64url')
					})
				).toString('base64url')
		},
		data: 'x'.repeat(1024 * 1024 + 1)
	});
	expect(result.status()).toBe(413);
});

test('enabled subscription actions agree on feed, direct post and signed-in profile pages', async ({
	page
}) => {
	for (const path of [
		'/feed',
		'/posts/did%3Aplc%3Aalice/first',
		'/profiles/did%3Aplc%3Aalice?tab=posts'
	]) {
		await page.goto(path);
		await page
			.getByRole('button', { name: 'Save', exact: true })
			.first()
			.click();
		const dialog = page.getByRole('dialog', { name: 'Save post', exact: true });
		await expect(dialog).toBeVisible();
		await expect(
			dialog.getByRole('button', { name: 'New folder', exact: true })
		).toBeVisible();
		await expect(
			dialog.getByRole('button', { name: 'New folder is coming soon' })
		).toHaveCount(0);
	}
});
