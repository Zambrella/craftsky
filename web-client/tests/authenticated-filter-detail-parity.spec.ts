import { test, expect } from '@playwright/test';
const base = 'http://127.0.0.1:19846';
const alice = {
	id: '0c10030a-6208-43c2-8048-d3edcc609271',
	did: 'did:plc:alice',
	handle: 'alice.test',
	token: Buffer.alloc(32, 1).toString('base64url')
};
test.beforeEach(async ({ context, page }) => {
	await fetch('http://127.0.0.1:19847/__test/reset');
	await page.route('https://status.craftsky.social/app.json', (route) =>
		route.fulfill({ json: { schemaVersion: 2, revision: 'healthy' } })
	);
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
	await page.route('**/api/appview/v1/projects*', (route) =>
		route.fulfill({ json: { items: [] } })
	);
});
test('native compact filter form uses search, dependent choices and Cancel without applying edits', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto('/feed?craft=knitting');
	const open = () =>
		page.getByRole('button', { name: /^Filter projects/ }).click();
	await open();
	const dialog = page.getByRole('region', {
		name: 'Filter Knitting projects',
		exact: true
	});
	await expect(dialog).toBeVisible();
	expect(
		await dialog.evaluate((x) => ({
			top: x.getBoundingClientRect().top,
			height: x.clientHeight,
			width: x.clientWidth
		}))
	).toEqual({ top: 0, height: 874, width: 402 });
	const type = dialog.getByRole('searchbox', {
		name: 'Project type',
		exact: true
	});
	const subtype = dialog.getByRole('searchbox', {
		name: 'Project subtype',
		exact: true
	});
	await expect(subtype).toBeDisabled();
	await expect(type).toHaveCSS('border-top-width', '0px');
	await expect(type).toHaveCSS('box-shadow', 'none');
	await type.fill('gar');
	const choices = dialog.getByRole('group', {
		name: 'Project type options',
		exact: true
	});
	await expect(choices.getByRole('checkbox')).toHaveCount(1);
	await page.screenshot({
		path: test.info().outputPath('native-filter-picker.png')
	});
	// Click the option label, as a person does, rather than the checkbox itself.
	await choices.locator('label').filter({ hasText: 'Garment' }).click();
	await expect(choices).toHaveCount(0);
	await expect(type).toBeFocused();
	await expect(type).toHaveValue('');
	await expect(subtype).toBeEnabled();
	await subtype.fill('sweater');
	await expect(choices).toHaveCount(0);
	await subtype.press('Enter');
	await expect(
		dialog.getByRole('button', {
			name: 'Remove Sweater from Project subtype',
			exact: true
		})
	).toBeVisible();
	await expect(dialog).toBeVisible();
	await dialog
		.getByRole('button', {
			name: 'Remove Garment from Project type',
			exact: true
		})
		.click();
	await expect(subtype).toBeDisabled();
	await expect(
		dialog.getByRole('button', {
			name: 'Remove Sweater from Project subtype',
			exact: true
		})
	).toHaveCount(0);
	const color = dialog.getByRole('searchbox', { name: 'Color', exact: true });
	await color.fill('blue');
	await dialog.getByRole('checkbox', { name: 'Blue', exact: true }).click();
	await color.press('Enter');
	await expect(dialog).toBeVisible();
	expect(new URL(page.url()).searchParams.has('color')).toBe(false);
	await dialog
		.getByRole('button', { name: 'Back to projects', exact: true })
		.click();
	await open();
	await expect(
		dialog.getByRole('button', { name: 'Remove Blue from Color', exact: true })
	).toHaveCount(0);
	await expect(
		dialog.getByRole('searchbox', { name: 'Project type', exact: true })
	).toHaveValue('');
	expect(
		await page.evaluate(
			() => document.documentElement.scrollWidth <= innerWidth
		)
	).toBe(true);
});
test('native selection limit and Clear all apply cleared filters and return to the craft feed', async ({
	page
}) => {
	await page.goto(
		'/feed?craft=knitting&sort=popular&color=blue&selfDrafted=true'
	);
	await page.getByRole('button', { name: /^Filter projects/ }).click();
	const dialog = page.getByRole('region', {
		name: 'Filter Knitting projects',
		exact: true
	});
	const color = dialog.getByRole('searchbox', { name: 'Color', exact: true });
	await color.click();
	const choices = dialog.getByRole('group', {
		name: 'Color options',
		exact: true
	});
	for (let i = 0; i < 9; i++) {
		await color.click();
		await choices.locator('input:not(:checked):not(:disabled)').first().click();
	}
	await color.click();
	await expect(choices.locator('input:checked')).toHaveCount(10);
	await expect(
		choices.locator('input:not(:checked):not(:disabled)')
	).toHaveCount(0);
	await color.press('Escape');
	await dialog.getByRole('button', { name: 'Clear all', exact: true }).click();
	await expect(dialog).not.toBeVisible();
	await expect
		.poll(() => new URL(page.url()).searchParams.has('color'))
		.toBe(false);
	const query = new URL(page.url()).searchParams;
	expect(query.get('craft')).toBe('knitting');
	expect(query.get('sort')).toBe('popular');
	expect(query.has('color')).toBe(false);
	expect(query.has('selfDrafted')).toBe(false);
});

