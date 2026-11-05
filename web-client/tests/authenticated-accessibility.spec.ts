import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const account = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
test.beforeEach(async ({ page, context }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'fixture-healthy' } })
	);
	await context.addCookies([
		{
			name: 'craftsky_account_' + account.id,
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

async function audit(page: Page) {
	await page.evaluate(() => document.fonts.ready);
	const results = await new AxeBuilder({ page })
		.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
		.analyze();
	expect(results.violations).toEqual([]);
}

for (const colorScheme of ['light', 'dark'] as const) {
	test(`warning recovery actions have accessible contrast in ${colorScheme}`, async ({
		page
	}) => {
		await page.emulateMedia({ colorScheme });
		await page.route('**/api/appview/v1/feed/timeline**', (route) =>
			route.fulfill({
				status: 503,
				json: { error: 'unavailable', message: 'Please try again.' }
			})
		);
		await page.goto('/feed');
		const warning = page.locator('.snackbar.warning');
		await expect(
			warning.getByRole('button', { name: 'Retry', exact: true })
		).toBeVisible();
		await audit(page);
	});

	test(`composer overlays and destructive confirmation are accessible in ${colorScheme}`, async ({
		page
	}) => {
		await page.emulateMedia({ colorScheme });
		await page.goto('/compose');
		const text = page.getByRole('textbox', {
			name: 'What would you like to share?'
		});
		const photos = page.getByLabel('Add photos or a video');
		await photos.focus();
		await expect(page.locator('.photo-picker')).toHaveCSS(
			'outline-style',
			'solid'
		);
		await text.fill('Accessibility draft');
		const sponsored = page.getByRole('button', { name: 'Sponsored options' });
		await sponsored.focus();
		await page.keyboard.press('Enter');
		await expect(
			page.getByRole('switch', { name: 'Sponsored content' })
		).toBeFocused();
		await audit(page);
		await page.keyboard.press('Tab');
		const closeSponsored = page.getByRole('button', {
			name: 'Close sponsored options'
		});
		await expect(closeSponsored).toBeFocused();
		await expect(closeSponsored).toHaveCSS('clip-path', 'none');
		await page.keyboard.press('Escape');
		await expect(sponsored).toBeFocused();
		await page.getByRole('button', { name: 'Post languages' }).click();
		await audit(page);
		await page.getByRole('menuitem', { name: 'Add language' }).click();
		await expect(
			page.getByRole('searchbox', { name: 'Search languages' })
		).toBeFocused();
		await audit(page);
		await page.keyboard.press('Escape');
		await expect(
			page.getByRole('button', { name: 'Post languages' })
		).toBeFocused();
		await page.getByRole('button', { name: 'Save draft', exact: true }).click();
		await expect(page).toHaveURL(/\/feed$/);
		await page.goto('/drafts');
		const remove = page.getByRole('button', {
			name: 'Delete draft',
			exact: true
		});
		await remove.click();
		const dialog = page.getByRole('dialog', { name: 'Delete draft?' });
		await expect(dialog).toBeVisible();
		await audit(page);
		await page.keyboard.press('Escape');
		await expect(dialog).toHaveCount(0);
		await expect(remove).toBeFocused();
	});
	test(`project selection and filter page support keyboard and accessible names in ${colorScheme}`, async ({
		page
	}) => {
		await page.emulateMedia({ colorScheme });
		await page.goto('/compose/project');
		const craft = page.getByRole('combobox', {
			name: 'Primary craft required'
		});
		await craft.focus();
		await page.keyboard.press('ArrowDown');
		await expect(
			page.getByRole('listbox', { name: 'Primary craft options' })
		).toBeVisible();
		await audit(page);
		await page.keyboard.press('Enter');
		await expect(craft).toBeFocused();
		await page.goto('/feed/filters?craft=knitting');
		await expect(
			page.getByRole('button', { name: 'Back to projects' })
		).toBeFocused();
		await audit(page);
		const color = page.getByRole('searchbox', { name: 'Color' });
		await color.focus();
		await color.fill('Blue');
		await page.keyboard.press('ArrowDown');
		const blue = page.getByRole('checkbox', { name: 'Blue', exact: true });
		await expect(blue).toBeFocused();
		await page.keyboard.press('Space');
		await expect(
			page.getByRole('button', { name: 'Remove Blue from Color' })
		).toBeVisible();
		await expect(color).toBeFocused();
		await color.press('ArrowDown');
		await expect(blue).toBeChecked();
		await audit(page);
		await page.keyboard.press('Escape');
		await expect(color).toBeFocused();
		await expect(page).toHaveURL(/\/feed\/filters/);
	});
}

test('sign-in suggestions and all onboarding steps remain accessible in both themes', async ({
	page
}) => {
	await page.route('https://typeahead.waow.tech/**', (route) =>
		route.fulfill({
			json: {
				actors: [
					{ did: 'did:plc:bob', handle: 'bob.test', displayName: 'Bob Maker' }
				]
			}
		})
	);
	for (const colorScheme of ['light', 'dark'] as const) {
		await page.emulateMedia({ colorScheme });
		await page.goto('/sign-in');
		const handle = page.getByRole('combobox', {
			name: 'Your Atmosphere Handle'
		});
		await handle.fill('bob');
		await expect(page.getByRole('option', { name: /Bob Maker/ })).toBeVisible();
		await audit(page);
		await handle.press('ArrowDown');
		await handle.press('Enter');
		await expect(handle).toHaveValue('bob.test');
		await expect(handle).toBeFocused();
		await fetch('http://127.0.0.1:19847/__test/control', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ owners: { alice: { completed: false } } })
		});
		await page.setViewportSize({ width: 402, height: 874 });
		await page.goto('/feed');
		await expect(
			page.getByRole('heading', { name: 'Make it yours' })
		).toBeVisible();
		await audit(page);
		await page.getByRole('button', { name: 'Next', exact: true }).click();
		await expect(
			page.getByRole('heading', { name: 'What do you make?' })
		).toBeVisible();
		await audit(page);
		await page.getByRole('button', { name: 'Next', exact: true }).click();
		await expect(
			page.getByRole('button', { name: 'Finish', exact: true })
		).toBeDisabled();
		await audit(page);
	}
});
