import { test, expect } from '@playwright/test';
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
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
	// These tests exercise device-private drafts only; publishing is never permitted.
	await page.route('**/api/appview/v1/posts', async (route) => {
		if (route.request().method() === 'POST')
			throw new Error('Project detail tests must not publish');
		await route.continue();
	});
});

test('tab strip drags without activating a tab and still permits clicks', async ({
	page
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.goto('/app/home');
	const tabs = page.getByRole('navigation', { name: 'Home feeds' });
	await expect(tabs).toBeVisible();
	const rect = (await tabs.boundingBox())!;
	const before = page.url();
	await page.mouse.move(rect.x + rect.width - 35, rect.y + rect.height / 2);
	await page.mouse.down();
	await page.mouse.move(rect.x + 20, rect.y + rect.height / 2, { steps: 15 });
	await page.mouse.up();
	expect(await tabs.evaluate((node) => node.scrollLeft)).toBeGreaterThan(20);
	expect(page.url()).toBe(before);
	await tabs.getByRole('link').last().click();
	await expect(page).not.toHaveURL(before);
});

test('facets paint in primary colour and suggestions follow the caret without keyboard hints', async ({
	page
}) => {
	await page.route('**/api/appview/v1/facets/hashtags?*', (route) =>
		route.fulfill({ json: { items: [{ tag: 'sewing', postCount: 12 }] } })
	);
	await page.goto('/app/compose/standard');
	const field = page.getByRole('textbox', {
		name: 'What would you like to share?'
	});
	await field.fill('Hello @bob.test #sew');
	const options = page.getByRole('listbox', { name: 'Hashtag suggestions' });
	await expect(options).toBeVisible();
	await expect(
		page.getByText('Use ↑ ↓ and Enter to choose, or Escape to close.')
	).toHaveCount(0);
	const bounds = (await field.boundingBox())!;
	const popup = (await options.boundingBox())!;
	expect(popup.y).toBeLessThan(bounds.y + bounds.height - 10);
	const paint = page.locator('.facet-text-paint');
	await expect(paint).toContainText('Hello @bob.test #sew');
	expect(
		await paint
			.locator('span')
			.nth(1)
			.evaluate((node) => getComputedStyle(node).color)
	).toBe('rgb(21, 53, 214)');
	await field.press('Escape');
	await field.press('End');
	await field.press('Enter');
	await field.pressSequentially('A second line');
	await expect(paint).toContainText('A second line');
	await page.screenshot({ path: test.info().outputPath('facet-input.png') });
	await page.getByRole('button', { name: 'Sponsored options' }).click();
	const toggle = page.getByRole('switch', { name: 'Sponsored content' });
	expect(
		await toggle.evaluate((node) => getComputedStyle(node, '::before').content)
	).toBe('none');
	await toggle.check();
	await page.getByRole('button', { name: 'Sponsored options' }).click();
	await expect(toggle).toBeChecked();
	await page
		.getByRole('dialog', { name: 'Sponsored', exact: true })
		.screenshot({ path: test.info().outputPath('sponsored.png') });
});

test('primary craft required note is inline and field titles do not highlight or activate controls', async ({
	page
}) => {
	await page.goto('/app/compose/project');
	const field = page.getByRole('combobox', { name: 'Primary craft' });
	const title = page
		.locator('.native-single-select')
		.filter({ has: field })
		.locator('label');
	const rect = (await title.boundingBox())!;
	const note = (await title.locator('small').boundingBox())!;
	expect(Math.abs(note.y - rect.y)).toBeLessThan(10);
	await page.mouse.move(rect.x + 10, rect.y + rect.height / 2);
	expect(
		await title.evaluate((node) => getComputedStyle(node).pointerEvents)
	).toBe('none');
	await page.mouse.click(rect.x + 10, rect.y + rect.height / 2);
	await expect(field).not.toBeFocused();
	await field.click();
	await expect(field).toBeFocused();
});

test('sign-in discovers accounts with keyboard selection and keeps manual sign-in available', async ({
	page,
	context
}) => {
	await context.clearCookies();
	await page.route('https://typeahead.waow.tech/**', async (route) => {
		expect(route.request().headers().authorization).toBeUndefined();
		expect(route.request().headers().cookie).toBeUndefined();
		await route.fulfill({
			json: {
				actors: [
					{
						did: 'did:plc:bob',
						handle: 'bob.test',
						displayName: 'Bob Maker',
						avatar: 'https://example.test/avatar.png'
					},
					{ handle: 'handle.invalid' },
					{ handle: 'bob.test' }
				]
			}
		});
	});
	await page.goto('/sign-in');
	const input = page.getByRole('combobox', { name: 'Your Atmosphere Handle' });
	await input.fill('bob');
	await expect(page.getByRole('option')).toHaveCount(1);
	await expect(page.getByRole('option')).toContainText('@bob.test');
	expect(
		await input.evaluate((node) => getComputedStyle(node).outlineStyle)
	).toBe('none');
	await input.press('ArrowDown');
	await input.press('Enter');
	await expect(input).toHaveValue('bob.test');
	await expect(page.getByRole('option')).toHaveCount(0);
	await expect(
		page.getByRole('button', { name: 'Sign in', exact: true })
	).toBeEnabled();
	await page.screenshot({ path: test.info().outputPath('sign-in.png') });
	await page.unroute('https://typeahead.waow.tech/**');
	await page.route('https://typeahead.waow.tech/**', (route) => route.abort());
	await input.fill('custom.example');
	await expect(
		page.getByRole('button', { name: 'Sign in', exact: true })
	).toBeEnabled();
});

test('onboarding keeps one editor across steps, uses fixed actions and retains unfinished changes on Back', async ({
	page
}) => {
	await fetch('http://127.0.0.1:19847/__test/control', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ owners: { alice: { completed: false } } })
	});
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto('/app/home');
	await expect(
		page.getByRole('heading', { name: 'Make it yours' })
	).toBeVisible();
	await expect(page.getByText('Signed in as @alice.test')).toBeVisible();
	await page.screenshot({
		path: test.info().outputPath('onboarding-profile.png')
	});
	const next = page.getByRole('button', { name: 'Next', exact: true });
	await expect(next).toBeInViewport();
	await next.click();
	await expect(
		page.getByRole('heading', { name: 'What do you make?' })
	).toBeVisible();
	await page.screenshot({
		path: test.info().outputPath('onboarding-crafts.png')
	});
	await next.click();
	await expect(
		page.getByRole('button', { name: 'Finish', exact: true })
	).toBeDisabled();
	await page.screenshot({
		path: test.info().outputPath('onboarding-guidelines.png')
	});
	await page.getByRole('button', { name: 'Previous onboarding step' }).click();
	await page.getByRole('checkbox', { name: 'Crochet', exact: true }).check();
	await page.getByRole('button', { name: 'Previous onboarding step' }).click();
	await expect(page.getByRole('button', { name: 'Save & next' })).toBeVisible();
	// A single editor holds both pages' edits until they are successfully saved.
	await page
		.getByRole('textbox', { name: 'Display name', exact: false })
		.fill('Alice Updated');
	await page.getByRole('button', { name: 'Save & next' }).click();
	await expect(
		page.getByRole('heading', { name: 'What do you make?' })
	).toBeVisible();
	await expect(
		page.getByRole('checkbox', { name: 'Crochet', exact: true })
	).toBeChecked();
	await page.getByRole('button', { name: 'Previous onboarding step' }).click();
	await expect(
		page.getByRole('textbox', { name: 'Display name', exact: false })
	).toHaveValue('Alice Updated');
});

test('draft deletion uses the shared branded confirmation in both themes', async ({
	page
}) => {
	await page.goto('/app/compose');
	await page
		.getByRole('textbox', { name: 'What would you like to share?' })
		.fill('Draft dialog preview');
	await page.getByRole('button', { name: 'Save draft', exact: true }).click();
	await expect(page).toHaveURL(/\/app\/home$/);
	await page.getByRole('link', { name: 'Drafts', exact: true }).click();
	await page.getByRole('button', { name: 'Delete draft', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Delete draft?' });
	await expect(dialog).toBeVisible();
	await expect(
		dialog.getByRole('button', { name: 'Delete', exact: true })
	).toBeVisible();
	await dialog.screenshot({
		path: test.info().outputPath('delete-draft-light.png')
	});
	await page.evaluate(() => {
		document.documentElement.dataset.theme = 'dark';
	});
	await dialog.screenshot({
		path: test.info().outputPath('delete-draft-dark.png')
	});
	await dialog.getByRole('button', { name: 'Cancel' }).click();
	await expect(
		page.getByText('Draft dialog preview', { exact: true })
	).toBeVisible();
});