test('native sort sheets preserve craft filters and move to a pointer menu on wide layouts', async ({
	page
}) => {
	await page.setViewportSize({ width: 402, height: 874 });
	await page.goto('/feed?craft=knitting&status=finished');
	await page.getByLabel('Sort projects: Newest', { exact: true }).click();
	const menu = page.locator('.choice-menu[open] .menu-items');
	await expect(menu).toHaveCSS('position', 'fixed');
	await menu.getByRole('link', { name: 'Popular', exact: true }).click();
	await expect(page).toHaveURL(/sort=popular/);
	expect(new URL(page.url()).searchParams.get('craft')).toBe('knitting');
	expect(new URL(page.url()).searchParams.get('status')).toBe('finished');
	await page.setViewportSize({ width: 1280, height: 900 });
	await page.getByLabel('Sort projects: Popular', { exact: true }).click();
	await expect(menu).toHaveCSS('border-top-left-radius', '14px');
	expect(
		await menu.evaluate((el) => el.getBoundingClientRect().width)
	).toBeLessThanOrEqual(280);
	await expect(page.locator('html')).not.toHaveCSS('overflow', 'hidden');
	await page.keyboard.press('Escape');
	await expect(page.locator('.choice-menu[open]')).toHaveCount(0);
});

test('project filters are a content-column route with Back and Forward and no modal history after Apply', async ({
	page
}) => {
	await page.goto('/feed');
	await page
		.getByRole('navigation', { name: 'Home feeds' })
		.getByRole('link', { name: 'Knitting', exact: true })
		.click();
	await expect(page).toHaveURL(/craft=.*knitting/);
	const origin = page.url();
	await page
		.getByRole('button', { name: 'Filter projects', exact: true })
		.click();
	await expect(page).toHaveURL(/\/feed\/filters\?/);
	const filters = page.getByRole('region', {
		name: 'Filter Knitting projects',
		exact: true
	});
	await expect(filters).toBeVisible();
	await expect(page.locator('dialog[open]')).toHaveCount(0);
	const main = (await page.locator('.app-main').boundingBox())!;
	const bounds = (await filters.boundingBox())!;
	expect(Math.abs(main.x - bounds.x)).toBeLessThan(2);
	expect(Math.abs(main.width - bounds.width)).toBeLessThan(2);
	await filters
		.getByRole('switch', { name: 'Self drafted', exact: true })
		.check();
	await page.goBack();
	await expect(page).toHaveURL(origin);
	await page.goForward();
	await expect(filters).toBeVisible();
	await filters
		.getByRole('switch', { name: 'Self drafted', exact: true })
		.check();
	await filters
		.getByRole('button', { name: 'Apply filters', exact: true })
		.click();
	await expect(page).toHaveURL(/selfDrafted=true/);
	await expect(filters).toHaveCount(0);
	await page.goBack();
	await expect(page).toHaveURL(/\/feed$/);
});
